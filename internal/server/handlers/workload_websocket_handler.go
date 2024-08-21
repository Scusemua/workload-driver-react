package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	"go.uber.org/zap"
)

type WorkloadWebsocketHandler struct {
	logger        *zap.Logger
	sugaredLogger *zap.SugaredLogger
}

func (s *WorkloadWebsocketHandler) serveWorkloadWebsocket(c *gin.Context) {
	s.logger.Debug("Handling workload-related websocket connection")
	expectedOriginV1 := fmt.Sprintf("http://127.0.0.1:%d", s.expectedOriginPort)
	expectedOriginV2 := fmt.Sprintf("http://localhost:%d", s.expectedOriginPort)
	s.logger.Debug("Handling websocket origin.", zap.String("request-origin", c.Request.Header.Get("Origin")), zap.String("request-host", c.Request.Host), zap.String("request-uri", c.Request.RequestURI), zap.String("expected-origin-v1", expectedOriginV1), zap.String("expected-origin-v2", expectedOriginV2))

	upgrader.CheckOrigin = func(r *http.Request) bool {
		if r.Header.Get("Origin") == expectedOriginV1 || r.Header.Get("Origin") == expectedOriginV2 {
			return true
		}

		s.sugaredLogger.Errorf("Unexpected origin: %v", r.Header.Get("Origin"))
		return false
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer conn.Close()

	var concurrentConn domain.ConcurrentWebSocket = newConcurrentWebSocket(conn)

	// Used to notify the server-push goroutine that a new workload has been registered.
	workloadStartedChan := make(chan string)
	doneChan := make(chan struct{})
	go s.serverPushRoutine(workloadStartedChan, doneChan)

	for {
		_, message, err := concurrentConn.ReadMessage()
		if err != nil {
			s.logger.Error("Error while reading message from websocket.", zap.Error(err))
			break
		}

		var request map[string]interface{}
		err = json.Unmarshal(message, &request)
		if err != nil {
			s.logger.Error("Error while unmarshalling data message from workload-related websocket.", zap.Error(err), zap.ByteString("message-bytes", message), zap.String("message-string", string(message)))

			time.Sleep(time.Millisecond * 100)
			continue
		}

		s.sugaredLogger.Debugf("Received workload-related WebSocket message: %v", request)

		var op_val interface{}
		var msgIdVal interface{}
		var ok bool
		if op_val, ok = request["op"]; !ok {
			s.logger.Error("Received unexpected message on websocket. It did not contain an 'op' field.", zap.Binary("message", message))

			time.Sleep(time.Millisecond * 100)
			continue
		}

		if msgIdVal, ok = request["msg_id"]; !ok {
			s.logger.Error("Received unexpected message on websocket. It did not contain a 'msg_id' field.", zap.Binary("message", message))

			time.Sleep(time.Millisecond * 100)
			continue
		}

		op := op_val.(string)
		msgId := msgIdVal.(string)
		if op == "get_workloads" {
			s.handleGetWorkloads(msgId, nil, true)
		} else if op == "register_workload" {
			var wrapper *domain.WorkloadRegistrationRequestWrapper
			json.Unmarshal(message, &wrapper)
			s.handleRegisterWorkload(wrapper.WorkloadRegistrationRequest, msgId, concurrentConn)
		} else if op == "start_workload" {
			var req *domain.StartStopWorkloadRequest
			json.Unmarshal(message, &req)
			s.handleStartWorkload(req, workloadStartedChan)
		} else if op == "stop_workload" {
			var req *domain.StartStopWorkloadRequest
			json.Unmarshal(message, &req)
			s.handleStopWorkload(req)
		} else if op == "stop_workloads" {
			var req *domain.StartStopWorkloadsRequest
			json.Unmarshal(message, &req)
			s.handleStopWorkloads(req)
		} else if op == "pause_workload" {
			var req *domain.PauseUnpauseWorkloadRequest
			json.Unmarshal(message, &req)
			s.handlePauseWorkload(req)
		} else if op == "unpause_workload" {
			var req *domain.PauseUnpauseWorkloadRequest
			json.Unmarshal(message, &req)
			s.handleUnpauseWorkload(req)
		} else if op == "toggle_debug_logs" {
			var req *domain.ToggleDebugLogsRequest
			json.Unmarshal(message, &req)
			s.handleToggleDebugLogs(req)
		} else if op == "subscribe" {
			var req *domain.SubscriptionRequest
			json.Unmarshal(message, &req)
			s.handleSubscriptionRequest(req, concurrentConn)
		} else {
			s.logger.Error("Unexpected or unsupported operation specified.", zap.String("op", op))
		}
	}
}
