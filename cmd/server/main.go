package main

import (
	"github.com/joho/godotenv"

	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	"github.com/scusemua/workload-driver-react/m/v2/internal/server"
)

func main() {
	conf := domain.GetDefaultConfig()
	conf.CheckUsage()

	// Load ENV from .env file
	err := godotenv.Load()
	if err != nil {
		panic("error loading .env file")
	}

	srv := server.NewServer(conf)

	// Blocking call.
	err = srv.Serve()
	if err != nil {
		panic(err)
	}
}
