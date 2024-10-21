package server

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/contrib/cors"
	"github.com/gin-gonic/contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/koding/websocketproxy"
	"github.com/mattn/go-colorable"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	gateway "github.com/scusemua/workload-driver-react/m/v2/internal/server/api/proto"
	"github.com/scusemua/workload-driver-react/m/v2/internal/server/auth"
	"github.com/scusemua/workload-driver-react/m/v2/internal/server/concurrent_websocket"
	"github.com/scusemua/workload-driver-react/m/v2/internal/server/handlers"
	"github.com/scusemua/workload-driver-react/m/v2/internal/server/proxy"
	"github.com/scusemua/workload-driver-react/m/v2/internal/server/workload"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"
	"time"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var jwtIdentityKey = "identityKey"

type serverImpl struct {
	logger           *zap.Logger
	sugaredLogger    *zap.SugaredLogger
	atom             *zap.AtomicLevel
	opts             *domain.Configuration
	app              *proxy.JupyterProxyRouter
	engine           *gin.Engine
	gatewayRpcClient *handlers.ClusterDashboardHandler

	// prometheusMetrics is a wrapper around the Prometheus metrics associated with workloads and the server itself.
	//prometheusMetrics *metrics.PrometheusMetricsWrapper

	// Handler returned by promhttp.Handler to serve Prometheus metrics.
	prometheusHandler http.Handler

	// workloadManager is responsible for managing workloads submitted to the server for execution/orchestration.
	workloadManager domain.WorkloadManager

	// nodeHandler is responsible for handling HTTP GET and HTTP PATCH requests for the nodes within the cluster.
	//
	// Initially, nodeHandler returns HTTP 503 "Service Unavailable" for all requests.
	// This changes after the backend server has registered with the Cluster Gateway (via gRPC).
	//
	// The registration procedure ends with the backend server receiving config info from the Cluster Gateway.
	// This info includes the domain.NodeType of the domain.ClusterNode instances within the Cluster.
	//
	// Based on that information, the nodeHandler creates an internal node handler of type either
	// handlers.KubeNodeHttpHandler or handlers.DockerSwarmNodeHttpHandler. From that point forward, all requests are
	// forwarded to the internal node handler, which knows how to handle the requests for the particular domain.NodeType.
	nodeHandler *handlers.NodeHttpHandler

	// These are websockets from frontends that are not tied to a particular workload, nor are they used for logs.
	generalWebsockets map[string]domain.ConcurrentWebSocket

	// Used to tell a goroutine to break out of the for-loop in which it is reading logs from Kubernetes.
	// This is used if the websocket connection is terminated. Otherwise, the loop will continue forever.
	getLogsResponseBodies map[string]io.ReadCloser

	expectedOriginPort      int
	expectedOriginAddresses []string

	logResponseBodyMutex sync.RWMutex

	// The base prefix. Useful as when we deploy this Dockerized in Docker Swarm, we need to set this to
	// something other than "/", as we use Traefik to reverse proxy external requests.
	baseListenPrefix string

	// Endpoint to serve prometheus metrics scraping requests
	// Defined separately from the base-listen-prefix.
	prometheusEndpoint string

	adminUsername           string
	adminPassword           string
	jwtTokenValidDuration   time.Duration
	jwtTokenRefreshInterval time.Duration
}

func NewServer(opts *domain.Configuration) domain.Server {
	atom := zap.NewAtomicLevelAt(zapcore.DebugLevel)
	s := &serverImpl{
		opts:                    opts,
		atom:                    &atom,
		engine:                  gin.New(),
		generalWebsockets:       make(map[string]domain.ConcurrentWebSocket),
		getLogsResponseBodies:   make(map[string]io.ReadCloser),
		workloadManager:         workload.NewWorkloadManager(opts, &atom),
		prometheusHandler:       promhttp.Handler(),
		adminUsername:           opts.AdminUser,
		adminPassword:           opts.AdminPassword,
		jwtTokenValidDuration:   time.Second * time.Duration(opts.TokenValidDurationSec),
		jwtTokenRefreshInterval: time.Second * time.Duration(opts.TokenRefreshIntervalSec),
		expectedOriginPort:      opts.ExpectedOriginPort,
		expectedOriginAddresses: make([]string, 0, len(opts.ExpectedOriginAddresses)),
		baseListenPrefix:        opts.BaseListenPrefix,
		prometheusEndpoint:      opts.PrometheusEndpoint,
	}

	// Default to "/"
	if s.baseListenPrefix == "" {
		s.baseListenPrefix = "/"
	}

	// Default value
	if s.prometheusEndpoint == "" {
		s.prometheusEndpoint = domain.PrometheusEndpoint
	}

	zapConfig := zap.NewDevelopmentEncoderConfig()
	zapConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	core := zapcore.NewCore(zapcore.NewConsoleEncoder(zapConfig), zapcore.AddSync(colorable.NewColorableStdout()), atom)
	logger := zap.New(core, zap.Development())
	if logger == nil {
		panic("failed to create logger for workload driver")
	}

	s.logger = logger
	s.sugaredLogger = logger.Sugar()

	expectedOriginAddresses := strings.Split(opts.ExpectedOriginAddresses, ",")
	for _, addr := range expectedOriginAddresses {
		expectedOrigin := fmt.Sprintf("%s:%d", addr, s.expectedOriginPort)
		s.logger.Debug("Loaded expected origin from configuration.", zap.String("origin", expectedOrigin))
		s.expectedOriginAddresses = append(s.expectedOriginAddresses, expectedOrigin)
	}

	if err := s.setupRoutes(); err != nil {
		panic(err)
	}

	if err := s.templateHtmlFiles(); err != nil {
		panic(err)
	}

	return s
}

// templateHtmlFiles rewrites the {{ base path }} string in the index.html and 200.html files with the base listen path.
func (s *serverImpl) templateHtmlFiles() error {
	executeTemplate := func(filePath string) error {
		content, err := os.ReadFile(filePath)
		if err != nil {
			return err
		}

		// Convert content to a string for replacement
		contentStr := string(content)

		s.logger.Debug("Loaded file.", zap.String("file", filePath), zap.String("content", contentStr),
			zap.String("base_listen_prefix", s.baseListenPrefix))

		// Replace "{{ base_url }}" with the actual base URL
		modifiedContent := strings.Replace(contentStr, "{{ base_URL }}", s.baseListenPrefix, -1)

		s.logger.Debug("Templated file.", zap.String("file", filePath), zap.String("modified-contents", modifiedContent))

		// Write the modified content back to the file (or to a new file)
		err = os.WriteFile(filePath, []byte(modifiedContent), 0644)
		if err != nil {
			return err
		}

		s.logger.Debug("Successfully templated file.", zap.String("file", filePath))

		return nil
	}

	err := executeTemplate("./dist/index.html")
	if err != nil {
		return err
	}

	err = executeTemplate("./dist/200.html")
	if err != nil {
		return err
	}

	return nil
}

// handleRpcRegistrationComplete is a callback for the handlers.ClusterDashboardHandler of the serverImpl to execute
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
// handleRpcRegistrationComplete will be re-triggered.
func (s *serverImpl) handleRpcRegistrationComplete(nodeType domain.NodeType, rpcHandler *handlers.ClusterDashboardHandler) {
	if s.nodeHandler == nil {
		panic("The server's node handler is nil during the execution of the RegistrationCompleteCallback")
	}

	s.logger.Debug("'Registration Complete' callback triggered.", zap.String("node-type", string(nodeType)))
	s.nodeHandler.AssignNodeType(nodeType, rpcHandler)
}

// ErrorHandlerMiddleware is gin middleware to handle errors that occur while the request handlers
// are processing/handling a request.
func (s *serverImpl) ErrorHandlerMiddleware(c *gin.Context) {
	c.Next() // Execute all the handlers.

	s.logger.Debug("Serving request.", zap.String("origin", c.Request.Header.Get("Origin")),
		zap.String("url", c.Request.URL.String()))

	errorsEncountered := make([]error, 0)
	for _, err := range c.Errors {
		errorsEncountered = append(errorsEncountered, err.Err)
		s.logger.Error("Error encountered.", zap.Error(err))
	}

	if len(errorsEncountered) > 0 {
		c.JSON(-1, gin.H{
			"message": errors.Join(errorsEncountered...).Error(),
		})
	}
}

func (s *serverImpl) jwtPayloadFunc() func(data interface{}) jwt.MapClaims {
	return func(data interface{}) jwt.MapClaims {
		s.logger.Debug("Executing jwtPayloadFunc", zap.Any("data", data))
		if v, ok := data.(*auth.AuthorizedUser); ok {
			return jwt.MapClaims{
				jwtIdentityKey: v.Username,
			}
		}
		return jwt.MapClaims{}
	}
}

func (s *serverImpl) jwtIdentityHandler() func(c *gin.Context) interface{} {
	return func(c *gin.Context) interface{} {
		claims := jwt.ExtractClaims(c)
		identity, ok := claims[jwtIdentityKey].(string)
		if ok {
			return &auth.AuthorizedUser{
				Username: identity,
			}
		} else {
			return nil
		}
	}
}

func (s *serverImpl) jwtAuthenticator() func(c *gin.Context) (interface{}, error) {
	return func(c *gin.Context) (interface{}, error) {
		var login *auth.LoginRequest
		if err := c.ShouldBind(&login); err != nil {
			s.logger.Warn("Received login request with missing login values.")
			return "", jwt.ErrMissingLoginValues
		}
		userID := login.Username
		password := login.Password

		s.logger.Debug("Received authentication request.", zap.String("username", userID), zap.String("password", password))

		if userID == s.adminUsername && password == s.adminPassword {
			return &auth.AuthorizedUser{Username: userID}, nil
		}
		return nil, jwt.ErrFailedAuthentication
	}
}

func (s *serverImpl) jwtAuthorizer() func(data interface{}, c *gin.Context) bool {
	return func(data interface{}, c *gin.Context) bool {
		s.logger.Debug("Executing jwtAuthorizer", zap.Any("data", data))

		var (
			user *auth.AuthorizedUser
			ok   bool
		)
		user, ok = data.(*auth.AuthorizedUser)

		if ok {
			s.logger.Debug("Inspecting request for authorization.", zap.String("username", user.Username))

			if user.Username == s.adminUsername {
				s.logger.Debug("Authorizing request from admin user.", zap.String("username", user.Username))
				return true
			} else {
				log.Fatalf("Found non-admin authorized user with username=\"%s\"\n", user.Username)
			}
		} else {
			s.logger.Debug("Rejecting unauthorized request.", zap.Any("data", data))
		}

		return false
	}
}

func (s *serverImpl) jwtHandleUnauthorized() func(c *gin.Context, code int, message string) {
	return func(c *gin.Context, code int, message string) {
		s.logger.Debug("JWT unauthorized request handler called.",
			zap.Int("code", code), zap.String("message", message),
			zap.String("remote_address", c.Request.RemoteAddr),
			zap.String("client_ip", c.ClientIP()))

		c.JSON(code, gin.H{
			"code":    code,
			"message": message,
		})
	}
}

func (s *serverImpl) initJWTParams() *jwt.GinJWTMiddleware {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		panic(err)
	}

	return &jwt.GinJWTMiddleware{
		Realm:             "Distributed Notebook Cluster",
		Key:               key,
		Timeout:           s.jwtTokenValidDuration,
		MaxRefresh:        s.jwtTokenRefreshInterval,
		IdentityKey:       jwtIdentityKey,
		PayloadFunc:       s.jwtPayloadFunc(),
		IdentityHandler:   s.jwtIdentityHandler(),
		Authenticator:     s.jwtAuthenticator(),
		Authorizator:      s.jwtAuthorizer(),
		Unauthorized:      s.jwtHandleUnauthorized(),
		SendAuthorization: true,
		TokenLookup:       "header: Authorization, query: token, cookie: jwt",
		TokenHeadName:     "Bearer",
		TimeFunc:          time.Now,
	}
}

func (s *serverImpl) jwtHandlerMiddleWare(authMiddleware *jwt.GinJWTMiddleware) gin.HandlerFunc {
	return func(context *gin.Context) {
		errInit := authMiddleware.MiddlewareInit()
		if errInit != nil {
			log.Fatal("authMiddleware.MiddlewareInit() Error:" + errInit.Error())
		}
	}
}

func lastChar(target string) uint8 {
	if target == "" {
		panic("Cannot find last character of an empty string!")
	}

	return target[len(target)-1]
}

func (s *serverImpl) getPath(relativePath string) string {
	if relativePath == "" {
		return s.baseListenPrefix
	}

	finalPath := path.Join(s.baseListenPrefix, relativePath)
	if lastChar(relativePath) == '/' && lastChar(finalPath) != '/' {
		return finalPath + "/"
	}
	return finalPath
}

func (s *serverImpl) setupRoutes() error {
	s.app = &proxy.JupyterProxyRouter{
		ContextPath:  domain.JupyterGroupEndpoint,
		Start:        len(domain.JupyterGroupEndpoint),
		Config:       s.opts,
		SpoofJupyter: s.opts.SpoofKernelSpecs,
		Engine:       s.engine,
	}

	s.nodeHandler = handlers.NewNodeHttpHandler(s.opts)

	s.app.ForwardedByClientIP = true
	if err := s.app.SetTrustedProxies([]string{"127.0.0.1"}); err != nil {
		panic(err)
	}

	// The jwt middleware.
	authMiddleware, err := jwt.New(s.initJWTParams())
	if err != nil {
		log.Fatal("JWT Error:" + err.Error())
	}

	// Serve frontend static files
	s.app.Use(static.Serve(s.baseListenPrefix, static.LocalFile("./dist", true)))
	s.logger.Debug("Attached static middleware.")
	s.app.Use(s.jwtHandlerMiddleWare(authMiddleware))
	s.logger.Debug("Attached auth middleware.")
	s.app.Use(gin.Logger())
	s.logger.Debug("Attached logger middleware.")
	s.app.Use(cors.Default())
	s.logger.Debug("Attached CORS middleware.")
	s.app.Use(s.ErrorHandlerMiddleware)
	s.logger.Debug("Attached error-handler middleware.")

	////////////////////////
	// Prometheus metrics //
	////////////////////////
	s.app.GET(s.prometheusEndpoint, s.HandlePrometheusRequest)

	////////////////////////
	// Websocket Handlers //
	////////////////////////
	s.app.GET(s.getPath(domain.WorkloadEndpoint), s.workloadManager.GetWorkloadWebsocketHandler())
	s.app.GET(s.getPath(domain.LogsEndpoint), s.serveLogWebsocket)
	s.app.GET(s.getPath(domain.GeneralWebsocketEndpoint), s.serveGeneralWebsocket)

	// TODO: Getting nil pointer exception because the callback occurs in the constructor, so s.gatewayRpcClient is still nil.
	s.gatewayRpcClient = handlers.NewClusterDashboardHandler(s.opts, true, s.notifyFrontend, s.handleRpcRegistrationComplete)

	s.sugaredLogger.Debugf("Creating route groups now. (gatewayRpcClient == nil: %v)", s.gatewayRpcClient == nil)

	pprof.Register(s.app, s.getPath("dev/pprof"))

	s.app.NoRoute(authMiddleware.MiddlewareFunc(), func(c *gin.Context) {
		claims := jwt.ExtractClaims(c)
		log.Printf("NoRoute claims: %#v\n", claims)
		c.JSON(404, gin.H{"code": "PAGE_NOT_FOUND", "message": "Page not found"})
	})

	// Used by frontend to authenticate and get access to the dashboard.
	s.app.POST(s.getPath(domain.AuthenticateRequest), func(c *gin.Context) {
		request, err := httputil.DumpRequest(c.Request, true)
		if err != nil {
			s.logger.Error("Failed to dump JWT login request.", zap.Error(err))
		}

		s.sugaredLogger.Debugf("JWT login handler called: \"%s\": %s", s.getPath(domain.AuthenticateRequest), request)
		authMiddleware.LoginHandler(c)
	})

	s.app.POST(s.getPath(domain.RefreshToken), func(c *gin.Context) {
		s.sugaredLogger.Debugf("JWT token refresh handler called: \"%s\"", s.getPath(domain.RefreshToken))
		authMiddleware.RefreshHandler(c)
	})

	///////////////////////////////
	// Standard/Primary Handlers //
	///////////////////////////////
	apiGroup := s.app.Group(s.getPath(domain.BaseApiGroupEndpoint), authMiddleware.MiddlewareFunc())
	{
		// Used internally (by the frontend) to get the current kubernetes nodes from the backend  (i.e., the backend).
		apiGroup.GET(domain.NodesEndpoint, s.nodeHandler.HandleRequest)

		// Enable/disable Kubernetes nodes.
		apiGroup.PATCH(domain.NodesEndpoint, s.nodeHandler.HandlePatchRequest)

		// Adjust vGPUs available on a particular Kubernetes node.
		apiGroup.PATCH(domain.AdjustVgpusEndpoint, handlers.NewAdjustVirtualGpusHandler(s.opts, s.gatewayRpcClient).HandlePatchRequest)

		// Used internally (by the frontend) to get the system config from the backend  (i.e., the backend).
		apiGroup.GET(domain.SystemConfigEndpoint, handlers.NewConfigHttpHandler(s.opts).HandleRequest)

		// Used internally (by the frontend) to get the current set of Jupyter kernels from us (i.e., the backend).
		apiGroup.GET(domain.GetKernelsEndpoint, handlers.NewKernelHttpHandler(s.opts, s.gatewayRpcClient).HandleRequest)

		// Used by the frontend to query the status of particular ZMQ messages.
		apiGroup.POST(domain.QueryMessageEndpoint, handlers.NewMessageQueryHttpHandler(s.opts, s.gatewayRpcClient).HandleRequest)

		// Used internally (by the frontend) to get the list of available workload presets from the backend.
		apiGroup.GET(domain.WorkloadPresetEndpoint, handlers.NewWorkloadPresetHttpHandler(s.opts).HandleRequest)

		// Used internally (by the frontend) to trigger kernel replica migrations.
		apiGroup.POST(domain.MigrationEndpoint, handlers.NewMigrationHttpHandler(s.opts, s.gatewayRpcClient).HandleRequest)

		// Used to stream logs from Kubernetes.
		apiGroup.GET(fmt.Sprintf("%s/pods/:pod", domain.LogsEndpoint), handlers.NewLogHttpHandler(s.opts).HandleRequest)

		// Queried by Grafana to query for values used to create Grafana variables that are then used to
		// dynamically create a Grafana Dashboard.
		apiGroup.GET(path.Join(domain.VariablesEndpoint, ":variable_name"), handlers.NewVariablesHttpHandler(s.opts, s.gatewayRpcClient).HandleRequest)

		// Used by the frontend to tell a kernel to stop training.
		apiGroup.POST(domain.StopTrainingEndpoint, handlers.NewStopTrainingHandler(s.opts, s.atom).HandleRequest)

		// Used by the frontend to upload/share Prometheus metrics.
		apiGroup.PATCH(domain.MetricsEndpoint, handlers.NewMetricsHttpHandler(s.opts).HandlePatchRequest)

		// Used by the frontend to retrieve the UnixMillisecond timestamp at which the Cluster was created.
		apiGroup.GET(domain.ClusterAgeEndpoint, handlers.NewClusterAgeHttpHandler(s.opts, s.gatewayRpcClient).HandleRequest)
	}

	///////////////////////////
	// Debugging and Testing //
	///////////////////////////
	{
		apiGroup.POST(domain.YieldNextRequestEndpoint, handlers.NewYieldNextExecuteHandler(s.opts, s.gatewayRpcClient).HandleRequest)

		apiGroup.POST(domain.PanicEndpoint, handlers.NewPanicHttpHandler(s.opts, s.gatewayRpcClient).HandleRequest)

		apiGroup.POST(domain.SpoofNotificationsEndpoint, s.handleSpoofedNotifications)

		apiGroup.POST(domain.SpoofErrorEndpoint, s.handleSpoofedError)

		apiGroup.POST(domain.PingKernelEndpoint, handlers.NewPingKernelHttpHandler(s.opts, s.gatewayRpcClient).HandleRequest)
	}

	/////////////////////
	// Jupyter Handler // This isn't really used anymore...
	/////////////////////
	if s.opts.SpoofKernelSpecs {
		jupyterGroup := s.app.Group(s.getPath(domain.JupyterGroupEndpoint))
		{
			jupyterGroup.GET(domain.BaseApiGroupEndpoint+domain.KernelSpecEndpoint, handlers.NewJupyterAPIHandler(s.opts).HandleGetKernelSpecRequest)
		}
	}

	gin.SetMode(gin.DebugMode)

	return nil
}

func (s *serverImpl) HandleAuthenticateRequest(c *gin.Context) {
	var login *auth.LoginRequest
	err := c.BindJSON(&login)
	if err != nil {
		_ = c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	if login.Username == s.adminUsername && login.Password == s.adminPassword {
		s.logger.Debug("Authenticated.")
		c.Status(http.StatusOK)
		return
	} else {
		s.logger.Warn("Received invalid authentication attempt.")
		c.Status(http.StatusBadRequest)
		return
	}
}

// HandlePrometheusRequest passes the request directly to the http.Handler returned by promhttp.Handler.
func (s *serverImpl) HandlePrometheusRequest(c *gin.Context) {
	s.prometheusHandler.ServeHTTP(c.Writer, c.Request)
}

func (s *serverImpl) notifyFrontend(notification *gateway.Notification) {
	message := &domain.GeneralWebSocketResponse{
		Op:      "notification",
		Payload: notification,
	}

	toRemove := make([]string, 0)

	for remoteIp, conn := range s.generalWebsockets {
		s.logger.Debug("Writing message to general WebSocket.", zap.String("remote-addr", remoteIp))
		err := conn.WriteJSON(message)
		if err != nil {
			s.logger.Debug("Failed to write spoofed error to WebSocket.", zap.String("remote-addr", remoteIp), zap.Error(err))

			var closeError *websocket.CloseError
			if errors.As(err, &closeError) || errors.Is(err, websocket.ErrCloseSent) {
				s.logger.Debug("Will remove general WebSocket.", zap.String("remote-addr", remoteIp))
				toRemove = append(toRemove, remoteIp)
			}
		} else {
			s.logger.Debug("Successfully wrote message to general WebSocket.", zap.String("remote-addr", remoteIp))
		}
	}

	for _, remoteIp := range toRemove {
		s.logger.Warn("Removing general WebSocket connection.", zap.String("remote_ip", remoteIp))

		ws := s.generalWebsockets[remoteIp]
		err := ws.Close()
		if err != nil {
			s.logger.Error("Error closing websocket.", zap.String("remote_ip", remoteIp), zap.Error(err))
		}

		delete(s.generalWebsockets, remoteIp)
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
		_ = ctx.AbortWithError(http.StatusInternalServerError, err)
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

func (s *serverImpl) serveGeneralWebsocket(c *gin.Context) {
	s.logger.Debug("Inspecting origin of incoming non-specific WebSocket connection.",
		zap.String("request-origin", c.Request.Header.Get("Origin")),
		zap.String("request-host", c.Request.Host), zap.String("request-uri", c.Request.RequestURI))

	upgrader.CheckOrigin = func(r *http.Request) bool {
		incomingOrigin := r.Header.Get("Origin")
		for _, expectedOrigin := range s.expectedOriginAddresses {
			if incomingOrigin == expectedOrigin {
				return true
			}
		}

		s.logger.Error("Incoming non-specific WebSocket connection had unexpected origin. Rejecting.",
			zap.String("request-origin", c.Request.Header.Get("Origin")),
			zap.String("request-host", c.Request.Host), zap.String("request-uri", c.Request.RequestURI))
		return false
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer func(conn *websocket.Conn) {
		err := conn.Close()
		if err != nil {
			s.logger.Error("Failed to close WebSocket connection.", zap.Error(err))
		}
	}(conn)

	var concurrentConn domain.ConcurrentWebSocket = concurrent_websocket.NewConcurrentWebSocket(conn)
	remoteIp := concurrentConn.RemoteAddr().String()
	s.generalWebsockets[remoteIp] = concurrentConn

	for {
		_, message, err := concurrentConn.ReadMessage()
		if err != nil {
			s.logger.Error("Error while reading message from general websocket.", zap.Error(err))
			delete(s.generalWebsockets, remoteIp)
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

		var opVal interface{}
		var msgIdVal interface{}
		var ok bool
		if opVal, ok = request["op"]; !ok {
			s.logger.Error("Received unexpected message on websocket. It did not contain an 'op' field.", zap.Binary("message", message))
			time.Sleep(time.Millisecond * 100)
			continue
		}

		if msgIdVal, ok = request["msg_id"]; !ok {
			s.logger.Error("Received unexpected message on websocket. It did not contain a 'msg_id' field.", zap.Binary("message", message))
			time.Sleep(time.Millisecond * 100)
			continue
		}

		s.logger.Debug("Received general WebSocket message.", zap.Any("op", opVal), zap.Any("message-id", msgIdVal))

		// var resp map[string]string = make(map[string]string)
		// resp["message"] = fmt.Sprintf("Hello there, WebSocket %s.", remote_ip)
		// conn.WriteJSON(resp)
	}
}

func (s *serverImpl) serveLogWebsocket(c *gin.Context) {
	s.logger.Debug("Inspecting origin of incoming log-related WebSocket connection.",
		zap.String("request-origin", c.Request.Header.Get("Origin")),
		zap.String("request-host", c.Request.Host), zap.String("request-uri", c.Request.RequestURI))

	upgrader.CheckOrigin = func(r *http.Request) bool {
		incomingOrigin := r.Header.Get("Origin")
		for _, expectedOrigin := range s.expectedOriginAddresses {
			if incomingOrigin == expectedOrigin {
				return true
			}
		}

		s.logger.Error("Incoming log-related WebSocket connection had unexpected origin. Rejecting.",
			zap.String("request-origin", c.Request.Header.Get("Origin")),
			zap.String("request-host", c.Request.Host), zap.String("request-uri", c.Request.RequestURI))
		return false
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer func(conn *websocket.Conn) {
		err := conn.Close()
		if err != nil {
			s.logger.Error("Failed to close WebSocket connection.", zap.Error(err))
		}
	}(conn)

	var connectionId = uuid.NewString()
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			s.logger.Error("Error while reading message from websocket.", zap.String("connection-id", connectionId), zap.String("error-message", err.Error()))

			s.logResponseBodyMutex.RLock()
			responseBody, ok := s.getLogsResponseBodies[connectionId]
			s.logResponseBodyMutex.RUnlock()
			// If we're already processing a get_logs request for this websocket, then terminate that request.
			if ok {
				if err := responseBody.Close(); err != nil {
					s.logger.Error("Failed to close logs response body.", zap.Error(err))
				}
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
				if err := responseBody.Close(); err != nil {
					s.logger.Error("Failed to close logs response body.", zap.Error(err))
				}
			}
			s.logResponseBodyMutex.RUnlock()

			time.Sleep(time.Millisecond * 100)
			continue
		}

		s.sugaredLogger.Debugf("Received log-related WebSocket message: %v", request)

		var opVal interface{}
		var ok bool
		if opVal, ok = request["op"]; !ok {
			s.logger.Error("Received unexpected message on websocket. It did not contain an 'op' field.", zap.Binary("message", message), zap.String("connection-id", connectionId))

			s.logResponseBodyMutex.RLock()
			// If we're already processing a get_logs request for this websocket, then terminate that request.
			if responseBody, ok := s.getLogsResponseBodies[connectionId]; ok {
				if err := responseBody.Close(); err != nil {
					s.logger.Error("Failed to close logs response body.", zap.Error(err))
				}
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
				if err := responseBody.Close(); err != nil {
					s.logger.Error("Failed to close logs response body.", zap.Error(err))
				}
			}
			s.logResponseBodyMutex.RUnlock()

			time.Sleep(time.Millisecond * 100)
			continue
		}

		if opVal == "get_logs" {
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
	s.logger.Debug("Retrieving logs.", zap.Any("request", req), zap.String("connection-id", connectionId))

	pod := req.Pod
	container := req.Container
	doFollow := req.Follow

	endpoint := fmt.Sprintf("http://localhost:8889/api/v1/namespaces/default/pods/%s/log?container=%s&follow=%v&sinceSeconds=3600", pod, container, doFollow)
	s.logger.Debug("Retrieving logs now.", zap.String("pod", pod), zap.String("container", container), zap.String("endpoint", endpoint), zap.String("connection-id", connectionId))
	resp, err := http.Get(endpoint)
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
		if err := responseBody.Close(); err != nil {
			s.logger.Error("Failed to close logs response body.", zap.Error(err))
		}
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

// Serve is a blocking call that launches additional goroutines to serve the HTTP server and the Jupyter WebSocket proxy.
func (s *serverImpl) Serve() error {
	var wg sync.WaitGroup
	wg.Add(3)

	s.serveHttp(&wg)
	s.serveJupyterWebSocketProxy(&wg)

	wg.Wait()
	return nil
}

func (s *serverImpl) serveHttp(wg *sync.WaitGroup) {
	s.logger.Debug("Listening for HTTP requests.", zap.String("address", fmt.Sprintf(":%d", s.opts.ServerPort)))
	go func() {
		addr := fmt.Sprintf(":%d", s.opts.ServerPort)
		if err := http.ListenAndServe(addr, s.app); err != nil {
			s.sugaredLogger.Errorf("HTTP Server failed to listen on '%s'. Error: %v", addr, err)
			panic(err)
		}

		wg.Done()
	}()
}

func (s *serverImpl) serveJupyterWebSocketProxy(wg *sync.WaitGroup) {
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
