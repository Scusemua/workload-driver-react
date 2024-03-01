package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	gateway "github.com/scusemua/workload-driver-react/m/v2/server/api/proto"
	"github.com/scusemua/workload-driver-react/m/v2/server/config"
	"github.com/scusemua/workload-driver-react/m/v2/server/domain"
	"github.com/scusemua/workload-driver-react/m/v2/server/proxy"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// This type of handler issues HTTP requests to the backend.
type BaseGRPCHandler struct {
	*BaseHandler

	gatewayAddress     string                       // Address that the Cluster Gateway's gRPC server is listening on.
	rpcClient          gateway.ClusterGatewayClient // gRPC client to the Cluster Gateway.
	connectedToGateway bool                         // Indicates whether we're connected or not.
}

func newBaseGRPCHandler(opts *config.Configuration, shouldConnect bool) *BaseGRPCHandler {
	handler := &BaseGRPCHandler{
		BaseHandler: newBaseHandler(opts),
	}

	if shouldConnect {
		err := handler.DialGatewayGRPC(opts.GatewayAddress)
		if err != nil {
			panic(err)
		}
	}

	handler.BackendHttpHandler = handler

	return handler
}

// Write an error back to the client.
func (h *BaseGRPCHandler) WriteError(c *gin.Context, errorMessage string) {
	// Write error back to front-end.
	msg := &domain.ErrorMessage{
		ErrorMessage: errorMessage,
		Valid:        true,
	}
	c.JSON(http.StatusInternalServerError, msg)
}

func (h *BaseGRPCHandler) HandleRequest(c *gin.Context) {
	h.BackendHttpHandler.HandleRequest(c)
}

// Attempt to connect to the Cluster Gateway's gRPC server using the provided address. Returns an error if connection failed, or nil on success. This should NOT be called from the UI goroutine.
func (h *BaseGRPCHandler) DialGatewayGRPC(gatewayAddress string) error {
	if gatewayAddress == "" {
		return domain.ErrEmptyGatewayAddr
	}

	h.logger.Debug("Attempting to dial Gateway gRPC server now.", zap.String("gateway-address", gatewayAddress))

	webSocketProxyClient := proxy.NewWebSocketProxyClient(time.Minute)
	conn, err := grpc.Dial("ws://"+gatewayAddress, grpc.WithContextDialer(webSocketProxyClient.Dialer), grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		h.logger.Error("Failed to dial Gateway gRPC server.", zap.String("gateway-address", gatewayAddress), zap.Error(err))
		return err
	}

	h.logger.Debug("Successfully dialed Cluster Gateway.", zap.String("gateway-address", gatewayAddress))
	h.rpcClient = gateway.NewClusterGatewayClient(conn)
	h.gatewayAddress = gatewayAddress
	h.connectedToGateway = true

	return nil
}
