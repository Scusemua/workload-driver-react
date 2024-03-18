package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	gateway "github.com/scusemua/workload-driver-react/m/v2/internal/server/api/proto"
	"github.com/scusemua/workload-driver-react/m/v2/internal/server/proxy"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// This type of handler issues HTTP requests to the backend.
type GrpcClient struct {
	*BaseHandler

	gatewayAddress     string                       // Address that the Cluster Gateway's gRPC server is listening on.
	rpcClient          gateway.ClusterGatewayClient // gRPC client to the Cluster Gateway.
	connectedToGateway bool                         // Indicates whether we're connected or not.
}

func NewGrpcClient(opts *domain.Configuration, shouldConnect bool) *GrpcClient {
	handler := &GrpcClient{
		BaseHandler: newBaseHandler(opts),
	}

	if shouldConnect {
		err := handler.DialGatewayGRPC(opts.GatewayAddress)
		if err != nil {
			panic(err)
		}
	}

	handler.BackendHttpGetHandler = handler

	return handler
}

// Write an error back to the client.
func (h *GrpcClient) WriteError(c *gin.Context, errorMessage string) {
	// Write error back to front-end.
	msg := &domain.ErrorMessage{
		ErrorMessage: errorMessage,
		Valid:        true,
	}
	c.JSON(http.StatusInternalServerError, msg)
}

func (h *GrpcClient) HandleRequest(c *gin.Context) {
	h.BackendHttpGetHandler.HandleRequest(c)
}

// Attempt to connect to the Cluster Gateway's gRPC server using the provided address. Returns an error if connection failed, or nil on success. This should NOT be called from the UI goroutine.
func (h *GrpcClient) DialGatewayGRPC(gatewayAddress string) error {
	if gatewayAddress == "" {
		return domain.ErrEmptyGatewayAddr
	}

	h.logger.Debug("Attempting to dial Gateway gRPC server now.", zap.String("gateway-address", gatewayAddress))

	var numTries int = 0
	var maxNumTries int = 5

	var conn *grpc.ClientConn
	var err error
	for numTries < maxNumTries {
		webSocketProxyClient := proxy.NewWebSocketProxyClient(time.Second * 10)
		conn, err = grpc.Dial("ws://"+gatewayAddress, grpc.WithContextDialer(webSocketProxyClient.Dialer), grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())

		if err == nil { // Connection successful?
			break
		}

		h.logger.Error("Failed to dial Gateway gRPC server.", zap.String("gateway-address", gatewayAddress), zap.Error(err))
		numTries += 1

		// Don't sleep if we're just gonna stop trying afterwards.
		// Only sleep if we're going to try again!
		if numTries < maxNumTries {
			time.Sleep(time.Second * (time.Duration(numTries)))
			continue
		} else {
			return err
		}
	}

	h.logger.Debug("Successfully dialed Cluster Gateway.", zap.String("gateway-address", gatewayAddress))
	h.rpcClient = gateway.NewClusterGatewayClient(conn)
	h.gatewayAddress = gatewayAddress
	h.connectedToGateway = true

	return nil
}
