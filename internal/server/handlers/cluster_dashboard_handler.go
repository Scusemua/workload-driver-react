package handlers

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
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

	sig = make(chan os.Signal, 1)
)

// This type of handler issues HTTP requests to the backend.
type ClusterDashboardHandler struct {
	gateway.DistributedClusterClient // gRPC client to the Cluster Gateway.
	gateway.UnimplementedClusterDashboardServer

	logger        *zap.Logger
	sugaredLogger *zap.SugaredLogger

	clusterDashboardHandlerPort int

	gatewayAddress string // Address that the Cluster Gateway's gRPC server is listening on.
}

func NewClusterDashboardHandler(opts *domain.Configuration, shouldConnect bool) *ClusterDashboardHandler {
	handler := &ClusterDashboardHandler{
		clusterDashboardHandlerPort: opts.ClusterDashboardHandlerPort,
	}

	var err error
	handler.logger, err = zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	handler.sugaredLogger = handler.logger.Sugar()

	if shouldConnect {
		err := handler.dialGatewayGRPC(opts.GatewayAddress)
		if err != nil {
			handler.logger.Error("Failed to dial gRPC Cluster Gateway.", zap.Error(err))
			panic(err)
		}
	}

	return handler
}

func (h *ClusterDashboardHandler) dialGatewayGRPC(gatewayAddress string) error {
	gOpts := []grpc.ServerOption{
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Timeout: 120 * time.Second,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			PermitWithoutStream: true,
		}),
	}

	srv := grpc.NewServer(gOpts...)
	gateway.RegisterClusterDashboardServer(srv, h)

	// Initialize gRPC listener
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", h.clusterDashboardHandlerPort))
	if err != nil {
		h.logger.Error("Failed to listen.", zap.Error(err))
		return err
	}
	// defer lis.Close()

	h.logger.Info("ClusterDashboardHandler listening for gRPC.", zap.Any("address", lis.Addr()))

	start := time.Now()
	connectionTimeout := time.Second * 5
	var connectedToGateway bool = false
	var numAttempts int = 1
	var gatewayConn net.Conn
	for !connectedToGateway && time.Since(start) < (time.Minute*1) {
		h.sugaredLogger.Debugf("Attempt #%d to connect to Gateway at %s. Connection timeout: %v.", numAttempts, gatewayAddress, connectionTimeout)
		gatewayConn, err = net.DialTimeout("tcp", gatewayAddress, connectionTimeout)

		if err != nil {
			h.sugaredLogger.Warnf("Failed to connect to provisioner at %s on attempt #%d: %v", gatewayAddress, numAttempts, err)
			numAttempts += 1
			time.Sleep(time.Second * 3)
		} else {
			connectedToGateway = true
			h.logger.Debug("Successfully connected to Gateway.", zap.String("gateway-address", gatewayAddress))
		}
	}

	if !connectedToGateway {
		h.sugaredLogger.Errorf("Failed to connect to Gateway after %d attempts.", numAttempts)
		return ErrFailedToConnect
	}
	// defer gatewayConn.Close()

	// Initialize provisioner and wait for ready
	provisioner, err := newConnectionProvisioner(gatewayConn)
	if err != nil {
		h.logger.Error("Failed to initialize the provisioner.", zap.Error(err))
		return err
	}

	// Wait for reverse connection
	go func() {
		defer h.finalize(true)
		num_tries := 0
		for num_tries < 3 {
			provisioner.logger.Debug("Trying to connect.")
			if err := srv.Serve(provisioner); err != nil {
				provisioner.logger.Error("Failed to serve reverse connection.", zap.Error(err))
				num_tries += 1
				time.Sleep(time.Millisecond * 500)
				continue
			}
		}
	}()

	<-provisioner.Ready()
	h.logger.Info("I've been informed that the provisioner is ready.")
	if err := provisioner.Validate(); err != nil {
		log.Fatalf("Failed to validate reverse provisioner connection: %v", zap.Error(err))
		return err
	}

	h.DistributedClusterClient = provisioner
	h.logger.Info("Connected to Cluster Gateway.", zap.Any("remote-address", gatewayConn.RemoteAddr()))

	// Start gRPC server
	go func() {
		defer h.finalize(true)
		if err := srv.Serve(lis); err != nil {
			log.Fatalf("Failed to serve regular connection: %v", err)
		}
	}()

	return nil
}

func (h *ClusterDashboardHandler) ErrorOccurred(ctx context.Context, errorMessage *gateway.ErrorMessage) (*gateway.Void, error) {
	h.logger.Error("A fatal error has occurred within the Cluster Gateway!", zap.Any("error", errorMessage))

	return &gateway.Void{}, nil
}

// Write an error back to the client.
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

	ready         chan struct{}
	logger        *zap.Logger
	sugaredLogger *zap.SugaredLogger
}

func newConnectionProvisioner(conn net.Conn) (*connectionProvisioner, error) {
	// Initialize yamux session for bi-directional gRPC calls
	// At host scheduler side, a connection replacement first made, then we wait for reverse connection by implementing net.Listener
	srvSession, err := yamux.Server(conn, yamux.DefaultConfig())
	if err != nil {
		return nil, err
	}

	provisioner := &connectionProvisioner{
		Listener: srvSession,
		ready:    make(chan struct{}),
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
				p.sugaredLogger.Errorf("Failed to open CLI session during dial: %v", err)
			} else {
				p.sugaredLogger.Debugf("Opened cliSession. conn.LocalAddr(): %v, conn.RemoteAddr(): %v", conn.LocalAddr(), conn.RemoteAddr())
			}

			return conn, err
		}))
	if err != nil {
		p.logger.Error("Failed to create a gRPC connection using dummy dialer.", zap.Error(err))
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
// func (h *ClusterDashboardHandler) dialGatewayGRPC(gatewayAddress string) error {
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
