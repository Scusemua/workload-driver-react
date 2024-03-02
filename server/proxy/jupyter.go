package proxy

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/scusemua/workload-driver-react/m/v2/server/config"
)

type JupyterProxyRouter struct {
	ContextPath string
	Start       int
	Config      *config.Configuration
	*gin.Engine
}

func (r *JupyterProxyRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	fmt.Printf("req.RequestURI: %s\n", req.RequestURI)
	if strings.HasPrefix(req.RequestURI, r.ContextPath) {
		fmt.Printf("request url:%s, will skip prefix:%s\n", req.RequestURI, r.ContextPath)
		req.RequestURI = req.RequestURI[r.Start:]
		req.URL.Path = req.URL.Path[r.Start:]

		director := func(req *http.Request) {
			req.URL.Scheme = "http"
			req.URL.Host = r.Config.JupyterServerAddress
		}
		proxy := &httputil.ReverseProxy{Director: director}
		proxy.ServeHTTP(w, req)
	} else {
		r.Engine.ServeHTTP(w, req)
	}
}
