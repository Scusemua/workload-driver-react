package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/gin-gonic/contrib/cors"
	"github.com/gin-gonic/contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"github.com/scusemua/workload-driver-react/m/v2/server/config"
	"github.com/scusemua/workload-driver-react/m/v2/server/domain"
	"github.com/scusemua/workload-driver-react/m/v2/server/handlers"
)

type JupyterProxyRouter struct {
	ContextPath string
	Start       int
	conf        *config.Configuration
	*gin.Engine
}

func main() {
	conf := config.GetConfiguration()

	// Load ENV from .env file
	err := godotenv.Load()
	if err != nil {
		panic("error loading .env file")
	}

	app := &JupyterProxyRouter{domain.JUPYTER_GROUP_ENDPOINT, len(domain.JUPYTER_GROUP_ENDPOINT), conf, gin.New()}

	app.ForwardedByClientIP = true
	app.SetTrustedProxies([]string{"127.0.0.1"})

	// Serve frontend static files
	app.Use(static.Serve("/", static.LocalFile("./dist", true)))
	app.Use(gin.Logger())
	app.Use(cors.Default())

	apiGroup := app.Group(domain.BASE_API_GROUP_ENDPOINT)
	{
		// Used internally (by the frontend) to get the current kubernetes nodes from the backend  (i.e., the backend).
		apiGroup.GET(domain.KUBERNETES_NODES_ENDPOINT, handlers.NewKubeNodeHttpHandler(conf).HandleRequest)

		// Used internally (by the frontend) to get the system config from the backend  (i.e., the backend).
		apiGroup.GET(domain.SYSTEM_CONFIG_ENDPOINT, handlers.NewConfigHttpHandler(conf).HandleRequest)

		// Used internally (by the frontend) to get the current set of Jupyter kernels from us (i.e., the backend).
		apiGroup.GET(domain.GET_KERNELS_ENDPOINT, handlers.NewKernelHttpHandler(conf).HandleRequest)
	}

	http.ListenAndServe(fmt.Sprintf("127.0.0.1:%d", conf.ServerPort), app)
}

func (r *JupyterProxyRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	fmt.Printf("req.RequestURI: %s\n", req.RequestURI)
	if strings.HasPrefix(req.RequestURI, r.ContextPath) {
		fmt.Printf("request url:%s, will skip prefix:%s\n", req.RequestURI, r.ContextPath)
		req.RequestURI = req.RequestURI[r.Start:]
		req.URL.Path = req.URL.Path[r.Start:]

		director := func(req *http.Request) {
			req.URL.Scheme = "http"
			req.URL.Host = r.conf.JupyterServerAddress
		}
		proxy := &httputil.ReverseProxy{Director: director}
		proxy.ServeHTTP(w, req)
	} else {
		r.Engine.ServeHTTP(w, req)
	}
}
