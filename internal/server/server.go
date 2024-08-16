package server

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/elliotchance/orderedmap/v2"
	"github.com/gin-gonic/contrib/cors"
	"github.com/gin-gonic/contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/koding/websocketproxy"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	gateway "github.com/scusemua/workload-driver-react/m/v2/internal/server/api/proto"
	"github.com/scusemua/workload-driver-react/m/v2/internal/server/driver"
	"github.com/scusemua/workload-driver-react/m/v2/internal/server/handlers"
	"github.com/scusemua/workload-driver-react/m/v2/internal/server/proxy"
	"go.uber.org/zap"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type serverImpl struct {
	logger             *zap.Logger
	sugaredLogger      *zap.SugaredLogger
	opts               *domain.Configuration
	app                *proxy.JupyterProxyRouter
	engine             *gin.Engine
	workloadDrivers    *orderedmap.OrderedMap[string, domain.WorkloadDriver] // Map from workload ID to the associated driver.
	workloadsMap       *orderedmap.OrderedMap[string, domain.Workload]       // Map from workload ID to workload
	workloads          []domain.Workload
	pushUpdateInterval time.Duration
	gatewayRpcClient   *handlers.ClusterDashboardHandler

	workloadMessageIndex atomic.Int32

	// Websockets that have submitted a workload and thus will want updates for that workload.
	subscribers map[string]domain.ConcurrentWebSocket

	// These are websockets from frontends that are not tied to a particular workload, nor are they used for logs.
	generalWebsockets map[string]domain.ConcurrentWebSocket

	// Used to tell a goroutine to break out of the for-loop in which it is reading logs from Kubernetes.
	// This is used if the websocket connection is terminated. Otherwise, the loop will continue forever.
	getLogsResponseBodies map[string]io.ReadCloser

	expectedOriginPort int

	logResponseBodyMutex sync.RWMutex
	driversMutex         sync.RWMutex
	workloadsMutex       sync.RWMutex
}

func NewServer(opts *domain.Configuration) domain.Server {
	s := &serverImpl{
		opts:                  opts,
		pushUpdateInterval:    time.Second * time.Duration(opts.PushUpdateInterval),
		engine:                gin.New(),
		expectedOriginPort:    opts.ExpectedOriginPort,
		workloadDrivers:       orderedmap.NewOrderedMap[string, domain.WorkloadDriver](),
		workloadsMap:          orderedmap.NewOrderedMap[string, domain.Workload](),
		workloads:             make([]domain.Workload, 0),
		subscribers:           make(map[string]domain.ConcurrentWebSocket),
		generalWebsockets:     make(map[string]domain.ConcurrentWebSocket),
		getLogsResponseBodies: make(map[string]io.ReadCloser),
		// workloadDriver: driver.NewWorkloadDriver(opts),
	}

	s.workloadMessageIndex.Store(0)

	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	s.logger = logger
	s.sugaredLogger = logger.Sugar()

	s.setupRoutes()

	return s
}

func (s *serverImpl) ErrorHandlerMiddleware(c *gin.Context) {
	c.Next()

	errors := make([]*gin.Error, 0, len(c.Errors))
	for _, err := range c.Errors {
		errors = append(errors, err)
	}

	c.JSON(-1, errors)
}

func (s *serverImpl) setupRoutes() error {
	s.app = &proxy.JupyterProxyRouter{
		ContextPath:  domain.JUPYTER_GROUP_ENDPOINT,
		Start:        len(domain.JUPYTER_GROUP_ENDPOINT),
		Config:       s.opts,
		SpoofJupyter: s.opts.SpoofKernelSpecs,
		Engine:       s.engine,
	}

	s.app.ForwardedByClientIP = true
	s.app.SetTrustedProxies([]string{"127.0.0.1"})

	// Serve frontend static files
	s.app.Use(static.Serve("/", static.LocalFile("./dist", true)))
	s.app.Use(gin.Logger())
	s.app.Use(cors.Default())

	////////////////////////
	// Websocket Handlers //
	////////////////////////
	s.app.GET(domain.WORKLOAD_ENDPOINT, s.serveWorkloadWebsocket)
	s.app.GET(domain.LOGS_ENDPOINT, s.serveLogWebsocket)
	s.app.GET(domain.GENERAL_WEBSOCKET_ENDPOINT, s.serveGeneralWebsocket)

	s.gatewayRpcClient = handlers.NewClusterDashboardHandler(s.opts, true, s.notifyFrontend)

	s.sugaredLogger.Debugf("Creating route groups now. (gatewayRpcClient == nil: %v)", s.gatewayRpcClient == nil)

	///////////////////////////////
	// Standard/Primary Handlers //
	///////////////////////////////
	apiGroup := s.app.Group(domain.BASE_API_GROUP_ENDPOINT)
	{
		nodeHandler := handlers.NewKubeNodeHttpHandler(s.opts, s.gatewayRpcClient)
		// Used internally (by the frontend) to get the current kubernetes nodes from the backend  (i.e., the backend).
		apiGroup.GET(domain.KUBERNETES_NODES_ENDPOINT, nodeHandler.HandleRequest)

		// Enable/disable Kubernetes nodes.
		apiGroup.PATCH(domain.KUBERNETES_NODES_ENDPOINT, nodeHandler.HandlePatchRequest)

		// Adjust vGPUs availabe on a particular Kubernetes node.
		apiGroup.PATCH(domain.ADJUST_VGPUS_ENDPOINT, handlers.NewAdjustVirtualGpusHandler(s.opts, s.gatewayRpcClient).HandlePatchRequest)

		// Used internally (by the frontend) to get the system config from the backend  (i.e., the backend).
		apiGroup.GET(domain.SYSTEM_CONFIG_ENDPOINT, handlers.NewConfigHttpHandler(s.opts).HandleRequest)

		// Used internally (by the frontend) to get the current set of Jupyter kernels from us (i.e., the backend).
		apiGroup.GET(domain.GET_KERNELS_ENDPOINT, handlers.NewKernelHttpHandler(s.opts, s.gatewayRpcClient).HandleRequest)

		// Used internally (by the frontend) to get the list of available workload presets from the backend.
		apiGroup.GET(domain.WORKLOAD_PRESET_ENDPOINT, handlers.NewWorkloadPresetHttpHandler(s.opts).HandleRequest)

		// Used internally (by the frontend) to trigger kernel replica migrations.
		apiGroup.POST(domain.MIGRATION_ENDPOINT, handlers.NewMigrationHttpHandler(s.opts, s.gatewayRpcClient).HandleRequest)

		// Used to stream logs from Kubernetes.
		apiGroup.GET(fmt.Sprintf("%s/pods/:pod", domain.LOGS_ENDPOINT), handlers.NewLogHttpHandler(s.opts).HandleRequest)

		// Used to tell a kernel to stop training.
		apiGroup.POST(domain.STOP_TRAINING_ENDPOINT, handlers.NewStopTrainingHandler(s.opts).HandleRequest)
	}

	///////////////////////////
	// Debugging and Testing //
	///////////////////////////
	{
		apiGroup.POST(domain.YIELD_NEXT_REQUEST_ENDPOINT, handlers.NewYieldNextExecuteHandler(s.opts, s.gatewayRpcClient).HandleRequest)

		apiGroup.POST(domain.PANIC_ENDPOINT, handlers.NewPanicHttpHandler(s.opts, s.gatewayRpcClient).HandleRequest)

		apiGroup.POST(domain.SPOOF_NOTIFICATIONS_ENDPOINT, s.handleSpoofedNotifications)

		apiGroup.POST(domain.SPOOF_ERROR_ENDPOINT, s.handleSpoofedError)

		apiGroup.POST(domain.PING_KERNEL_ENDPOINT, handlers.NewPingKernelHttpHandler(s.opts, s.gatewayRpcClient).HandleRequest)
	}

	/////////////////////
	// Jupyter Handler // This isn't really used anymore...
	/////////////////////
	if s.opts.SpoofKernelSpecs {
		jupyterGroup := s.app.Group(domain.JUPYTER_GROUP_ENDPOINT)
		{
			jupyterGroup.GET(domain.BASE_API_GROUP_ENDPOINT+domain.KERNEL_SPEC_ENDPOINT, handlers.NewJupyterAPIHandler(s.opts).HandleGetKernelSpecRequest)
		}
	}

	gin.SetMode(gin.DebugMode)

	s.app.Use(s.ErrorHandlerMiddleware)

	return nil
}

func (s *serverImpl) notifyFrontend(notification *gateway.Notification) {
	message := &domain.GeneralWebSocketResponse{
		Op:      "notification",
		Payload: notification,
	}

	toRemove := make([]string, 0)

	for remote_ip, conn := range s.generalWebsockets {
		s.logger.Debug("Writing message to general WebSocket.", zap.String("remote-addr", remote_ip))
		err := conn.WriteJSON(message)
		if err != nil {
			s.logger.Debug("Failed to write spoofed error to WebSocket.", zap.String("remote-addr", remote_ip), zap.Error(err))

			if _, ok := err.(*websocket.CloseError); ok || err == websocket.ErrCloseSent {
				s.logger.Debug("Will remove general WebSocket.", zap.String("remote-addr", remote_ip))
				toRemove = append(toRemove, remote_ip)
			} else {
				s.logger.Debug("Cannot remove general WebSocket. Error is not a CloseError...", zap.String("remote-addr", remote_ip))
			}
		} else {
			s.logger.Debug("Successfully wrote message to general WebSocket.", zap.String("remote-addr", remote_ip))
		}
	}

	for _, remote_ip := range toRemove {
		s.logger.Warn("Removing general WebSocket connection.", zap.String("remote_ip", remote_ip))

		ws := s.generalWebsockets[remote_ip]
		err := ws.Close()
		if err != nil {
			s.logger.Error("Error closing websocket.", zap.String("remote_ip", remote_ip), zap.Error(err))
		}

		delete(s.generalWebsockets, remote_ip)
	}
}

func (s *serverImpl) handleSpoofedNotifications(ctx *gin.Context) {
	_, err := s.gatewayRpcClient.SpoofNotifications(context.Background(), &gateway.Void{})

	if err != nil {
		s.logger.Error("Failed to issue `SpoofNotifications` RPC to Cluster Gateway.", zap.Error(err))

		notification := &gateway.Notification{
			Title:            "SpoofedError",
			Message:          fmt.Sprintf("This is a spoofed/fake error message with UUID=%s.", uuid.NewString()),
			NotificationType: int32(domain.ErrorNotification),
			Panicked:         false,
		}

		s.notifyFrontend(notification) // Might be redundant given we're responding with an erroneous status code.
		ctx.AbortWithError(http.StatusInternalServerError, err)
	}

	ctx.Status(http.StatusOK)
}

func (s *serverImpl) handleSpoofedError(ctx *gin.Context) {
	errorMessage := &gateway.Notification{
		Title:            "SpoofedError",
		Message:          fmt.Sprintf("This is a spoofed/fake error message with UUID=%s.", uuid.NewString()),
		NotificationType: int32(domain.ErrorNotification),
		Panicked:         false,
	}

	s.logger.Debug("Broadcasting spoofed error message.", zap.Int("num-recipients", len(s.generalWebsockets)))
	s.notifyFrontend(errorMessage)

	ctx.Status(http.StatusOK)
}

// Used to push updates about active workloads to the frontend.
func (s *serverImpl) serverPushRoutine(workloadStartedChan chan string, doneChan chan struct{}) {
	// Keep track of the active workloads.
	activeWorkloads := make(map[string]domain.Workload)

	// Add all active workloads to the map.
	for _, workload := range s.workloads {
		if workload.IsRunning() {
			activeWorkloads[workload.GetId()] = workload
		}
	}

	// We'll loop forever, unless the connection is terminated.
	for {
		// If we have any active workloads, then we'll push some updates to the front-end for the active workloads.
		if len(activeWorkloads) > 0 {
			toRemove := make([]string, 0)
			updatedWorkloads := make([]domain.Workload, 0)

			s.driversMutex.RLock()
			// Iterate over all the active workloads.
			for _, workload := range activeWorkloads {
				// If the workload is no longer active, then make a note to remove it after this next update.
				// (We need to include it in the update so the frontend knows it's no longer active.)
				if !workload.IsRunning() {
					toRemove = append(toRemove, workload.GetId())
				}

				associatedDriver, _ := s.workloadDrivers.Get(workload.GetId())
				associatedDriver.LockDriver()

				workload.UpdateTimeElapsed() // Update this field.

				// Lock the workloads' drivers while we marshal the workloads to JSON.
				updatedWorkloads = append(updatedWorkloads, workload)
			}
			s.driversMutex.RUnlock()

			var msgId string = uuid.NewString()
			payload, err := json.Marshal(s.createWorkloadResponseMessage(msgId, nil, updatedWorkloads, nil))
			// 	&domain.WorkloadResponse{
			// 	MessageId:         msgId,
			// 	ModifiedWorkloads: updatedWorkloads,
			// 	MessageIndex:      s.workloadMessageIndex.Add(1),
			// })

			if err != nil {
				s.logger.Error("Error while marshalling message payload.", zap.Error(err))
				panic(err)
			}

			s.driversMutex.RLock()
			for _, workload := range updatedWorkloads {
				associatedDriver, _ := s.workloadDrivers.Get(workload.GetId())
				associatedDriver.UnlockDriver()
			}
			s.driversMutex.RUnlock()

			// Send an update to the frontend.
			s.broadcastToWorkloadWebsockets(payload)

			s.logger.Debug("Pushed 'Active Workloads' update to frontend.", zap.String("message-id", msgId))

			// Remove workloads that are now inactive from the map.
			for _, id := range toRemove {
				delete(activeWorkloads, id)
			}
		}

		// In case there are a bunch of notifications in the 'workload started channel', consume all of them before breaking out.
		var done bool = false
		for !done {
			// Do stuff.
			select {
			case id := <-workloadStartedChan:
				{
					s.workloadsMutex.RLock()
					// Add the newly-registered workload to the active workloads map.
					activeWorkloads[id], _ = s.workloadsMap.Get(id)
					s.workloadsMutex.RUnlock()
				}
			case <-doneChan:
				{
					return
				}
			default:
				// Do nothing.
				time.Sleep(time.Second * 1)
				done = true // No more notifications right now. We'll process what we have.
			}
		}
	}
}

func (s *serverImpl) serveGeneralWebsocket(c *gin.Context) {
	expectedOriginV1 := fmt.Sprintf("http://127.0.0.1:%d", s.expectedOriginPort)
	expectedOriginV2 := fmt.Sprintf("http://localhost:%d", s.expectedOriginPort)
	s.logger.Debug("Handling websocket origin.", zap.String("request-origin", c.Request.Header.Get("Origin")), zap.String("request-host", c.Request.Host), zap.String("request-uri", c.Request.RequestURI), zap.String("expected-origin-v1", expectedOriginV1), zap.String("expected-origin-v2", expectedOriginV2))

	upgrader.CheckOrigin = func(r *http.Request) bool {
		if r.Header.Get("Origin") == expectedOriginV1 || r.Header.Get("Origin") == expectedOriginV2 {
			return true
		}

		s.sugaredLogger.Errorf("Unexpected origin: %v.", r.Header.Get("Origin"))
		return false
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer conn.Close()

	var concurrentConn domain.ConcurrentWebSocket = newConcurrentWebSocket(conn)
	remote_ip := concurrentConn.RemoteAddr().String()
	s.generalWebsockets[remote_ip] = concurrentConn

	for {
		_, message, err := concurrentConn.ReadMessage()
		if err != nil {
			s.logger.Error("Error while reading message from general websocket.", zap.Error(err))
			delete(s.generalWebsockets, remote_ip)
			break
		}

		var request map[string]interface{}
		err = json.Unmarshal(message, &request)
		if err != nil {
			s.logger.Error("Error while unmarshalling data message from general websocket.", zap.Error(err), zap.ByteString("message-bytes", message), zap.String("message-string", string(message)))
			time.Sleep(time.Millisecond * 100)
			continue
		}

		s.sugaredLogger.Debugf("Received general WebSocket message: %v", request)

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

		s.logger.Debug("Received general WebSocket message.", zap.Any("op", op_val), zap.Any("message-id", msgIdVal))

		var resp map[string]string = make(map[string]string)
		resp["message"] = fmt.Sprintf("Hello there, WebSocket %s.", remote_ip)
		conn.WriteJSON(resp)
	}
}

func (s *serverImpl) serveLogWebsocket(c *gin.Context) {
	s.logger.Debug("Handling log-related websocket connection")
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

	var connectionId string = uuid.NewString()
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			s.logger.Error("Error while reading message from websocket.", zap.String("connection-id", connectionId), zap.String("error-message", err.Error()))

			s.logResponseBodyMutex.RLock()
			responseBody, ok := s.getLogsResponseBodies[connectionId]
			s.logResponseBodyMutex.RUnlock()
			// If we're already processing a get_logs request for this websocket, then terminate that request.
			if ok {
				responseBody.Close()
			}
			break
		}

		var request map[string]interface{}
		err = json.Unmarshal(message, &request)
		if err != nil {
			s.logger.Error("Error while unmarshalling data message from log-related websocket.", zap.Error(err), zap.String("connection-id", connectionId))

			s.logResponseBodyMutex.RLock()
			// If we're already processing a get_logs request for this websocket, then terminate that request.
			if responseBody, ok := s.getLogsResponseBodies[connectionId]; ok {
				responseBody.Close()
			}
			s.logResponseBodyMutex.RUnlock()

			time.Sleep(time.Millisecond * 100)
			continue
		}

		s.sugaredLogger.Debugf("Received log-related WebSocket message: %v", request)

		var op_val interface{}
		var ok bool
		if op_val, ok = request["op"]; !ok {
			s.logger.Error("Received unexpected message on websocket. It did not contain an 'op' field.", zap.Binary("message", message), zap.String("connection-id", connectionId))

			s.logResponseBodyMutex.RLock()
			// If we're already processing a get_logs request for this websocket, then terminate that request.
			if responseBody, ok := s.getLogsResponseBodies[connectionId]; ok {
				responseBody.Close()
			}
			s.logResponseBodyMutex.RUnlock()

			time.Sleep(time.Millisecond * 100)
			continue
		}

		if _, ok := request["msg_id"]; !ok {
			s.logger.Error("Received unexpected message on websocket. It did not contain a 'msg_id' field.", zap.Binary("message", message), zap.String("connection-id", connectionId))

			s.logResponseBodyMutex.RLock()
			// If we're already processing a get_logs request for this websocket, then terminate that request.
			if responseBody, ok := s.getLogsResponseBodies[connectionId]; ok {
				responseBody.Close()
			}
			s.logResponseBodyMutex.RUnlock()

			time.Sleep(time.Millisecond * 100)
			continue
		}

		if op_val == "get_logs" {
			var req *domain.GetLogsRequest
			err = json.Unmarshal(message, &req)

			if err != nil {
				s.logger.Error("Failed to unmarshal GetLogsRequest.", zap.Error(err), zap.String("connection-id", connectionId))
				return
			}

			s.getLogsWebsocket(req, conn, connectionId)
		}
	}
}

func (s *serverImpl) getLogsWebsocket(req *domain.GetLogsRequest, conn *websocket.Conn, connectionId string) {
	s.logger.Debug("Retrieiving logs.", zap.Any("request", req), zap.String("connection-id", connectionId))

	pod := req.Pod
	container := req.Container
	doFollow := req.Follow

	url := fmt.Sprintf("http://localhost:8889/api/v1/namespaces/default/pods/%s/log?container=%s&follow=%v&sinceSeconds=3600", pod, container, doFollow)
	s.logger.Debug("Retrieving logs now.", zap.String("pod", pod), zap.String("container", container), zap.String("url", url), zap.String("connection-id", connectionId))
	resp, err := http.Get(url)
	if err != nil {
		s.logger.Error("Failed to get logs.", zap.String("pod", pod), zap.String("container", container), zap.Error(err), zap.String("connection-id", connectionId))
		return
	}

	if resp.StatusCode != http.StatusOK {
		s.logger.Error("Failed to retrieve logs.", zap.Int("status-code", resp.StatusCode), zap.String("status", resp.Status), zap.String("connection-id", connectionId))
		payload, err := io.ReadAll(resp.Body)
		if err != nil {
			s.sugaredLogger.Errorf("failed to retrieve logs: received HTTP %d %s", resp.StatusCode, resp.Status)
		} else {
			s.sugaredLogger.Errorf("failed to retrieve logs (received HTTP %d %s): %s", resp.StatusCode, resp.Status, payload)
		}
	}

	s.logResponseBodyMutex.RLock()
	// If we're already processing a get_logs request for this websocket, then terminate that request.
	if responseBody, ok := s.getLogsResponseBodies[connectionId]; ok {
		responseBody.Close()
	}
	s.logResponseBodyMutex.RUnlock()

	s.logResponseBodyMutex.Lock()
	s.getLogsResponseBodies[connectionId] = resp.Body
	s.logResponseBodyMutex.Unlock()

	firstReadCompleted := false
	amountToRead := -1
	reader := bufio.NewReader(resp.Body)
	buf := make([]byte, 0)
	for {
		msg, err := reader.ReadString('\n')
		if err != nil {
			s.logger.Error("Failed to read logs from Kubernetes", zap.Error(err), zap.String("connection-id", connectionId))
			return
		}

		buf = append(buf, msg...)

		if !firstReadCompleted {
			amountToRead = reader.Buffered()
			// s.sugaredLogger.Debugf("First read: %d bytes. Bytes buffered: %d. (From container %s of pod %s)", len(msg), amountToRead, container, pod)
			firstReadCompleted = true

			if amountToRead > 0 {
				continue
			}
		}

		if len(buf) < amountToRead {
			// s.sugaredLogger.Debugf("Read %d / %d bytes so far.", len(buf), amountToRead)
			continue
		}

		// messageChan <- buf

		err = conn.WriteMessage(websocket.BinaryMessage, buf)
		if err != nil {
			s.logger.Error("Error while writing stream response for logs.", zap.String("pod", pod), zap.String("container", container), zap.String("connection-id", connectionId), zap.String("error_message", err.Error()))
			return
		}

		buf = buf[:0]
		firstReadCompleted = false
		amountToRead = -1
	}
}

func (s *serverImpl) serveWorkloadWebsocket(c *gin.Context) {
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

// Add a websocket to the subscribers field. This is used for workload-related communication.
func (s *serverImpl) handleSubscriptionRequest(req *domain.SubscriptionRequest, conn domain.ConcurrentWebSocket) {
	s.subscribers[conn.RemoteAddr().String()] = conn
	s.handleGetWorkloads(req.MessageId, conn, false)
}

// Remove a websocket from the subscribers field.
func (s *serverImpl) removeSubscription(conn domain.ConcurrentWebSocket) {
	if conn.RemoteAddr() != nil {
		s.logger.Debug("Removing subscription for WebSocket.", zap.String("remote-address", conn.RemoteAddr().String()))
		delete(s.subscribers, conn.RemoteAddr().String())
	}
}

// Send a binary websocket message to all workload websockets (contained in the 'subscribers' field of the serverImpl struct).
func (s *serverImpl) broadcastToWorkloadWebsockets(payload []byte) []error {
	errors := make([]error, 0)

	toRemove := make([]domain.ConcurrentWebSocket, 0)

	for _, conn := range s.subscribers {
		err := conn.WriteMessage(websocket.BinaryMessage, payload)
		if err != nil {
			s.logger.Error("Error while broadcasting websocket message.", zap.Error(err))
			errors = append(errors, err)

			if _, ok := err.(*websocket.CloseError); ok || err == websocket.ErrCloseSent {
				toRemove = append(toRemove, conn)
			}
		}
	}

	for _, conn := range toRemove {
		s.removeSubscription(conn)
	}

	return errors
}

// Create and return a *domain.WorkloadResponse struct.
// We use this function so we can increment the message index field.
func (s *serverImpl) createWorkloadResponseMessage(id string, new []domain.Workload, modified []domain.Workload, deleted []domain.Workload) *domain.WorkloadResponse {
	return &domain.WorkloadResponse{
		MessageId:         id,
		MessageIndex:      s.workloadMessageIndex.Add(1),
		NewWorkloads:      new,
		ModifiedWorkloads: modified,
		DeletedWorkloads:  deleted,
	}
}

func (s *serverImpl) handleToggleDebugLogs(req *domain.ToggleDebugLogsRequest) {
	s.driversMutex.RLock()
	driver, _ := s.workloadDrivers.Get(req.WorkloadId)
	s.driversMutex.RUnlock()

	if driver != nil {
		workload := driver.ToggleDebugLogging(req.Enabled)

		driver.LockDriver()
		payload, err := json.Marshal(s.createWorkloadResponseMessage(req.MessageId, nil, []domain.Workload{workload}, nil))
		// &domain.WorkloadResponse{
		// 	MessageId:         req.MessageId,
		// 	ModifiedWorkloads: []domain.Workload{workload},
		// })
		driver.UnlockDriver()

		if err != nil {
			s.logger.Error("Error while marshalling message payload.", zap.Error(err))
			panic(err)
		}

		s.broadcastToWorkloadWebsockets(payload)

		s.logger.Debug("Wrote response for TOGGLE_DEBUG_LOGS to frontend.", zap.String("message-id", req.MessageId))
	} else {
		s.sugaredLogger.Errorf("Could not find driver associated with workload ID=%s", req.WorkloadId)
	}
}

func (s *serverImpl) handleStartWorkload(req *domain.StartStopWorkloadRequest, workloadStartedChan chan string) {
	if req.Operation != "start_workload" {
		panic(fmt.Sprintf("Unexpected operation field in StartStopWorkloadRequest: \"%s\"", req.Operation))
	}

	s.logger.Debug("Starting workload.", zap.String("workload-id", req.WorkloadId))

	s.driversMutex.RLock()
	workloadDriver, ok := s.workloadDrivers.Get(req.WorkloadId)
	s.driversMutex.RUnlock()

	if ok {
		// var wg sync.WaitGroup
		// wg.Add(1)

		// Start the workload.
		// This sets the "start time" and transitions the workload to the "running" state.
		workloadDriver.GetWorkload().StartWorkload()

		go workloadDriver.ProcessWorkload(nil) // &wg
		go workloadDriver.DriveWorkload(nil)   // &wg

		s.workloadsMutex.RLock()
		workload, _ := s.workloadsMap.Get(req.WorkloadId)
		workload.UpdateTimeElapsed()
		s.workloadsMutex.RUnlock()

		s.logger.Debug("Started workload.", zap.String("workload-id", req.WorkloadId), zap.Any("workload-source", workload.GetWorkloadSource()))

		// Lock the workload's driver while we marshal the workload to JSON.
		workloadDriver.LockDriver()
		payload, err := json.Marshal(s.createWorkloadResponseMessage(req.MessageId, nil, []domain.Workload{workload}, nil))
		// 	&domain.WorkloadResponse{
		// 	MessageId:         req.MessageId,
		// 	ModifiedWorkloads: []domain.Workload{workload},
		// }
		workloadDriver.UnlockDriver()

		if err != nil {
			s.logger.Error("Error while marshalling message payload.", zap.Error(err))
			panic(err)
		}

		s.broadcastToWorkloadWebsockets(payload)

		s.logger.Debug("Wrote response for START_WORKLOAD to frontend.", zap.String("message-id", req.MessageId), zap.String("workload-id", workloadDriver.ID()))

		// Notify the server-push goutine that the workload has started.
		workloadStartedChan <- req.WorkloadId
	} else {
		s.logger.Error("Could not find already-registered workload with the given workload ID.", zap.String("workload-id", req.WorkloadId))
	}
}

func (s *serverImpl) handlePauseWorkload(req *domain.PauseUnpauseWorkloadRequest) {
	panic("Not implemented yet.")
}

func (s *serverImpl) handleUnpauseWorkload(req *domain.PauseUnpauseWorkloadRequest) {
	panic("Not implemented yet.")
}

func (s *serverImpl) handleStopWorkloads(req *domain.StartStopWorkloadsRequest) {
	if req.Operation != "stop_workloads" {
		panic(fmt.Sprintf("Unexpected operation field in StartStopWorkloadRequest: \"%s\"", req.Operation))
	}

	var updatedWorkloads []domain.Workload = make([]domain.Workload, 0, len(req.WorkloadIDs))

	for _, workloadID := range req.WorkloadIDs {
		s.logger.Debug("Stopping workload.", zap.String("workload-id", workloadID))

		s.driversMutex.RLock()
		workloadDriver, ok := s.workloadDrivers.Get(workloadID)
		s.driversMutex.RUnlock()

		if ok {
			err := workloadDriver.StopWorkload()
			if err != nil {
				s.logger.Error("Error encountered when trying to stop workload.", zap.String("workload-id", workloadID), zap.Error(err))
			} else {
				workload := workloadDriver.GetWorkload()
				// workload.TimeElasped = time.Since(workload.StartTime).String()
				workload.UpdateTimeElapsed()

				s.logger.Debug("Stopped workload.", zap.String("workload-id", workloadID), zap.Any("workload-source", workload.GetWorkloadSource()))
				updatedWorkloads = append(updatedWorkloads, workload)
			}
		} else {
			s.logger.Error("Could not find already-registered workload with the given workload ID.", zap.String("workload-id", workloadID))
		}
	}

	// Lock the workload's driver while we marshal the workload to JSON.
	payload, err := json.Marshal(s.createWorkloadResponseMessage(uuid.NewString(), nil, updatedWorkloads, nil))
	// 	&domain.WorkloadResponse{
	// 	MessageId:         msgId,
	// 	ModifiedWorkloads: updatedWorkloads,
	// }

	if err != nil {
		s.logger.Error("Error while marshalling message payload.", zap.Error(err))
		panic(err)
	}

	s.driversMutex.RLock()
	for _, workload := range updatedWorkloads {
		associatedDriver, _ := s.workloadDrivers.Get(workload.GetId())
		associatedDriver.UnlockDriver()
	}
	s.driversMutex.RUnlock()

	s.broadcastToWorkloadWebsockets(payload)

	s.logger.Debug("Wrote response for STOP_WORKLOADS to frontend.", zap.String("message-id", req.MessageId), zap.Int("requested-num-workloads-stopped", len(req.WorkloadIDs)), zap.Int("actual-num-workloads-stopped", len(updatedWorkloads)))
}

func (s *serverImpl) handleStopWorkload(req *domain.StartStopWorkloadRequest) {
	if req.Operation != "stop_workload" {
		panic(fmt.Sprintf("Unexpected operation field in StartStopWorkloadRequest: \"%s\"", req.Operation))
	}

	s.logger.Debug("Stopping workload.", zap.String("workload-id", req.WorkloadId))

	s.driversMutex.RLock()
	workloadDriver, ok := s.workloadDrivers.Get(req.WorkloadId)
	s.driversMutex.RUnlock()

	if ok {
		err := workloadDriver.StopWorkload()
		if err != nil {
			s.logger.Error("Error encountered when trying to stop workload.", zap.String("workload-id", req.WorkloadId), zap.Error(err))
		} else {
			workload := workloadDriver.GetWorkload()
			// workload.TimeElasped = time.Since(workload.StartTime).String()
			workload.UpdateTimeElapsed()

			s.logger.Debug("Stopped workload.", zap.String("workload-id", req.WorkloadId), zap.Any("workload-source", workload.GetWorkloadSource()))
		}

		// Lock the workload's driver while we marshal the workload to JSON.
		workloadDriver.LockDriver()
		payload, err := json.Marshal(s.createWorkloadResponseMessage(req.MessageId, nil, []domain.Workload{workloadDriver.GetWorkload()}, nil))
		// 	&domain.WorkloadResponse{
		// 	MessageId:         req.MessageId,
		// 	ModifiedWorkloads: []domain.Workload{workloadDriver.GetWorkload()},
		// }
		workloadDriver.UnlockDriver()

		if err != nil {
			s.logger.Error("Error while marshalling message payload.", zap.Error(err))
			panic(err)
		}

		s.broadcastToWorkloadWebsockets(payload)

		s.logger.Debug("Wrote response for STOP_WORKLOAD to frontend.", zap.String("message-id", req.MessageId), zap.String("workload-id", req.WorkloadId))
	} else {
		s.logger.Error("Could not find already-registered workload with the given workload ID.", zap.String("workload-id", req.WorkloadId))
	}
}

func (s *serverImpl) handleRegisterWorkload(request *domain.WorkloadRegistrationRequest, msgId string, websocket domain.ConcurrentWebSocket) {
	workloadDriver := driver.NewWorkloadDriver(s.opts, true, request.TimescaleAdjustmentFactor, websocket)

	workload, _ := workloadDriver.RegisterWorkload(request)

	if workload != nil {
		s.workloadsMutex.Lock()
		s.workloads = append(s.workloads, workload)
		s.workloadsMap.Set(workload.GetId(), workload)
		s.workloadsMutex.Unlock()

		s.driversMutex.Lock()
		s.workloadDrivers.Set(workload.GetId(), workloadDriver)
		s.driversMutex.Unlock()

		// Lock the workload's driver while we marshal the workload to JSON.
		workloadDriver.LockDriver()
		payload, err := json.Marshal(s.createWorkloadResponseMessage(msgId, []domain.Workload{workload}, nil, nil))
		// 	&domain.WorkloadResponse{
		// 	MessageId:    msgId,
		// 	NewWorkloads: []domain.Workload{workload},
		// }
		workloadDriver.UnlockDriver()

		if err != nil {
			s.logger.Error("Error while marshalling message payload.", zap.Error(err))
			panic(err)
		}

		s.broadcastToWorkloadWebsockets(payload)

		s.logger.Debug("Wrote response for REGISTER_WORKLOAD to frontend.", zap.String("message-id", msgId), zap.Any("workload-source", workload.GetWorkloadSource()), zap.Any("workload-id", workload.GetId()))
	} else {
		s.logger.Error("Workload registration did not return a Workload object...")
	}
}

func (s *serverImpl) handleGetWorkloads(msgId string, conn domain.ConcurrentWebSocket, broadcastToWorkloadWebsockets bool) {
	s.driversMutex.RLock()
	for el := s.workloadDrivers.Front(); el != nil; el = el.Next() {
		el.Value.LockDriver()
	}

	payload, err := json.Marshal(s.createWorkloadResponseMessage(msgId, nil, s.workloads, nil))
	// 	&domain.WorkloadResponse{
	// 	MessageId:         msgId,
	// 	ModifiedWorkloads: s.workloads, /* Send all as modified so they're all parsed */
	// }

	if err != nil {
		s.logger.Error("Error while marshalling message payload.", zap.Error(err))
		panic(err)
	}

	for el := s.workloadDrivers.Front(); el != nil; el = el.Next() {
		el.Value.UnlockDriver()
	}
	s.driversMutex.RUnlock()

	s.sugaredLogger.Debugf("Returning %d workloads to user.", len(s.workloads))

	if broadcastToWorkloadWebsockets {
		s.broadcastToWorkloadWebsockets(payload)
	}
	if conn != nil {
		conn.WriteMessage(websocket.BinaryMessage, payload)
	}

	s.logger.Debug("Wrote response for GET_WORKLOADS to frontend.", zap.String("message-id", msgId))
}

// Blocking call.
func (s *serverImpl) Serve() error {
	var wg sync.WaitGroup
	wg.Add(3)

	s.serveHttp(&wg)
	s.serveJupyterWebsocketProxy(&wg)

	wg.Wait()
	return nil
}

func (s *serverImpl) serveHttp(wg *sync.WaitGroup) {
	s.logger.Debug("Listening for HTTP requests.", zap.String("address", fmt.Sprintf("127.0.0.1:%d", s.opts.ServerPort)))
	go func() {
		addr := fmt.Sprintf(":%d", s.opts.ServerPort)
		if err := http.ListenAndServe(addr, s.app); err != nil {
			s.sugaredLogger.Errorf("HTTP Server failed to listen on '%s'. Error: %v", addr, err)
			panic(err)
		}

		wg.Done()
	}()
}

func (s *serverImpl) serveJupyterWebsocketProxy(wg *sync.WaitGroup) {
	wsUrlString := fmt.Sprintf("ws://%s", s.opts.JupyterServerAddress)
	wsUrl, err := url.Parse(wsUrlString)
	if err != nil {
		s.logger.Error("Failed to parse URL for websocket proxy.", zap.String("url", wsUrlString), zap.Error(err))
		panic(err)
	}

	// Websocket connections for the Jupyter Notebook server. We proxy these to the server.
	s.logger.Debug(fmt.Sprintf("Listening for Websocket Connections on '127.0.0.1:%d' and proxying them to '%s'\n", s.opts.WebsocketProxyPort, wsUrl))
	addr := fmt.Sprintf("127.0.0.1:%d", s.opts.WebsocketProxyPort)
	go func() {
		if err := http.ListenAndServe(addr, websocketproxy.NewProxy(wsUrl)); err != nil {
			s.sugaredLogger.Errorf("Websocket Proxy failed to listen on '%s'. Error: %v", addr, err)
			panic(err)
		}

		wg.Done()
	}()
}

type concurrentWebSocketImpl struct {
	rlock sync.Mutex
	wlock sync.Mutex
	conn  *websocket.Conn
}

func newConcurrentWebSocket(conn *websocket.Conn) domain.ConcurrentWebSocket {
	return &concurrentWebSocketImpl{
		conn: conn,
	}
}

// WriteJSON writes the JSON encoding of v as a message.
func (w *concurrentWebSocketImpl) WriteJSON(v interface{}) error {
	log.Printf("Preparing to write JSON message. Acquiring lock...\n")
	w.wlock.Lock()
	defer w.wlock.Unlock()

	log.Printf("Preparing to write JSON message. Successfully acquired lock...\n")
	return w.conn.WriteJSON(v)
}

// Close the websocket.
func (w *concurrentWebSocketImpl) Close() error {
	log.Printf("Preparing to close websocket. Acquiring lock...\n")

	log.Printf("Preparing to close websocket. Successfully acquired lock...\n")
	return w.conn.Close()
}

// WriteMessage is a helper method for getting a writer using NextWriter, writing the message and closing the writer.
func (w *concurrentWebSocketImpl) WriteMessage(messageType int, data []byte) error {
	log.Printf("Preparing to write %v message. Acquiring lock...\n", messageType)
	w.wlock.Lock()
	defer w.wlock.Unlock()

	log.Printf("Preparing to write %v message. Successfully acquired lock...\n", messageType)
	return w.conn.WriteMessage(messageType, data)
}

// ReadJSON reads the next JSON-encoded message from the connection and stores it in the value pointed to by v.
func (w *concurrentWebSocketImpl) ReadJSON(v interface{}) error {
	w.rlock.Lock()
	defer w.rlock.Unlock()

	return w.conn.ReadJSON(v)
}

// ReadMessage is a helper method for getting a reader using NextReader and reading from that reader to a buffer.
func (w *concurrentWebSocketImpl) ReadMessage() (messageType int, p []byte, err error) {
	w.rlock.Lock()
	defer w.rlock.Unlock()

	return w.conn.ReadMessage()
}

// RemoteAddr returns the remote network address.
func (w *concurrentWebSocketImpl) RemoteAddr() net.Addr {
	return w.conn.RemoteAddr()
}
