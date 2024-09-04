package handlers

import (
	"context"
	"errors"
	"fmt"
	"log"
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

	logger        *zap.Logger
	sugaredLogger *zap.SugaredLogger

	clusterDashboardHandlerPort int

	// Called in its own Goroutine when the ErrorOccurred RPC is called by the ClusterGateway to us.
	notificationCallback notificationCallback

	srv *grpc.Server

	gatewayAddress string // Address that the Cluster Gateway's gRPC server is listening on.
}

func NewClusterDashboardHandler(opts *domain.Configuration, shouldConnect bool, notificationCallback notificationCallback) *ClusterDashboardHandler {
	handler := &ClusterDashboardHandler{
		clusterDashboardHandlerPort: opts.ClusterDashboardHandlerPort,
		gatewayAddress:              opts.GatewayAddress,
		setupInProgress:             0,
		notificationCallback:        notificationCallback,
	}

	var err error
	handler.logger, err = zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	handler.sugaredLogger = handler.logger.Sugar()

	if shouldConnect {
		err := handler.setupRpcResources(opts.GatewayAddress)
		if err != nil {
			handler.logger.Error("Failed to dial gRPC Cluster Gateway.", zap.Error(err))
			panic(err)
		}
	}

	return handler
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

	h.logger.Debug("Handling connection error: attempting to reconnect to Cluster Gateway.")
	go h.setupRpcResources(h.gatewayAddress)
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

// setupRpcResources sets up the gRPC resources (client and server).
func (h *ClusterDashboardHandler) setupRpcResources(gatewayAddress string) error {
	swapped := atomic.CompareAndSwapInt32(&h.setupInProgress, 0, 1)
	if !swapped {
		h.logger.Debug("There is already 'setup RPC resources' operation taking place.")
		return ErrConcurrentSetupOperations
	}

	h.logger.Debug("Dialing Cluster Gateway now.", zap.String("gateway-address", gatewayAddress))

	gOpts := []grpc.ServerOption{
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Timeout: 120 * time.Second,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			PermitWithoutStream: true,
		}),
	}

	if h.srv != nil {
		h.logger.Warn("Cluster Dashboard Handler already has an existing server. Attempting to stop that server first.")

		ctx, cancel := context.WithTimeout(context.TODO(), time.Second*30)
		defer cancel()
		doneChan := make(chan interface{})

		go func(doneChan chan interface{}) {
			h.logger.Debug("Attempting to stop existing gRPC server gracefully.")
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
				h.logger.Debug("Successfully stopped existing gRPC server gracefully.")
			}
		case <-ctx.Done():
			{
				h.logger.Warn("Failed to stop existing gRPC server gracefully. Forcefully stopping server now.")
				h.srv.Stop()
				h.logger.Debug("Successfully stopped existing gRPC server forcefully.")
			}
		}
	}

	h.srv = grpc.NewServer(gOpts...)
	gateway.RegisterClusterDashboardServer(h.srv, h)

	// Initialize gRPC listener
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", h.clusterDashboardHandlerPort))
	if err != nil {
		h.logger.Error("Failed to listen.", zap.Error(err))
		h.exitSetup()
		return err
	}
	// defer lis.Close()

	h.logger.Info("ClusterDashboardHandler listening for gRPC.", zap.Any("address", lis.Addr()))

	start := time.Now()
	connectionTimeout := time.Second * 30 // How long each individual connection attempt should last before timing-out.
	totalTimeout := time.Minute * 5       // How long to keep trying to connect over-and-over before completely giving up and panicking.
	var connectedToGateway bool = false
	var numAttempts int = 1
	var gatewayConn net.Conn
	for !connectedToGateway && time.Since(start) < (totalTimeout) {
		h.sugaredLogger.Debugf("Attempt #%d to connect to Gateway at %s. Connection timeout: %v. Time elapsed: %v.", numAttempts, gatewayAddress, connectionTimeout, time.Since(start))
		gatewayConn, err = net.DialTimeout("tcp", gatewayAddress, connectionTimeout)

		if err != nil {
			h.sugaredLogger.Warnf("Failed to connect to provisioner at %s on attempt #%d: %v. Time elapsed: %v.", gatewayAddress, numAttempts, err, time.Since(start))
			numAttempts += 1
			time.Sleep(time.Second * 3)
		} else {
			connectedToGateway = true
			h.logger.Debug("Successfully connected to Gateway.", zap.String("gateway-address", gatewayAddress), zap.Duration("time-elapsed", time.Since(start)))
		}
	}

	if !connectedToGateway {
		h.sugaredLogger.Errorf("Failed to connect to Gateway after %d attempts. Time elapsed: %v.", numAttempts, time.Since(start))
		h.exitSetup()
		return ErrFailedToConnect
	}

	// Initialize provisioner and wait for ready
	provisioner, err := newConnectionProvisioner(gatewayConn)
	if err != nil {
		h.logger.Error("Failed to initialize the provisioner.", zap.Error(err))
		h.exitSetup()
		return err
	}

	// Wait for reverse connection
	go func() {
		defer h.finalize(true)
		numTries := 0
		maxNumAttempts := 5
		for numTries < maxNumAttempts {
			provisioner.logger.Debug("Trying to connect.", zap.Int("attempt-number", numTries+1))
			if err := h.srv.Serve(provisioner); err != nil {
				provisioner.logger.Error("Failed to serve reverse connection.", zap.Int("attempt-number", numTries+1), zap.Error(err))
				numTries += 1

				if numTries < 3 {
					time.Sleep((time.Millisecond * 1000) * time.Duration(numTries))
					continue
				} else {
					provisioner.sugaredLogger.Errorf("Failed to serve reverse connection after %d attempts. Aborting.", maxNumAttempts)
					provisioner.failedToConnect <- err
					return
				}
			}
		}

	}()

	select {
	case <-provisioner.Ready():
		{
			h.logger.Debug("Provisioner connected successfully and is now ready.")
		}
	case err = <-provisioner.FailedToConnect():
		{
			h.sugaredLogger.Errorf("Provisioner failed to connect successfully. Aborting. Last error: %v.", err)
			h.exitSetup()
			return err
		}
	}

	if err := provisioner.Validate(); err != nil {
		log.Fatalf("Failed to validate reverse provisioner connection: %v", zap.Error(err))
		h.exitSetup()
		return err
	}

	h.DistributedClusterClient = provisioner
	h.logger.Info("Connected to Cluster Gateway.", zap.Any("remote-address", gatewayConn.RemoteAddr()))

	// Start gRPC server
	go func() {
		defer h.finalize(true)
		if err := h.srv.Serve(lis); err != nil {
			log.Fatalf("Failed to serve regular connection: %v", err)
		}
	}()

	h.exitSetup()

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
	c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("could not handle request: %s", errorMessage))
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

type connectionProvisioner struct {
	net.Listener
	gateway.DistributedClusterClient

	ready           chan struct{}
	failedToConnect chan error
	logger          *zap.Logger
	sugaredLogger   *zap.SugaredLogger
}

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

	// Initialize the gRPC client
	if err := provisioner.InitClient(srvSession); err != nil {
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
		p.logger.Error("Failed to accept connection.", zap.Error(err))
		return nil, err
	}

	p.sugaredLogger.Infof("Accepted connection. RemoteAddr: %v. LocalAddr: %v", conn.RemoteAddr(), conn.LocalAddr())

	// Notify possible blocking caller that the gRPC client is initialized
	go func() {
		select {
		case p.ready <- struct{}{}:
			p.sugaredLogger.Infof("connectionProvisioner is ready.")
		default:
			p.logger.Warn("Unexpected duplicated reverse provisioner connection.")
		}
	}()

	return conn, nil
}

// InitClient initializes the gRPC client
func (p *connectionProvisioner) InitClient(session *yamux.Session) error {
	// Dial to create a gRPC connection with dummy dialer.
	gConn, err := grpc.Dial(":0",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
			conn, err := session.Open()
			if err != nil {
				p.sugaredLogger.Errorf("Failed to open CLI session during dial: %s", err.Error())
			} else {
				p.sugaredLogger.Debugf("Opened cliSession. conn.LocalAddr(): %v, conn.RemoteAddr(): %v", conn.LocalAddr(), conn.RemoteAddr())
			}

			return conn, err
		}))
	if err != nil {
		p.sugaredLogger.Errorf("Failed to create a gRPC connection using dummy dialer because: %s", err.Error())
		return err
	}

	p.sugaredLogger.Debugf("Successfully created gRPC connection using dummy dialer. Target: %v", gConn.Target())

	p.DistributedClusterClient = gateway.NewDistributedClusterClient(gConn)
	return nil
}

// Validate validates the provisioner client.
func (p *connectionProvisioner) Validate() error {
	if p.DistributedClusterClient == nil {
		p.logger.Error("Cannot validate connection with Distributed Cluster Gateway. gRPC client is not initialized.")
		return ErrProvisionerNotInitialized
	}

	p.logger.Debug("Validating connection to Gateway now...")

	// Test the connection
	resp, err := p.DistributedClusterClient.Ping(context.Background(), &gateway.Void{})
	if err != nil {
		p.logger.Error("Failed to validate connection with Distributed Cluster Gateway.", zap.Error(err))
	} else {
		p.sugaredLogger.Debugf("Successfully validated connection to Gateway: %s", resp.Id)
	}

	return err
}

// Attempt to connect to the Cluster Gateway's gRPC server using the provided address. Returns an error if connection failed, or nil on success. This should NOT be called from the UI goroutine.
// func (h *ClusterDashboardHandler) setupRpcResources(gatewayAddress string) error {
// 	if gatewayAddress == "" {
// 		return domain.ErrEmptyGatewayAddr
// 	}

// 	h.logger.Debug("Attempting to dial Gateway gRPC server now.", zap.String("gateway-address", gatewayAddress))

// 	var numTries int = 0
// 	var maxNumTries int = 5

// 	var conn *grpc.ClientConn
// 	var err error
// 	for numTries < maxNumTries {
// 		webSocketProxyClient := proxy.NewWebSocketProxyClient(time.Second * 10)
// 		conn, err = grpc.Dial("ws://"+gatewayAddress, grpc.WithContextDialer(webSocketProxyClient.Dialer), grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())

// 		if err == nil { // Connection successful?
// 			break
// 		}

// 		h.logger.Error("Failed to dial Gateway gRPC server.", zap.String("gateway-address", gatewayAddress), zap.Error(err))
// 		numTries += 1

// 		// Don't sleep if we're just gonna stop trying afterwards.
// 		// Only sleep if we're going to try again!
// 		if numTries < maxNumTries {
// 			time.Sleep(time.Second * (time.Duration(numTries)))
// 			continue
// 		} else {
// 			return err
// 		}
// 	}

// 	h.logger.Debug("Successfully dialed Cluster Gateway.", zap.String("gateway-address", gatewayAddress))
// 	h.DistributedClusterClient = gateway.NewDistributedClusterClient(conn)
// 	h.gatewayAddress = gatewayAddress
// 	h.connectedToGateway = true

// 	return nil
// }
