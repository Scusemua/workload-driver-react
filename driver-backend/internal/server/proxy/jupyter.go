package proxy

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/mattn/go-colorable"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"net/http"
	"net/http/httputil"
	"path"
	"strings"
)

type JupyterProxyRouter struct {
	ContextPath          string
	JupyterServerAddress string
	BaseUrl              string
	*gin.Engine

	logger *zap.Logger
}

func NewJupyterProxyRouter(engine *gin.Engine, config *domain.Configuration, atom *zap.AtomicLevel) *JupyterProxyRouter {
	proxyRouter := &JupyterProxyRouter{
		Engine:               engine,
		ContextPath:          domain.JupyterGroupEndpoint,
		JupyterServerAddress: config.JupyterServerAddress,
		BaseUrl:              config.BaseUrl,
	}

	zapConfig := zap.NewDevelopmentEncoderConfig()
	zapConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	core := zapcore.NewCore(zapcore.NewConsoleEncoder(zapConfig), zapcore.AddSync(colorable.NewColorableStdout()), atom)
	logger := zap.New(core, zap.Development())
	if logger == nil {
		panic("failed to create logger for workload driver")
	}

	proxyRouter.logger = logger
	// proxyRouter.sugaredLogger = logger.Sugar()

	return proxyRouter
}

func (r *JupyterProxyRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	fmt.Printf("req.RequestURI: %s. req.Method:%v\n", req.RequestURI, req.Method)

	// If the request is NOT prefixed by __BASE_URL__/{{ context_path }}, then serve it normally.
	// So, this may mean that if a request's path has the prefix /dashboard/jupyter, then we will forward it.
	// Otherwise, we serve the request normally.
	if !strings.HasPrefix(req.RequestURI, path.Join(r.BaseUrl, r.ContextPath)) {
		r.Engine.ServeHTTP(w, req)
	} else {
		// If we're here, then we're not spoofing jupyter AND the request has the "/jupyter" prefix.
		req.RequestURI = strings.TrimPrefix(req.RequestURI, path.Join(r.BaseUrl, r.ContextPath))
		req.URL.Path = strings.TrimPrefix(req.URL.Path, path.Join(r.BaseUrl, r.ContextPath))

		r.logger.Debug("Proxying request to Jupyter.",
			zap.String("updated_request_uri", req.RequestURI),
			zap.String("updated_request_path", req.URL.Path),
			zap.String("request_method", req.Method),
			zap.String("jupyter_server_address", r.JupyterServerAddress))

		director := func(req *http.Request) {
			req.URL.Scheme = "http"
			req.URL.Host = r.JupyterServerAddress
		}
		proxy := &httputil.ReverseProxy{
			Director: director,
		}
		proxy.ServeHTTP(w, req)
	}
}
