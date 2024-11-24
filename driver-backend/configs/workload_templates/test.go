package main

import (
	"encoding/json"
	"fmt"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	"io"
	"log"
	"os"
	"time"
)

type WorkloadRegistrationRequest struct {
	AdjustGpuReservations     bool                              `json:"adjustGpuReservations"`
	WorkloadName              string                            `json:"name"`
	DebugLogging              bool                              `json:"debugLogging"`
	TemplateFilePath          string                            `json:"template_file_path"`
	Type                      string                            `json:"type"`
	Key                       string                            `json:"key"`
	Seed                      int64                             `json:"seed"`
	TimescaleAdjustmentFactor float64                           `json:"timescaleAdjustmentFactor"`
	SessionsSamplePercentage  float64                           `json:"sessionsSamplePercentage"`
	Sessions                  []*domain.WorkloadTemplateSession `json:"sessions"`
}

func (r *WorkloadRegistrationRequest) String() string {
	m, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		panic(err)
	}

	return string(m)
}

func main() {
	fmt.Println("Reading file now.")
	st := time.Now()
	// Open the JSON file
	file, err := os.Open("trace_summer.json")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// Read the file contents
	data, err := io.ReadAll(file)
	if err != nil {
		log.Fatalf("failed to read file: %v", err)
	}

	fmt.Printf("Read file in %v.\n", time.Since(st))
	st2 := time.Now()

	// Unmarshal JSON into the struct
	var result *WorkloadRegistrationRequest
	err = json.Unmarshal(data, &result)
	if err != nil {
		log.Fatalf("failed to unmarshal JSON: %v", err)
	}

	fmt.Printf("Decoded file in %v.\n", time.Since(st2))

	fmt.Printf("Num Sessions: %d\n", len(result.Sessions))

	fmt.Println("Finished processing Sessions.")
}
