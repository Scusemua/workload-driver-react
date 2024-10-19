package handlers

import (
	"context"
	"errors"
	"fmt"
	"github.com/petermattis/goid"
	"net"
	"net/http"
	"os"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hashicorp/yamux"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	gateway "github.com/scusemua/workload-driver-react/m/v2/internal/server/api/proto"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

// RegistrationCompleteCallback is a callback for the handlers.ClusterDashboardHandler of the serverImpl to execute
// once it establishes its two-way, bidirectional gRPC connection with the Cluster Gateway.
//
// This callback is primarily used to instruct the serverImpl's
// nodeHandler to create its internal node handler, depending on the domain.NodeType received during the gRPC
// registration process.
//
// It is important that this callback can be executed multiple times, in case the node type changes for whatever reason.
// For example, a gRPC connection with the cluster may be established at one point, and the cluster will be in Docker
// mode at that point. Later on, the connection may be lost, and the cluster is restarted in Kubernetes mode, while
// the Dashboard backend server is not restarted. This will prompt a reconfiguration of the NodeHttpHandler's
// domain.NodeType and thus its internal node handler. Once that reconfiguration is completed, the specified
// RegistrationCompleteCallback will be re-triggered.
type RegistrationCompleteCallback func(nodeType domain.NodeType, rpcHandler *ClusterDashboardHandler)

var (
	ErrFailedToConnect           = errors.New("a connection to the Gateway could not be established within the configured timeout")
	ErrProvisionerNotInitialized = errors.New("provisioner is not initialized")
	ErrConcurrentSetupOperations = errors.New("there is already 'setup RPC resources' operation taking place")

	sig = make(chan os.Signal, 1)
)

type notificationCallback func(notification *gateway.Notification)

// ClusterDashboardHandler is a type of handler that issues HTTP requests to the backend.
type ClusterDashboardHandler struct {
	// If this is equal to 1, then the gRPC resources are in the process of being setup.
	// In this case, additional attempts to reconnect should not be performed.
	setupInProgress int32

	gateway.DistributedClusterClient // gRPC client to the Cluster Gateway.
	gateway.UnimplementedClusterDashboardServer

	// deploymentMode indicates whether the Cluster is running in Kubernetes mode or Docker mode.
	// Valid options include "local", "docker", and "kubernetes".
	deploymentMode string

	// schedulingPolicy indicates the scheduling policy that the Cluster Gateway has been configured to use.
	schedulingPolicy string

	// numReplicas refers to the number of replicas that each Jupyter kernel is configured to have.
	numReplicas int32

	logger        *zap.Logger
	sugaredLogger *zap.SugaredLogger

	clusterDashboardHandlerPort int

	// Called in its own Goroutine when the ErrorOccurred RPC is called by the ClusterGateway to us.
	notificationCallback notificationCallback

	connected atomic.Int32

	srv *grpc.Server

	gatewayAddress string // Address that the Cluster Gateway's gRPC server is listening on.

	// postRegistrationCallback is a callback for the handlers.ClusterDashboardHandler of the serverImpl to execute
	// once it establishes its two-way, bidirectional gRPC connection with the Cluster Gateway.
	//
	// This callback is primarily used to instruct the serverImpl's
	// nodeHandler to create its internal node handler, depending on the domain.NodeType received during the gRPC
	// registration process.
	//
	// It is important that this callback can be executed multiple times, in case the node type changes for whatever reason.
	// For example, a gRPC connection with the cluster may be established at one point, and the cluster will be in Docker
	// mode at that point. Later on, the connection may be lost, and the cluster is restarted in Kubernetes mode, while
	// the Dashboard backend server is not restarted. This will prompt a reconfiguration of the NodeHttpHandler's
	// domain.NodeType and thus its internal node handler. Once that reconfiguration is completed, the specified
	// RegistrationCompleteCallback will be re-triggered.
	postRegistrationCallback RegistrationCompleteCallback
}

func NewClusterDashboardHandler(
	opts *domain.Configuration,
	shouldConnect bool,
	notificationCallback notificationCallback,
	postRegistrationCallback RegistrationCompleteCallback) *ClusterDashboardHandler {

	if opts == nil {
		panic("opts cannot be nil.")
	}

	if notificationCallback == nil {
		panic("notificationCallback cannot be nil.")
	}

	if postRegistrationCallback == nil {
		panic("postRegistrationCallback cannot be nil.")
	}

	handler := &ClusterDashboardHandler{
		clusterDashboardHandlerPort: opts.ClusterDashboardHandlerPort,
		gatewayAddress:              opts.GatewayAddress,
		setupInProgress:             0,
		notificationCallback:        notificationCallback,
		postRegistrationCallback:    postRegistrationCallback,
	}
	handler.connected.Store(0)

	var err error
	handler.logger, err = zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	handler.sugaredLogger = handler.logger.Sugar()

	if shouldConnect {
		go func() {
			err = handler.setupRpcResources(opts.GatewayAddress)
			if err != nil {
				handler.logger.Error("Failed to dial gRPC Cluster Gateway.", zap.Error(err))
				panic(err)
			}
		}()
	}

	return handler
}

// NumReplicas returns
func (h *ClusterDashboardHandler) NumReplicas() int32 {
	return h.numReplicas
}

// SchedulingPolicy returns the scheduling policy that the Cluster Gateway has been configured to use.
func (h *ClusterDashboardHandler) SchedulingPolicy() string {
	return h.schedulingPolicy
}

// DeploymentMode returns the deployment mode configured for the distributed notebook cluster.
// Valid options include "local", "docker", and "kubernetes".
func (h *ClusterDashboardHandler) DeploymentMode() string {
	return h.deploymentMode
}

// HandleConnectionError should be called by another handler if a gRPC error is encountered.
// This will attempt to restart/recreate the gRPC server + client encapsulated by ClusterDashboardHandler.
//
// This does not need to be called in its own goroutine; this function spawns a goroutine itself.
func (h *ClusterDashboardHandler) HandleConnectionError() {
	if h.gatewayAddress == "" {
		// The gatewayAddress field is set after the initial connection attempt succeeds.
		h.logger.Error("Cannot attempt to recreate gRPC resources as the gateway address was not specified when the Cluster Dashboard Handler was created.")
		return
	}

	go func() {
		gid := goid.Get()
		h.sugaredLogger.Debug("Handling connection error: attempting to reconnect to Cluster Gateway.", zap.Int64("gid", gid))
		st := time.Now()
		err := h.setupRpcResources(h.gatewayAddress)
		if err != nil {
			domain.LogErrorWithoutStacktrace(h.logger, "Failed to reestablish connection with Cluster Gateway in HandleConnectionError.", zap.Error(err), zap.Duration("time_elapsed", time.Since(st)), zap.Int64("gid", gid))
		} else {
			h.logger.Debug("Successfully reestablished connection with Cluster Gateway in HandleConnectionError.", zap.Duration("time_elapsed", time.Since(st)), zap.Int64("gid", gid))
		}
	}()
}

// SendNotification is called by the Cluster Gateway targeting us. It is used to publish notifications that should
// ultimately be pushed to the frontend to be displayed to the user.
func (h *ClusterDashboardHandler) SendNotification(_ context.Context, notification *gateway.Notification) (*gateway.Void, error) {
	if notification.NotificationType == int32(domain.ErrorNotification) {
		h.logger.Warn("Notified of error that occurred within Cluster.", zap.String("error-name", notification.Title), zap.String("error-message", notification.Message))
	} else {
		h.logger.Debug("Received notification from Cluster.", zap.String("notification-name", notification.Title), zap.String("notification-message", notification.Message))
	}

	go h.notificationCallback(notification)

	return &gateway.Void{}, nil
}

// setupRpcResources establishes the two-way, bidirectional gRPC connection between the Cluster Dashboard backend
// server (us) and the Cluster Gateway component.
//
// This involves creating both a gRPC client and a gRPC server (I think?)
func (h *ClusterDashboardHandler) setupRpcResources(gatewayAddress string) error {
	gid := goid.Get()
	swapped := atomic.CompareAndSwapInt32(&h.setupInProgress, 0, 1)
	if !swapped {
		h.logger.Debug("There is already 'setup RPC resources' operation taking place.")
		return ErrConcurrentSetupOperations
	}

	h.logger.Debug("Dialing Cluster Gateway now.", zap.String("gateway-address", gatewayAddress), zap.Int64("gid", gid))

	gOpts := []grpc.ServerOption{
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Timeout: 120 * time.Second,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			PermitWithoutStream: true,
		}),
	}

	if h.srv != nil {
		h.logger.Warn("Cluster Dashboard Handler already has an existing server. Attempting to stop that server first.", zap.Int64("gid", gid))

		ctx, cancel := context.WithTimeout(context.TODO(), time.Second*30)
		defer cancel()
		doneChan := make(chan interface{})

		go func(doneChan chan interface{}) {
			h.logger.Debug("Attempting to stop existing gRPC server gracefully.", zap.Int64("gid", goid.Get()))
			h.srv.GracefulStop()
			select {
			case doneChan <- struct{}{}:
				return
			default:
				return
			}
		}(doneChan)

		select {
		case <-doneChan:
			{
				h.logger.Debug("Successfully stopped existing gRPC server gracefully.", zap.Int64("gid", gid))
			}
		case <-ctx.Done():
			{
				h.logger.Warn("Failed to stop existing gRPC server gracefully. Forcefully stopping server now.", zap.Int64("gid", gid))
				h.srv.Stop()
				h.logger.Debug("Successfully stopped existing gRPC server forcefully.", zap.Int64("gid", gid))
			}
		}
	}

	h.srv = grpc.NewServer(gOpts...)
	gateway.RegisterClusterDashboardServer(h.srv, h)

	// Initialize gRPC listener
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", h.clusterDashboardHandlerPort))
	if err != nil {
		h.logger.Error("Failed to listen.", zap.Error(err), zap.Int64("gid", gid))
		h.exitSetup()
		return err
	}
	// defer lis.Close()

	h.logger.Info("ClusterDashboardHandler listening for gRPC.", zap.Any("address", lis.Addr()), zap.Int64("gid", gid))

	start := time.Now()
	connectionTimeout := time.Second * 30 // How long each individual connection attempt should last before timing-out.
	totalTimeout := time.Minute * 5       // How long to keep trying to connect over-and-over before completely giving up and panicking.
	var connectedToGateway = false
	var numAttempts = 1
	var gatewayConn net.Conn
	for !connectedToGateway && time.Since(start) < (totalTimeout) {
		h.sugaredLogger.Debugf("[gid=%d] Attempt #%d to connect to Gateway at %s. Connection timeout: %v. Time elapsed: %v.", gid, numAttempts, gatewayAddress, connectionTimeout, time.Since(start))
		gatewayConn, err = net.DialTimeout("tcp", gatewayAddress, connectionTimeout)

		if err != nil {
			h.sugaredLogger.Warnf("[gid=%d] Failed to connect to provisioner at %s on attempt #%d: %v. Time elapsed: %v.", gid, gatewayAddress, numAttempts, err, time.Since(start))
			numAttempts += 1
			time.Sleep(time.Second * 3)
		} else {
			connectedToGateway = true
			h.logger.Debug("Successfully connected to Gateway.", zap.String("gateway-address", gatewayAddress), zap.Duration("time-elapsed", time.Since(start)), zap.Int64("gid", gid))
		}
	}

	if !connectedToGateway {
		h.sugaredLogger.Errorf("[gid=%d] Failed to connect to Gateway after %d attempts. Time elapsed: %v.", gid, numAttempts, time.Since(start))
		h.exitSetup()
		h.connected.Store(0)
		return ErrFailedToConnect
	}

	// Initialize provisioner and wait for ready.
	provisioner, err := newConnectionProvisioner(gatewayConn)
	if err != nil {
		h.logger.Error("Failed to initialize the Connection Provisioner.",
			zap.String("local_address", gatewayConn.LocalAddr().String()),
			zap.String("remote_address", gatewayConn.RemoteAddr().String()),
			zap.Error(err), zap.Int64("gid", gid))
		h.exitSetup()
		h.connected.Store(0)
		return err
	}

	// Wait for reverse connection
	go func() {
		newGid := goid.Get()
		defer h.finalize(true)
		numTries := 0
		maxNumAttempts := 5
		for numTries < maxNumAttempts {
			provisioner.logger.Debug("Trying to connect.", zap.Int("attempt-number", numTries+1), zap.Int64("gid", newGid))
			if serveError := h.srv.Serve(provisioner); serveError != nil {
				domain.LogErrorWithoutStacktrace(provisioner.logger, "Temporary failure in serving reverse connection. Will retry if not out of attempts...",
					zap.Int("attempt-number", numTries+1), zap.Error(serveError), zap.Int64("gid", newGid))
				numTries += 1

				if numTries < 3 {
					time.Sleep((time.Millisecond * 1000) * time.Duration(numTries))
					continue
				} else {
					domain.LogErrorWithoutStacktrace(provisioner.logger, "Failed to serve reverse connection. Aborting.",
						zap.Int("num_attempts", maxNumAttempts), zap.Int64("gid", newGid))
					provisioner.failedToConnect <- serveError
					return
				}
			}
		}

	}()

	select {
	case <-provisioner.Ready():
		{
			h.logger.Debug("Provisioner connected successfully and is now ready.", zap.Int64("gid", gid))
		}
	case err = <-provisioner.FailedToConnect():
		{
			h.sugaredLogger.Errorf("[gid=%d] Provisioner failed to connect successfully. Aborting. Last error: %v.", gid, err)
			h.exitSetup()
			h.connected.Store(0)
			return err
		}
	}

	registrationResponse, err := provisioner.Validate()
	if err != nil {
		h.exitSetup()
		h.logger.DPanic("Failed to validate reverse provisioner connection.", zap.Error(err), zap.Int64("gid", gid))
		h.connected.Store(0)
		return err
	}

	h.DistributedClusterClient = provisioner

	h.numReplicas = registrationResponse.NumReplicas
	h.deploymentMode = registrationResponse.DeploymentMode
	h.schedulingPolicy = registrationResponse.SchedulingPolicy

	h.logger.Info("Connected to Cluster Gateway.", zap.Any("remote-address", gatewayConn.RemoteAddr()), zap.Int64("gid", gid))

	// Start gRPC server
	go func() {
		defer h.finalize(true)
		if err := h.srv.Serve(lis); err != nil {
			h.logger.DPanic("Failed to serve regular connection.", zap.Error(err), zap.Int64("gid", gid))
		}
	}()

	var nodeType domain.NodeType
	if h.deploymentMode == "docker-compose" {
		nodeType = domain.VirtualDockerNodeType
	} else if h.deploymentMode == "docker-swarm" {
		nodeType = domain.DockerSwarmNodeType
	} else if h.deploymentMode == "kubernetes" {
		nodeType = domain.KubernetesNodeType
	} else {
		h.sugaredLogger.DPanicf("[gid=%d] Unsupported deployment mode received during gRPC registration procedure: \"%s\"", gid, h.deploymentMode)
	}

	h.postRegistrationCallback(nodeType, h)

	h.exitSetup()

	h.connected.Store(1)
	return nil
}

// Swap the flag back to 0. Panic if it is already 0.
func (h *ClusterDashboardHandler) exitSetup() {
	swapped := atomic.CompareAndSwapInt32(&h.setupInProgress, 1, 0)
	if !swapped {
		panic("'setupInProgress' flag was not set to 1")
	}
}

// WriteError writes an error back to the client. This is called by the Cluster Gateway.
func (h *ClusterDashboardHandler) WriteError(c *gin.Context, errorMessage string) {
	// Write error back to front-end.
	_ = c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("could not handle request: %s", errorMessage))
}

func (h *ClusterDashboardHandler) finalize(fix bool) {
	if !fix {
		return
	}

	if err := recover(); err != nil {
		h.logger.Error("Fatal error.", zap.Any("error", err))
	}

	sig <- syscall.SIGINT
}

// ConnectedToGateway returns true if the ClusterDashboardHandler is currently connected.
func (h *ClusterDashboardHandler) ConnectedToGateway() bool {
	return h.connected.Load() > 0
}

// connectionProvisioner is used to establish a 2-way (bidirectional) gRPC connection between
// the Cluster Dashboard backend server and the Cluster Gateway component.
type connectionProvisioner struct {
	net.Listener
	gateway.DistributedClusterClient

	ready           chan struct{}
	failedToConnect chan error
	logger          *zap.Logger
	sugaredLogger   *zap.SugaredLogger
}

// newConnectionProvisioner creates a new connectionProvisioner struct and returns a pointer to it.
func newConnectionProvisioner(conn net.Conn) (*connectionProvisioner, error) {
	// Initialize yamux session for bidirectional gRPC calls
	// At host scheduler side, a connection replacement first made, then we wait for reverse connection by implementing net.Listener
	srvSession, err := yamux.Server(conn, yamux.DefaultConfig())
	if err != nil {
		return nil, err
	}

	provisioner := &connectionProvisioner{
		Listener:        srvSession,
		ready:           make(chan struct{}),
		failedToConnect: make(chan error),
	}

	provisioner.logger, err = zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	provisioner.sugaredLogger = provisioner.logger.Sugar()

	gid := goid.Get()
	provisioner.logger.Debug("Created new Connection Provisioner.",
		zap.String("local_address", conn.LocalAddr().String()),
		zap.String("remote_address", conn.RemoteAddr().String()),
		zap.Int64("gid", gid))

	// Initialize the gRPC client
	if err = provisioner.InitClient(srvSession); err != nil {
		return nil, err
	}

	return provisioner, nil
}

// Ready returns a channel that is closed when the gRPC client is initialized
func (p *connectionProvisioner) Ready() <-chan struct{} {
	return p.ready
}

// FailedToConnect returns a channel that is used to communicate that the Provisioner could not connect successfully.
func (p *connectionProvisioner) FailedToConnect() <-chan error {
	return p.failedToConnect
}

// Accept overrides the default Accept method and initializes the gRPC client
func (p *connectionProvisioner) Accept() (conn net.Conn, err error) {
	conn, err = p.Listener.Accept()
	if err != nil {
		domain.LogErrorWithoutStacktrace(p.logger, "Failed to accept connection.", zap.Error(err))
		return nil, err
	}

	gid := goid.Get()
	p.logger.Debug("Accepted remote connection.",
		zap.String("local_address", conn.LocalAddr().String()),
		zap.String("remote_address", conn.RemoteAddr().String()),
		zap.Int64("gid", gid))

	// Notify possible blocking caller that the gRPC client is initialized
	select {
	case p.ready <- struct{}{}:
		p.logger.Info("Connection Provisioner is ready to rock and roll.", zap.Int64("gid", gid))
	default:
		p.logger.Warn("Duplicate reverse provisioner connection detected. This is, at best, unexpected.", zap.Int64("gid", gid))
	}

	return conn, nil
}

// InitClient initializes the gRPC client
func (p *connectionProvisioner) InitClient(session *yamux.Session) error {
	// Dial to create a gRPC connection with dummy dialer.
	gConn, err := grpc.Dial(":0",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
			gid := goid.Get()
			conn, err := session.Open()
			if err != nil {
				domain.LogErrorWithoutStacktrace(p.logger, "Connection attempt has failed.", zap.String("address", addr), zap.Error(err), zap.Int64("gid", gid))
			} else {
				p.logger.Debug("Successfully opened yamux Session.",
					zap.String("local_address", conn.LocalAddr().String()),
					zap.String("remote_address", conn.RemoteAddr().String()),
					zap.Int64("gid", gid))
			}

			return conn, err
		}))
	if err != nil {
		gid := goid.Get()
		domain.LogErrorWithoutStacktrace(p.logger, "Failed to create a gRPC connection using dummy dialer.",
			zap.Error(err),
			zap.Int64("gid", gid))
		return err
	}

	p.sugaredLogger.Debugf("Successfully created gRPC connection using dummy dialer. Target: %v", gConn.Target())

	p.DistributedClusterClient = gateway.NewDistributedClusterClient(gConn)
	return nil
}

// Validate validates the provisioner client.
func (p *connectionProvisioner) Validate() (*gateway.DashboardRegistrationResponse, error) {
	if p.DistributedClusterClient == nil {
		p.logger.Error("Cannot validate connection with Distributed Cluster Gateway. gRPC client is not initialized.")
		return nil, ErrProvisionerNotInitialized
	}

	p.logger.Debug("Validating connection to Gateway now...")

	// Test the connection
	resp, err := p.DistributedClusterClient.RegisterDashboard(context.Background(), &gateway.Void{})
	if err != nil {
		p.logger.Error("Failed to validate connection with Distributed Cluster Gateway.", zap.Error(err))
	} else {
		p.sugaredLogger.Debugf("Successfully validated connection to Gateway: %v", resp)
	}

	return resp, err
}
