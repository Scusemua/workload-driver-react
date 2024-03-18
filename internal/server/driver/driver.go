package driver

import (
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	"github.com/scusemua/workload-driver-react/m/v2/internal/generator"
	"github.com/scusemua/workload-driver-react/m/v2/internal/server/handlers"
	"github.com/scusemua/workload-driver-react/m/v2/internal/server/jupyter"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	ErrWorkloadPresetNotFound    = errors.New("could not find workload preset with specified key")
	ErrWorkloadAlreadyRegistered = errors.New("driver already has a workload registered with it")

	ErrTrainingStartedUnknownSession = errors.New("received 'training-started' event for unknown session")
	ErrTrainingStoppedUnknownSession = errors.New("received 'training-ended' event for unknown session")
)

// The Workload Driver consumes events from the Workload Generator and takes action accordingly.
type WorkloadDriver struct {
	id string // Unique ID (relative to other drivers). The workload registered with this driver will be assigned this ID.

	rpc           *handlers.GrpcClient
	kernelManager *jupyter.KernelManager

	logger        *zap.Logger
	sugaredLogger *zap.SugaredLogger
	atom          *zap.AtomicLevel

	kernels map[string]struct{}

	workloadEndTime time.Time         // The time at which the workload completed.
	eventChan       chan domain.Event // Receives events from the Synthesizer.

	workloadPresets             map[string]*domain.WorkloadPreset
	workloadPreset              *domain.WorkloadPreset
	workloadRegistrationRequest *domain.WorkloadRegistrationRequest
	workload                    *domain.Workload

	mu sync.Mutex

	opts      *domain.Configuration
	doneChan  chan struct{} // Used to signal that the workload has successfully processed all events.
	stopChan  chan struct{} // Used to stop the workload early/prematurely (i.e., before all events have been processed).
	errorChan chan error    // Used to stop the workload due to a critical error.
}

func NewWorkloadDriver(opts *domain.Configuration) *WorkloadDriver {
	atom := zap.NewAtomicLevelAt(zapcore.DebugLevel)
	driver := &WorkloadDriver{
		id:            uuid.NewString(),
		opts:          opts,
		doneChan:      make(chan struct{}),
		eventChan:     make(chan domain.Event),
		stopChan:      make(chan struct{}),
		errorChan:     make(chan error),
		atom:          &atom,
		rpc:           handlers.NewGrpcClient(opts, !opts.SpoofKernels),
		kernelManager: jupyter.NewKernelManager(opts, &atom),
		kernels:       make(map[string]struct{}),
	}

	core := zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewDevelopmentEncoderConfig()), os.Stdout, driver.atom)
	logger := zap.New(core)
	if logger == nil {
		panic("failed to create logger for workload driver")
	}

	driver.logger = logger
	driver.sugaredLogger = logger.Sugar()

	// Load the list of workload presets from the specified file.
	driver.logger.Debug("Loading workload presets from file now.", zap.String("filepath", opts.WorkloadPresetsFilepath))
	presets, err := domain.LoadWorkloadPresetsFromFile(opts.WorkloadPresetsFilepath)
	if err != nil {
		driver.logger.Error("Error encountered while loading workload presets from file now.", zap.String("filepath", opts.WorkloadPresetsFilepath), zap.Error(err))
		panic(err)
	}

	driver.workloadPresets = make(map[string]*domain.WorkloadPreset, len(presets))
	for _, preset := range presets {
		driver.workloadPresets[preset.Key] = preset

		driver.logger.Debug("Discovered preset.", zap.Any(fmt.Sprintf("preset-%s", preset.Key), preset.String()))
	}

	if !opts.SpoofKernels {
		err := driver.rpc.DialGatewayGRPC(opts.GatewayAddress)
		if err != nil {
			panic(fmt.Sprintf("WorkloadDriver %s failed to dial Cluster Gateway at addr='%s'", driver.id, opts.GatewayAddress))
		}
	}

	return driver
}

// Acquire the Driver's mutex externally.
//
// IMPORTANT: This will prevent the Driver's workload from progressing until the lock is released!
func (d *WorkloadDriver) LockDriver() {
	d.mu.Lock()
}

// Attempt to acquire the Driver's mutex externally.
// Returns true on successful acquiring of the lock. If lock was not acquired, return false.
//
// IMPORTANT: This will prevent the Driver's workload from progressing until the lock is released!
func (d *WorkloadDriver) TryLockDriver() bool {
	return d.mu.TryLock()
}

// Release the Driver's mutex externally.
func (d *WorkloadDriver) UnlockDriver() {
	d.mu.Unlock()
}

func (d *WorkloadDriver) ToggleDebugLogging(enabled bool) *domain.Workload {
	d.mu.Lock()
	defer d.mu.Unlock()

	if enabled {
		d.atom.SetLevel(zap.DebugLevel)
		d.workload.DebugLoggingEnabled = true
	} else {
		d.atom.SetLevel(zap.InfoLevel)
		d.workload.DebugLoggingEnabled = false
	}

	return d.workload
}

func (d *WorkloadDriver) GetWorkload() *domain.Workload {
	return d.workload
}

func (d *WorkloadDriver) GetWorkloadPreset() *domain.WorkloadPreset {
	return d.workloadPreset
}

func (d *WorkloadDriver) GetWorkloadRegistrationRequest() *domain.WorkloadRegistrationRequest {
	return d.workloadRegistrationRequest
}

// Returns nil if the workload could not be registered.
func (d *WorkloadDriver) RegisterWorkload(workloadRegistrationRequest *domain.WorkloadRegistrationRequest, conn *websocket.Conn) (*domain.Workload, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Only one workload per driver.
	if d.workload != nil {
		return nil, ErrWorkloadAlreadyRegistered
	}

	d.workloadRegistrationRequest = workloadRegistrationRequest
	d.sugaredLogger.Debugf("User is requesting the execution of workload '%s'", workloadRegistrationRequest.Key)

	var ok bool
	if d.workloadPreset, ok = d.workloadPresets[workloadRegistrationRequest.Key]; !ok {
		d.logger.Error("Could not find workload preset with specified key.", zap.String("key", workloadRegistrationRequest.Key))

		conn.WriteJSON(&domain.ErrorMessage{
			Description:  "Could not find workload preset with specified key.",
			ErrorMessage: "Could not find workload preset with specified key.",
			Valid:        true,
		})
		return nil, ErrWorkloadPresetNotFound
	}

	if !workloadRegistrationRequest.DebugLogging {
		d.logger.Debug("Setting log-level to INFO.")
		d.atom.SetLevel(zapcore.InfoLevel)
		d.logger.Debug("Ignored.")
	} else {
		d.logger.Debug("Debug-level logging is ENABLED.")
	}

	d.workload = &domain.Workload{
		ID:                  d.id, // Same ID as the driver.
		Name:                workloadRegistrationRequest.WorkloadName,
		WorkloadState:       domain.WorkloadReady,
		WorkloadPreset:      d.workloadPreset,
		WorkloadPresetName:  d.workloadPreset.Name,
		WorkloadPresetKey:   d.workloadPreset.Key,
		TimeElasped:         time.Duration(0).String(),
		Seed:                d.workloadRegistrationRequest.Seed,
		RegisteredTime:      time.Now(),
		NumTasksExecuted:    0,
		NumEventsProcessed:  0,
		NumSessionsCreated:  0,
		NumActiveSessions:   0,
		NumActiveTrainings:  0,
		DebugLoggingEnabled: workloadRegistrationRequest.DebugLogging,
	}

	// If the workload seed is negative, then assign it a random value.
	if d.workloadRegistrationRequest.Seed < 0 {
		d.workload.Seed = rand.Int63n(2147483647) // We restrict the user to the range 0-2,147,483,647 when they specify a seed.
		d.logger.Debug("Will use random seed for RNG.", zap.Int64("seed", d.workload.Seed))
	} else {
		d.logger.Debug("Will use user-specified seed for RNG.", zap.Int64("seed", d.workload.Seed))
	}

	return d.workload, nil
}

// Write an error back to the client.
func (d *WorkloadDriver) WriteError(c *gin.Context, errorMessage string) {
	// Write error back to front-end.
	msg := &domain.ErrorMessage{
		ErrorMessage: errorMessage,
		Valid:        true,
	}
	c.JSON(http.StatusInternalServerError, msg)
}

// Return true if the workload has completed; otherwise, return false.
func (d *WorkloadDriver) IsWorkloadComplete() bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	return d.workload.WorkloadState == domain.WorkloadFinished
}

func (d *WorkloadDriver) ID() string {
	return d.id
}

// Stop a workload that's already running/in-progress.
// Returns nil on success, or an error if one occurred.
func (d *WorkloadDriver) StopWorkload() error {
	d.logger.Debug("Stopping workload.", zap.String("workload-id", d.id))
	d.stopChan <- struct{}{}

	d.mu.Lock()
	defer d.mu.Unlock()

	d.workloadEndTime = time.Now()
	d.workload.WorkloadState = domain.WorkloadTerminated
	return nil
}

func (d *WorkloadDriver) StopChan() chan struct{} {
	return d.stopChan
}

// This should be called from its own goroutine.
// Accepts a waitgroup that is used to notify the caller when the workload has entered the 'WorkloadRunning' state.
func (d *WorkloadDriver) DriveWorkload(wg *sync.WaitGroup) {
	d.logger.Info("Starting workload.", zap.Any("workload-preset", d.workloadPreset), zap.Any("workload-request", d.workloadRegistrationRequest))

	workloadGenerator := generator.NewWorkloadGenerator(d.opts)
	go workloadGenerator.GenerateWorkload(d, d.workload, d.workloadPreset, d.workloadRegistrationRequest)

	d.mu.Lock()
	d.workload.StartTime = time.Now()
	d.workload.WorkloadState = domain.WorkloadRunning
	d.mu.Unlock()

	wg.Done()

	d.logger.Info("The Workload Driver has started running.")

	for {
		select {
		case err := <-d.errorChan:
			{
				d.logger.Error("Workload encountered a critical error and must abort.", zap.String("workload-id", d.id), zap.Error(err))
				workloadGenerator.StopGeneratingWorkload()

				d.mu.Lock()
				defer d.mu.Unlock()

				d.workload.WorkloadState = domain.WorkloadErred
				d.workload.Error = err
				return
			}
		case <-d.stopChan:
			{
				d.logger.Info("Workload has been instructed to terminate early.")
				workloadGenerator.StopGeneratingWorkload()
				return
			}
		case evt := <-d.eventChan:
			{
				// d.logger.Debug("Received event.", zap.Any("event-name", evt.Name()))
				d.handleEvent(evt)

				d.mu.Lock()
				defer d.mu.Unlock()

				d.workload.NumEventsProcessed += 1
				d.workload.TimeElasped = time.Since(d.workload.StartTime).String()

				return
			}
		case <-d.doneChan:
			{
				d.mu.Lock()
				defer d.mu.Unlock()

				d.workloadEndTime = time.Now()
				d.workload.WorkloadState = domain.WorkloadFinished

				d.logger.Info("The Workload Generator has finished generating events.")
				d.logger.Info("The Workload has ended.", zap.Any("workload-duration", time.Since(d.workload.StartTime)), zap.Any("workload-start-time", d.workload.StartTime), zap.Any("workload-end-time", d.workloadEndTime))

				return
			}
		}
	}
}

func (d *WorkloadDriver) handleEvent(evt domain.Event) {
	sessionId := evt.Data().(*generator.Session).Pod
	switch evt.Name() {
	case generator.EventSessionStarted:
		d.logger.Debug("Received SessionStarted event.", zap.String("session-id", sessionId))

		// TODO: Start session.
		d.createKernel(sessionId)
		d.kernels[sessionId] = struct{}{}

		d.mu.Lock()
		defer d.mu.Unlock()
		d.workload.NumActiveSessions += 1
		d.workload.NumSessionsCreated += 1
	case generator.EventSessionTrainingStarted:
		d.logger.Debug("Received TrainingStarted event.", zap.String("session", sessionId))

		// TODO: Initiate training.
		if _, ok := d.kernels[sessionId]; !ok {
			d.logger.Error("Received 'training-started' event for unknown session.", zap.String("session-id", sessionId))
			d.errorChan <- ErrTrainingStartedUnknownSession
		}

		d.mu.Lock()
		defer d.mu.Unlock()
		d.workload.NumActiveTrainings += 1
	case generator.EventSessionUpdateGpuUtil:
		d.logger.Debug("Received UpdateGpuUtil event.", zap.String("session", sessionId))
		// TODO: Update GPU utilization.
	case generator.EventSessionTrainingEnded:
		d.logger.Debug("Received TrainingEnded event.", zap.String("session", sessionId))

		// TODO: Stop training.
		if _, ok := d.kernels[sessionId]; !ok {
			d.logger.Error("Received 'training-stopped' event for unknown session.", zap.String("session-id", sessionId))
			d.errorChan <- ErrTrainingStoppedUnknownSession
		}

		d.mu.Lock()
		defer d.mu.Unlock()
		d.workload.NumTasksExecuted += 1
		d.workload.NumActiveTrainings -= 1
	case generator.EventSessionStopped:
		d.logger.Debug("Received SessionStopped event.", zap.String("session", sessionId))

		// TODO: Stop session.
		d.stopKernel(sessionId)
		delete(d.kernels, sessionId)

		d.mu.Lock()
		defer d.mu.Unlock()
		d.workload.NumActiveSessions -= 1
	}
}

func (d *WorkloadDriver) createKernel(id string) {
	d.logger.Debug("Creating new kernel.", zap.String("kernel-id", id))
	d.kernelManager.CreateSession(id, id, fmt.Sprintf("%s.ipynb", id), "notebook")
}

func (d *WorkloadDriver) stopKernel(id string) {
	d.logger.Debug("Stopping kernel.", zap.String("kernel-id", id))
	d.kernelManager.StopKernel(id)
}

// Return the Workload Driver's "done" channel, which is used to signal that the simulation is complete.
func (d *WorkloadDriver) DoneChan() chan struct{} {
	return d.doneChan
}

// Submit an event to the Workload Driver for processing.
func (d *WorkloadDriver) SubmitEvent(evt domain.Event) {
	d.eventChan <- evt
}
