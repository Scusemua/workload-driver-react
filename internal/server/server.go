package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/elliotchance/orderedmap/v2"
	"github.com/gin-gonic/contrib/cors"
	"github.com/gin-gonic/contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/koding/websocketproxy"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
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
	workloadDrivers    *orderedmap.OrderedMap[string, *driver.WorkloadDriver] // Map from workload ID to the associated driver.
	workloadsMap       *orderedmap.OrderedMap[string, *domain.Workload]       // Map from workload ID to workload
	workloads          []*domain.Workload
	pushUpdateInterval time.Duration

	driversMutex   sync.RWMutex
	workloadsMutex sync.RWMutex
}

func NewServer(opts *domain.Configuration) domain.Server {
	s := &serverImpl{
		opts:               opts,
		pushUpdateInterval: time.Second * time.Duration(opts.PushUpdateInterval),
		engine:             gin.New(),
		workloadDrivers:    orderedmap.NewOrderedMap[string, *driver.WorkloadDriver](),
		workloadsMap:       orderedmap.NewOrderedMap[string, *domain.Workload](),
		workloads:          make([]*domain.Workload, 0),
		// workloadDriver: driver.NewWorkloadDriver(opts),
	}

	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	s.logger = logger
	s.sugaredLogger = logger.Sugar()

	s.setupRoutes()

	return s
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

	s.app.GET(domain.WORKLOAD_ENDPOINT, s.serveWebsocket)

	apiGroup := s.app.Group(domain.BASE_API_GROUP_ENDPOINT)
	{
		// Used internally (by the frontend) to get the current kubernetes nodes from the backend  (i.e., the backend).
		apiGroup.GET(domain.KUBERNETES_NODES_ENDPOINT, handlers.NewKubeNodeHttpHandler(s.opts).HandleRequest)

		// Used internally (by the frontend) to get the system config from the backend  (i.e., the backend).
		apiGroup.GET(domain.SYSTEM_CONFIG_ENDPOINT, handlers.NewConfigHttpHandler(s.opts).HandleRequest)

		// Used internally (by the frontend) to get the current set of Jupyter kernels from us (i.e., the backend).
		apiGroup.GET(domain.GET_KERNELS_ENDPOINT, handlers.NewKernelHttpHandler(s.opts).HandleRequest)

		// Used internally (by the frontend) to get the list of available workload presets from the backend.
		apiGroup.GET(domain.WORKLOAD_PRESET_ENDPOINT, handlers.NewWorkloadPresetHttpHandler(s.opts).HandleRequest)

		// Used internally (by the frontend) to trigger kernel replica migrations.
		apiGroup.POST(domain.MIGRATION_ENDPOINT, handlers.NewMigrationHttpHandler(s.opts).HandleRequest)
	}

	if s.opts.SpoofKernelSpecs {
		jupyterGroup := s.app.Group(domain.JUPYTER_GROUP_ENDPOINT)
		{
			jupyterGroup.GET(domain.BASE_API_GROUP_ENDPOINT+domain.KERNEL_SPEC_ENDPOINT, handlers.NewJupyterAPIHandler(s.opts).HandleGetKernelSpecRequest)
		}
	}

	return nil
}

// Used to push updates about active workloads to the frontend.
func (s *serverImpl) serverPushRoutine(conn *websocket.Conn, workloadStartedChan chan string, doneChan chan struct{}) {
	// Keep track of the active workloads.
	activeWorkloads := make(map[string]*domain.Workload)

	// Add all active workloads to the map.
	for _, workload := range s.workloads {
		if workload.IsRunning() {
			activeWorkloads[workload.ID] = workload
		}
	}

	// We'll loop forever, unless the connection is terminated.
	for {
		// If we have any active workloads, then we'll push some updates to the front-end for the active workloads.
		if len(activeWorkloads) > 0 {
			toRemove := make([]string, 0)
			updatedWorkloads := make([]*domain.Workload, 0)

			s.workloadsMutex.RLock()
			s.driversMutex.RLock()
			// Iterate over all the active workloads.
			// for _, workload := range s.workloadsMap {
			for el := s.workloadsMap.Front(); el != nil; el = el.Next() {
				workload := el.Value

				// If the workload is no longer active, then make a note to remove it and then skip it.
				if !workload.IsRunning() {
					toRemove = append(toRemove, workload.ID)
					continue
				}

				s.driversMutex.RLock()
				associatedDriver, _ := s.workloadDrivers.Get(workload.ID)
				s.driversMutex.RUnlock()
				associatedDriver.LockDriver()

				// Lock the workloads' drivers while we marshal the workloads to JSON.
				updatedWorkloads = append(updatedWorkloads, workload)
			}
			s.driversMutex.RUnlock()
			s.workloadsMutex.RUnlock()

			msgId := uuid.NewString()
			payload, err := json.Marshal(&domain.ActiveWorkloadsUpdate{
				MessageId:        msgId,
				Op:               "active_workloads_update",
				UpdatedWorkloads: updatedWorkloads,
			})

			if err != nil {
				s.logger.Error("Error while marshalling message payload.", zap.Error(err))
				panic(err)
			}

			s.driversMutex.RLock()
			for _, workload := range updatedWorkloads {
				associatedDriver, _ := s.workloadDrivers.Get(workload.ID)
				associatedDriver.UnlockDriver()
			}
			s.driversMutex.RUnlock()

			// Send an update to the frontend.
			conn.WriteMessage(websocket.BinaryMessage, payload)

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

func (s *serverImpl) serveWebsocket(c *gin.Context) {
	s.logger.Debug("Handling websocket connection")

	upgrader.CheckOrigin = func(r *http.Request) bool {
		if r.Header.Get("Origin") == "http://127.0.0.1:9001" || r.Header.Get("Origin") == "http://localhost:9001" {
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

	// Used to notify the server-push goroutine that a new workload has been registered.
	workloadStartedChan := make(chan string)
	doneChan := make(chan struct{})
	go s.serverPushRoutine(conn, workloadStartedChan, doneChan)

	closeHandler := conn.CloseHandler()
	conn.SetCloseHandler(func(code int, text string) error {
		doneChan <- struct{}{} // Tell the server-push goroutine that this connection has been terminated.
		err := closeHandler(code, text)
		return err
	})

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			s.logger.Error("Error while reading message from websocket.", zap.Error(err))
			break
		}

		var request map[string]interface{}
		err = json.Unmarshal(message, &request)
		if err != nil {
			s.logger.Error("Error while unmarshalling data message from websocket.", zap.Error(err))
			break
		}

		s.sugaredLogger.Debugf("Received WebSocket message: %v", request)

		var op_val interface{}
		var msgIdVal interface{}
		var ok bool
		if op_val, ok = request["op"]; !ok {
			s.logger.Error("Received unexpected message on websocket. It did not contain an 'op' field.", zap.Binary("message", message))
			break
		}

		if msgIdVal, ok = request["msg_id"]; !ok {
			s.logger.Error("Received unexpected message on websocket. It did not contain a 'msg_id' field.", zap.Binary("message", message))
			break
		}

		op := op_val.(string)
		msgId := msgIdVal.(string)
		if op == "get_workloads" {
			s.handleGetWorkloads(conn, msgId)
		} else if op == "register_workload" {
			var wrapper *domain.WorkloadRegistrationRequestWrapper
			json.Unmarshal(message, &wrapper)
			s.handleRegisterWorkload(wrapper.WorkloadRegistrationRequest, conn, msgId)
		} else if op == "start_workload" {
			var req *domain.StartStopWorkloadRequest
			json.Unmarshal(message, &req)
			s.handleStartWorkload(req, conn, workloadStartedChan)
		} else if op == "stop_workload" {
			var req *domain.StartStopWorkloadRequest
			json.Unmarshal(message, &req)
			s.handleStopWorkload(req, conn)
		} else if op == "stop_workloads" {
			var req *domain.StartStopWorkloadsRequest
			json.Unmarshal(message, &req)
			s.handleStopWorkloads(req, conn)
		} else if op == "toggle_debug_logs" {
			var req *domain.ToggleDebugLogsRequest
			json.Unmarshal(message, &req)
			s.handleToggleDebugLogs(req, conn)
		} else {
			s.logger.Error("Unexpected or unsupported operation specified.", zap.String("op", op))
			conn.WriteJSON(&domain.ErrorMessage{
				Description:  "op",
				ErrorMessage: "Unexpected or unsupported operation specified.",
				Valid:        true,
			})
		}
	}
}

func (s *serverImpl) handleToggleDebugLogs(req *domain.ToggleDebugLogsRequest, conn *websocket.Conn) {
	s.driversMutex.RLock()
	driver, _ := s.workloadDrivers.Get(req.WorkloadId)
	s.driversMutex.RUnlock()

	if driver != nil {
		workload := driver.ToggleDebugLogging(req.Enabled)

		driver.LockDriver()
		payload, err := json.Marshal(&domain.SingleWorkloadResponse{
			MessageId: req.MessageId,
			Workload:  workload,
		})

		if err != nil {
			s.logger.Error("Error while marshalling message payload.", zap.Error(err))
			panic(err)
		}
		driver.UnlockDriver()

		conn.WriteMessage(websocket.BinaryMessage, payload)

		s.logger.Debug("Wrote response for TOGGLE_DEBUG_LOGS to frontend.", zap.String("message-id", req.MessageId))
	} else {
		s.sugaredLogger.Errorf("Could not find driver associated with workload ID=%s", req.WorkloadId)
	}
}

func (s *serverImpl) handleStartWorkload(req *domain.StartStopWorkloadRequest, conn *websocket.Conn, workloadStartedChan chan string) {
	if req.Operation != "start_workload" {
		panic(fmt.Sprintf("Unexpected operation field in StartStopWorkloadRequest: \"%s\"", req.Operation))
	}

	s.logger.Debug("Starting workload.", zap.String("workload-id", req.WorkloadId))

	s.driversMutex.RLock()
	workloadDriver, ok := s.workloadDrivers.Get(req.WorkloadId)
	s.driversMutex.RUnlock()

	if ok {
		var wg sync.WaitGroup
		wg.Add(1)
		go workloadDriver.DriveWorkload(&wg)
		wg.Wait()

		s.workloadsMutex.RLock()
		workload, _ := s.workloadsMap.Get(req.WorkloadId)
		workload.TimeElasped = time.Since(workload.StartTime).String()
		s.workloadsMutex.RUnlock()

		s.logger.Debug("Started workload.", zap.String("workload-id", req.WorkloadId), zap.Any("workload", workload.String()))

		// Lock the workload's driver while we marshal the workload to JSON.
		workloadDriver.LockDriver()
		payload, err := json.Marshal(&domain.SingleWorkloadResponse{
			MessageId: req.MessageId,
			Workload:  workload,
		})

		if err != nil {
			s.logger.Error("Error while marshalling message payload.", zap.Error(err))
			panic(err)
		}
		workloadDriver.UnlockDriver()

		conn.WriteMessage(websocket.BinaryMessage, payload)

		s.logger.Debug("Wrote response for START_WORKLOAD to frontend.", zap.String("message-id", req.MessageId), zap.String("workload-id", workloadDriver.ID()))

		// Notify the server-push goutine that the workload has started.
		workloadStartedChan <- req.WorkloadId
	} else {
		s.logger.Error("Could not find already-registered workload with the given workload ID.", zap.String("workload-id", req.WorkloadId))
	}
}

func (s *serverImpl) handleStopWorkloads(req *domain.StartStopWorkloadsRequest, conn *websocket.Conn) {
	if req.Operation != "stop_workload" {
		panic(fmt.Sprintf("Unexpected operation field in StartStopWorkloadRequest: \"%s\"", req.Operation))
	}

	var updatedWorkloads []*domain.Workload = make([]*domain.Workload, 0, len(req.WorkloadIDs))

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
				workload.TimeElasped = time.Since(workload.StartTime).String()

				s.logger.Debug("Stopped workload.", zap.String("workload-id", workloadID), zap.Any("workload", workload.String()))
				updatedWorkloads = append(updatedWorkloads, workload)
			}
		} else {
			s.logger.Error("Could not find already-registered workload with the given workload ID.", zap.String("workload-id", workloadID))
		}
	}

	// Lock the workload's driver while we marshal the workload to JSON.
	msgId := uuid.NewString()
	payload, err := json.Marshal(&domain.ActiveWorkloadsUpdate{
		MessageId:        msgId,
		UpdatedWorkloads: updatedWorkloads,
	})

	if err != nil {
		s.logger.Error("Error while marshalling message payload.", zap.Error(err))
		panic(err)
	}

	s.driversMutex.RLock()
	for _, workload := range updatedWorkloads {
		associatedDriver, _ := s.workloadDrivers.Get(workload.ID)
		associatedDriver.UnlockDriver()
	}
	s.driversMutex.RUnlock()

	conn.WriteMessage(websocket.BinaryMessage, payload)

	s.logger.Debug("Wrote response for STOP_WORKLOADS to frontend.", zap.String("message-id", req.MessageId), zap.Int("requested-num-workloads-stopped", len(req.WorkloadIDs)), zap.Int("actual-num-workloads-stopped", len(updatedWorkloads)))
}

func (s *serverImpl) handleStopWorkload(req *domain.StartStopWorkloadRequest, conn *websocket.Conn) {
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
			workload.TimeElasped = time.Since(workload.StartTime).String()

			s.logger.Debug("Stopped workload.", zap.String("workload-id", req.WorkloadId), zap.Any("workload", workload.String()))
		}

		// Lock the workload's driver while we marshal the workload to JSON.
		workloadDriver.LockDriver()
		payload, err := json.Marshal(&domain.SingleWorkloadResponse{
			MessageId: req.MessageId,
			Workload:  workloadDriver.GetWorkload(),
		})

		if err != nil {
			s.logger.Error("Error while marshalling message payload.", zap.Error(err))
			panic(err)
		}
		workloadDriver.UnlockDriver()

		conn.WriteMessage(websocket.BinaryMessage, payload)

		s.logger.Debug("Wrote response for STOP_WORKLOAD to frontend.", zap.String("message-id", req.MessageId), zap.String("workload-id", req.WorkloadId))
	} else {
		s.logger.Error("Could not find already-registered workload with the given workload ID.", zap.String("workload-id", req.WorkloadId))
	}
}

func (s *serverImpl) handleRegisterWorkload(request *domain.WorkloadRegistrationRequest, conn *websocket.Conn, msgId string) {
	workloadDriver := driver.NewWorkloadDriver(s.opts)

	workload, _ := workloadDriver.RegisterWorkload(request, conn)

	if workload != nil {
		s.workloadsMutex.Lock()
		s.workloads = append(s.workloads, workload)
		s.workloadsMap.Set(workload.ID, workload)
		s.workloadsMutex.Unlock()

		s.driversMutex.Lock()
		s.workloadDrivers.Set(workload.ID, workloadDriver)
		s.driversMutex.Unlock()

		// Lock the workload's driver while we marshal the workload to JSON.
		workloadDriver.LockDriver()
		payload, err := json.Marshal(&domain.SingleWorkloadResponse{
			MessageId: msgId,
			Workload:  workload,
		})

		if err != nil {
			s.logger.Error("Error while marshalling message payload.", zap.Error(err))
			panic(err)
		}
		workloadDriver.UnlockDriver()

		conn.WriteMessage(websocket.BinaryMessage, payload)

		s.logger.Debug("Wrote response for REGISTER_WORKLOAD to frontend.", zap.String("message-id", msgId), zap.Any("workload", workload))
	} else {
		s.logger.Error("Workload registration did not return a Workload object...")
	}
}

func (s *serverImpl) handleGetWorkloads(conn *websocket.Conn, msgId string) {

	// Lock the workloads' drivers while we marshal the workloads to JSON.
	// for _, driver := range s.workloadDrivers {
	s.driversMutex.RLock()
	for el := s.workloadDrivers.Front(); el != nil; el = el.Next() {
		el.Value.LockDriver()
	}
	s.driversMutex.RUnlock()

	payload, err := json.Marshal(&domain.WorkloadsResponse{
		MessageId: msgId,
		Workloads: s.workloads,
	})

	if err != nil {
		s.logger.Error("Error while marshalling message payload.", zap.Error(err))
		panic(err)
	}

	// for _, driver := range s.workloadDrivers {
	s.driversMutex.RLock()
	for el := s.workloadDrivers.Front(); el != nil; el = el.Next() {
		el.Value.UnlockDriver()
	}
	s.driversMutex.RUnlock()

	s.sugaredLogger.Debugf("Returning %d workloads to user.", len(s.workloads))
	conn.WriteMessage(websocket.BinaryMessage, payload)
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
			s.sugaredLogger.Error("HTTP Server failed to listen on '%s'. Error: %v", addr, err)
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
			s.sugaredLogger.Error("Websocket Proxy failed to listen on '%s'. Error: %v", addr, err)
			panic(err)
		}

		wg.Done()
	}()
}
