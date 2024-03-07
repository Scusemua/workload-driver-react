package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/gin-gonic/contrib/cors"
	"github.com/gin-gonic/contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/koding/websocketproxy"

	"github.com/scusemua/workload-driver-react/m/v2/internal/server/config"
	"github.com/scusemua/workload-driver-react/m/v2/internal/server/domain"
	"github.com/scusemua/workload-driver-react/m/v2/internal/server/handlers"
	"github.com/scusemua/workload-driver-react/m/v2/internal/server/proxy"
)

func main() {
	conf := config.GetConfiguration()

	// Load ENV from .env file
	err := godotenv.Load()
	if err != nil {
		panic("error loading .env file")
	}

	app := &proxy.JupyterProxyRouter{
		ContextPath:  domain.JUPYTER_GROUP_ENDPOINT,
		Start:        len(domain.JUPYTER_GROUP_ENDPOINT),
		Config:       conf,
		SpoofJupyter: conf.SpoofKernelSpecs,
		Engine:       gin.New(),
	}

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

		// Used internally (by the frontend) to get the list of available workload presets from the backend.
		apiGroup.GET(domain.WORKLOAD_PRESET_ENDPOINT, handlers.NewWorkloadPresetHttpHandler(conf).HandleRequest)

		// Used internally (by the frontend) to trigger kernel replica migrations.
		apiGroup.GET(domain.MIGRATION_ENDPOINT, handlers.NewMigrationHttpHandler(conf).HandleRequest)
	}

	if conf.SpoofKernelSpecs {
		jupyterGroup := app.Group(domain.JUPYTER_GROUP_ENDPOINT)
		{
			jupyterGroup.GET(domain.BASE_API_GROUP_ENDPOINT+domain.KERNEL_SPEC_ENDPOINT, handlers.NewJupyterAPIHandler(conf).HandleGetKernelSpecRequest)
		}
	}

	fmt.Printf("Listening for HTTP requests on '%s'\n", fmt.Sprintf("127.0.0.1:%d", conf.ServerPort))
	go http.ListenAndServe(fmt.Sprintf(":%d", conf.ServerPort), app)

	wsUrlString := fmt.Sprintf("ws://%s", conf.JupyterServerAddress)
	wsUrl, err := url.Parse(wsUrlString)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Listening for Websocket Connections on '%s' and proxying them to '%s'\n", fmt.Sprintf("127.0.0.1:%d", conf.WebsocketProxyPort), wsUrl)
	err = http.ListenAndServe(fmt.Sprintf("127.0.0.1:%d", conf.WebsocketProxyPort), websocketproxy.NewProxy(wsUrl))
	if err != nil {
		log.Fatalln(err)
	}
}
