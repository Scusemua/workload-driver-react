package main

import (
	"github.com/joho/godotenv"
	"log"
	"os"

	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	"github.com/scusemua/workload-driver-react/m/v2/internal/server"
)

func main() {
	conf := domain.GetDefaultConfig()
	conf.CheckUsage()

	devEnvironment := os.Getenv("DEV_ENVIRONMENT")
	var environmentFileName string
	if devEnvironment == "production" {
		environmentFileName = ".production.env"
	} else {
		environmentFileName = ".development.env"
	}

	// Load ENV from .env file
	err := godotenv.Load(environmentFileName)
	if err != nil {
		log.Fatalf("Failed to load environment file \"%s\"", environmentFileName)
	}

	srv := server.NewServer(conf)

	// Blocking call.
	err = srv.Serve()
	if err != nil {
		panic(err)
	}
}
