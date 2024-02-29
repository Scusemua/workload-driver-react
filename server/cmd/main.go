package main

import (
	"fmt"

	"github.com/gin-gonic/contrib/cors"
	"github.com/gin-gonic/contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/roylisto/gin-golang-react/api"

	"github.com/scusemua/workload-driver-react/m/v2/server/config"
	"github.com/scusemua/workload-driver-react/m/v2/server/domain"
	"github.com/scusemua/workload-driver-react/m/v2/server/handlers"
)

var (
	router api.Router = api.NewRouter()
)

func main() {
	conf := config.GetConfiguration()
	
	// Load ENV from .env file
	err := godotenv.Load()
	if err != nil {
		panic("error loading .env file")
	}

	// Set the router as the default one shipped with Gin
	app := gin.New()

	app.ForwardedByClientIP = true
	app.SetTrustedProxies([]string{"127.0.0.1"})

	// Serve frontend static files
	app.Use(static.Serve("/", static.LocalFile("./dist", true)))
	app.Use(gin.Logger())
	app.Use(cors.Default())

	// Used internally (by the frontend) to get the current kubernetes nodes from the backend  (i.e., the backend).
	app.GET(domain.KUBERNETES_NODES_ENDPOINT, handlers.NewKubeNodeHttpHandler(conf).HandleRequest)

	// Used internally (by the frontend) to get the system config from the backend  (i.e., the backend).
	app.GET(domain.SYSTEM_CONFIG_ENDPOINT, handlers.NewConfigHttpHandler(conf).HandleRequest)

	// Used internally (by the frontend) to get the current set of Jupyter kernel specs from us (i.e., the backend).
	app.GET(domain.KERNEL_SPEC_ENDPOINT, handlers.NewKernelSpecHttpHandler(conf).HandleRequest)

	// Initialize the route
	router.SetupRouter(app)

	// Start and run the server on localhost as default
	app.Run(fmt.Sprintf("127.0.0.1:%d", conf.ServerPort))
}