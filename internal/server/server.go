package server

import (
	"fmt"
	"net/http"
	"net/url"
	"sync"

	"github.com/gin-gonic/contrib/cors"
	"github.com/gin-gonic/contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/koding/websocketproxy"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	"github.com/scusemua/workload-driver-react/m/v2/internal/server/driver"
	"github.com/scusemua/workload-driver-react/m/v2/internal/server/handlers"
	"github.com/scusemua/workload-driver-react/m/v2/internal/server/proxy"
	"go.uber.org/zap"
)

type serverImpl struct {
	logger         *zap.Logger
	sugaredLogger  *zap.SugaredLogger
	opts           *domain.Configuration
	app            *proxy.JupyterProxyRouter
	engine         *gin.Engine
	workloadDriver *driver.WorkloadDriver
}

func NewServer(opts *domain.Configuration) domain.Server {
	s := &serverImpl{
		opts:           opts,
		engine:         gin.New(),
		workloadDriver: driver.NewWorkloadDriver(opts),
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

		// Used internally (by the frontend) to trigger the start of a new workload.
		apiGroup.POST(domain.START_WORKLOAD_ENDPOINT, s.workloadDriver.HandleRequest)
	}

	if s.opts.SpoofKernelSpecs {
		jupyterGroup := s.app.Group(domain.JUPYTER_GROUP_ENDPOINT)
		{
			jupyterGroup.GET(domain.BASE_API_GROUP_ENDPOINT+domain.KERNEL_SPEC_ENDPOINT, handlers.NewJupyterAPIHandler(s.opts).HandleGetKernelSpecRequest)
		}
	}

	return nil
}

// Blocking call.
func (s *serverImpl) Serve() error {
	var wg sync.WaitGroup
	wg.Add(2)

	s.logger.Debug("Listening for HTTP requests.", zap.String("address", fmt.Sprintf("127.0.0.1:%d", s.opts.ServerPort)))
	go func() {
		addr := fmt.Sprintf(":%d", s.opts.ServerPort)
		if err := http.ListenAndServe(addr, s.app); err != nil {
			s.sugaredLogger.Error("HTTP Server failed to listen on '%s'. Error: %v", addr, err)
			panic(err)
		}

		wg.Done()
	}()

	wsUrlString := fmt.Sprintf("ws://%s", s.opts.JupyterServerAddress)
	wsUrl, err := url.Parse(wsUrlString)
	if err != nil {
		s.logger.Error("Failed to parse URL for websocket proxy.", zap.String("url", wsUrlString), zap.Error(err))
		panic(err)
	}

	s.logger.Debug(fmt.Sprintf("Listening for Websocket Connections on '127.0.0.1:%d' and proxying them to '%s'\n", s.opts.WebsocketProxyPort, wsUrl))
	addr := fmt.Sprintf("127.0.0.1:%d", s.opts.WebsocketProxyPort)
	go func() {
		if err := http.ListenAndServe(addr, websocketproxy.NewProxy(wsUrl)); err != nil {
			s.sugaredLogger.Error("Websocket Proxy failed to listen on '%s'. Error: %v", addr, err)
			panic(err)
		}

		wg.Done()
	}()

	wg.Wait()
	return nil
}
