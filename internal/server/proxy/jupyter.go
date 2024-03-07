package proxy

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/scusemua/workload-driver-react/m/v2/internal/server/config"
)

type JupyterProxyRouter struct {
	ContextPath  string
	Start        int
	Config       *config.Configuration
	SpoofJupyter bool
	*gin.Engine
}

func (r *JupyterProxyRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	fmt.Printf("req.RequestURI: %s. req.Method:%v\n", req.RequestURI, req.Method)

	if r.SpoofJupyter || !strings.HasPrefix(req.RequestURI, r.ContextPath) {
		r.Engine.ServeHTTP(w, req)
	} else {
		// If we're here, then we're not spoofing jupyter AND the request has the "/jupyter" prefix.
		req.RequestURI = req.RequestURI[r.Start:]
		req.URL.Path = req.URL.Path[r.Start:]

		fmt.Printf("\tAdjusted RequestURI to \"%s\" and URL.Path to \"%s\" for %v request.\n", req.RequestURI, req.URL.Path, req.Method)

		director := func(req *http.Request) {
			req.URL.Scheme = "http"
			req.URL.Host = r.Config.JupyterServerAddress
		}
		proxy := &httputil.ReverseProxy{
			Director: director,
		}
		proxy.ServeHTTP(w, req)
	}
}
