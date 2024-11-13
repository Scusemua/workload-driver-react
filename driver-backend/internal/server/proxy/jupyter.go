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
	// If we receive a request whose path is prefixed by ContextPath,
	// then we'll forward that request to the Jupyter Server.
	//
	// So, a request that should be forwarded will have the path OurServerBasePath/ContextPath/<whatever>
	ContextPath string

	// JupyterServerAddress is the address of the Jupyter Server.
	JupyterServerAddress string

	// OurServerBasePath is the base path prefix that all requests we receive should have, or else we 404.
	//
	// So, a request that should be forwarded will have the path OurServerBasePath/ContextPath/<whatever>
	OurServerBasePath string

	// JupyterServerBasePath is the base URL that the Jupyter Server is listening on.
	// We have to carefully update request URLs to respect JupyterServerBasePath.
	//
	// For example, if we receive a request with path OurServerBasePath/ContextPath/<whatever>, then we will forward it.
	// And we will change the path of that request to: JupyterServerBasePath/<whatever>.
	JupyterServerBasePath string

	// If the ContextPath and JupyterServerBasePath are identical, then we can take a shortcut when
	// updating the requests during request-forwarding.
	//
	// Specifically, requests that are to be forwarded will have the path:
	// OurServerBasePath/ContextPath/<whatever>
	//
	// They will need to be changed to have the path:
	// JupyterServerBasePath/<whatever>
	//
	// If ContextPath and JupyterServerBasePath are equal, then we can simply trim the OurServerBasePath prefix.
	//
	// So, if JustTrimOurBasePathPrefix is true, then we can take that shortcut. Otherwise, we have to first remove
	// OurServerBasePath/ContextPath from the request path before adding the new JupyterServerBasePath prefix.
	JustTrimOurBasePathPrefix bool

	*gin.Engine

	logger *zap.Logger
}

func NewJupyterProxyRouter(engine *gin.Engine, config *domain.Configuration, atom *zap.AtomicLevel) *JupyterProxyRouter {
	proxyRouter := &JupyterProxyRouter{
		Engine:                engine,
		ContextPath:           domain.JupyterGroupEndpoint,
		JupyterServerAddress:  config.InternalJupyterServerAddress,
		JupyterServerBasePath: config.JupyterServerBasePath,
		OurServerBasePath:     config.BaseUrl,
	}

	// If the ContextPath and JupyterServerBasePath are identical, then we can take a shortcut when
	// updating the requests during request-forwarding.
	//
	// Specifically, requests that are to be forwarded will have the path:
	// OurServerBasePath/ContextPath/<whatever>
	//
	// They will need to be changed to have the path:
	// JupyterServerBasePath/<whatever>
	//
	// If ContextPath and JupyterServerBasePath are equal, then we can simply trim the OurServerBasePath prefix.
	if proxyRouter.ContextPath == config.JupyterServerBasePath {
		proxyRouter.JustTrimOurBasePathPrefix = true
	}

	zapConfig := zap.NewDevelopmentEncoderConfig()
	zapConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	core := zapcore.NewCore(zapcore.NewConsoleEncoder(zapConfig), zapcore.AddSync(colorable.NewColorableStdout()), atom)
	logger := zap.New(core, zap.Development())
	if logger == nil {
		panic("failed to create logger for workload driver")
	}

	proxyRouter.logger = logger

	return proxyRouter
}

// adjustRequest adjusts the request's RequestURI and URL.Path.
//
// It returns the updated request (which is modified in-place, to be clear) as well as the original requestURI
// and URL.Path values.
func (r *JupyterProxyRouter) adjustRequest(req *http.Request) (*http.Request, string, string) {
	originalRequestURI := req.RequestURI
	originalUrlPath := req.URL.Path

	if r.JustTrimOurBasePathPrefix {
		req = r.adjustRequestShortcut(req)
	} else {
		req = r.adjustRequestTheLongWay(req)
	}

	return req, originalRequestURI, originalUrlPath
}

// adjustRequestTheLongWay adjusts the request's URL.Path and RequestURI by first removing the
// OurServerBasePath/ContextPath prefix before adding the new JupyterServerBasePath prefix.
//
// For an explanation, see the documentation of the JustTrimOurBasePathPrefix field of the JupyterProxyRouter struct.
func (r *JupyterProxyRouter) adjustRequestTheLongWay(req *http.Request) *http.Request {
	// First, remove both OurServerBasePath and ContextPath from the RequestURI and the URL.Path.
	req.RequestURI = strings.TrimPrefix(req.RequestURI, path.Join(r.OurServerBasePath, r.ContextPath))
	req.URL.Path = strings.TrimPrefix(req.URL.Path, path.Join(r.OurServerBasePath, r.ContextPath))

	// Next, add the JupyterServerBasePath as a prefix to the RequestURI and the URL.Path.
	req.RequestURI = path.Join(r.JupyterServerBasePath, req.RequestURI)
	req.URL.Path = path.Join(r.JupyterServerBasePath, req.URL.Path)

	return req
}

// adjustRequestShortcut adjusts the request's URL.Path and RequestURI by simply removing the OurServerBasePath prefix.
//
// For an explanation, see the documentation of the JustTrimOurBasePathPrefix field of the JupyterProxyRouter struct.
func (r *JupyterProxyRouter) adjustRequestShortcut(req *http.Request) *http.Request {
	// Simply remove the 'OurServerBasePath' prefix from the request's RequestURI and URL.Path.
	req.RequestURI = strings.TrimPrefix(req.RequestURI, r.OurServerBasePath)
	req.URL.Path = strings.TrimPrefix(req.URL.Path, r.OurServerBasePath)

	return req
}

func (r *JupyterProxyRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	fmt.Printf("req.RequestURI: %s. req.Method:%v\n", req.RequestURI, req.Method)

	// If the request is NOT prefixed by __BASE_URL__/{{ context_path }}, then serve it normally.
	// So, this may mean that if a request's path has the prefix /dashboard/jupyter, then we will forward it.
	// Otherwise, we serve the request normally.
	if !strings.HasPrefix(req.RequestURI, path.Join(r.OurServerBasePath, r.ContextPath)) {
		r.Engine.ServeHTTP(w, req)
	} else {
		// If we're here, then the request path has the ContextPath prefix.
		updatedReq, originalRequestURI, originalUrlPath := r.adjustRequest(req)

		//r.logger.Debug("Proxying request to Jupyter.",
		//	zap.String("original_request_uri", originalRequestURI),
		//	zap.String("updated_request_uri", updatedReq.RequestURI),
		//	zap.String("original_request_path", originalUrlPath),
		//	zap.String("updated_request_path", updatedReq.URL.Path),
		//	zap.String("request_method", updatedReq.Method),
		//	zap.String("jupyter_server_address", r.JupyterServerAddress),
		//	zap.String("jupyter_server_base_path", r.JupyterServerBasePath),
		//	zap.String("request_host", req.URL.Host))

		director := func(request *http.Request) {
			request.URL.Scheme = "http"
			request.URL.Host = r.JupyterServerAddress
		}
		proxy := &httputil.ReverseProxy{
			Director: director,
			ErrorHandler: func(writer http.ResponseWriter, request *http.Request, err error) {
				r.logger.Error("ErrorHandler called for Jupyter ReverseProxy.",
					zap.String("original_request_uri", originalRequestURI),
					zap.String("updated_request_uri", updatedReq.RequestURI),
					zap.String("original_request_path", originalUrlPath),
					zap.String("updated_request_path", updatedReq.URL.Path),
					zap.String("request_method", updatedReq.Method),
					zap.String("jupyter_server_address", r.JupyterServerAddress),
					zap.String("jupyter_server_base_path", r.JupyterServerBasePath),
					zap.String("request_host", req.URL.Host),
					zap.Error(err))

				// Default error handler just logs the error and returns HTTP 502 Bad Gateway, so we'll do that.
				writer.WriteHeader(http.StatusBadGateway)
			},
		}
		proxy.ServeHTTP(w, updatedReq)
	}
}
