package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/scusemua/workload-driver-react/m/v2/internal/server/jupyter"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	JupyterServerAddress string = "localhost:8888/"
)

func main() {
	kernelIdPtr := flag.String("kernel", "example-kernel-id", "The ID of the kernel to connect to")
	sessionIdPtr := flag.String("session", "example-session-id", "The ID of the associated session")
	flag.Parse()

	atom := zap.NewAtomicLevelAt(zapcore.DebugLevel)
	manager := jupyter.NewKernelSessionManager(JupyterServerAddress, &atom)

	fmt.Printf("Connecting to kernel %s, session %s.\n", *kernelIdPtr, *sessionIdPtr)

	kernelConnection, err := manager.ConnectTo(*kernelIdPtr, *sessionIdPtr, "")
	if err != nil {
		log.Fatalf("Failed to connect to the specified kernel. Error: %v\n", err)
	}

	err = kernelConnection.StopRunningTrainingCode(true)
	if err != nil {
		log.Fatalf("Failed to stop running training code on the specified kernel. Error: %v\n", err)
	}
}
