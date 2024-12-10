package workload

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/mattn/go-colorable"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	"github.com/scusemua/workload-driver-react/m/v2/internal/server/api/proto"
	"github.com/scusemua/workload-driver-react/m/v2/internal/server/clock"
	"github.com/scusemua/workload-driver-react/m/v2/internal/server/event_queue"
	"github.com/scusemua/workload-driver-react/m/v2/internal/server/metrics"
	"github.com/scusemua/workload-driver-react/m/v2/pkg/jupyter"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"k8s.io/apimachinery/pkg/util/wait"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	// ErrClientNotRunning  = errors.New("cannot stop client as it is not running")

	ErrInvalidFirstEvent = errors.New("client received invalid first event")
)

// ClientBuilder constructs a Client instance step-by-step.
type ClientBuilder struct {
	sessionId                 string
	workloadId                string
	sessionReadyEvent         *domain.Event
	startingTick              time.Time
	atom                      *zap.AtomicLevel
	targetTickDurationSeconds int64
	errorChan                 chan<- error
	session                   *domain.WorkloadTemplateSession
	workload                  internalWorkload
	kernelSessionManager      jupyter.KernelSessionManager
	notifyCallback            func(notification *proto.Notification)
	schedulingPolicy          string
	waitGroup                 *sync.WaitGroup
}

// NewClientBuilder initializes a new ClientBuilder.
func NewClientBuilder() *ClientBuilder {
	return &ClientBuilder{}
}

func (b *ClientBuilder) WithSessionId(sessionId string) *ClientBuilder {
	b.sessionId = sessionId
	return b
}

func (b *ClientBuilder) WithWorkloadId(workloadId string) *ClientBuilder {
	b.workloadId = workloadId
	return b
}

func (b *ClientBuilder) WithSessionReadyEvent(event *domain.Event) *ClientBuilder {
	b.sessionReadyEvent = event
	return b
}

func (b *ClientBuilder) WithStartingTick(startingTick time.Time) *ClientBuilder {
	b.startingTick = startingTick
	return b
}

func (b *ClientBuilder) WithAtom(atom *zap.AtomicLevel) *ClientBuilder {
	b.atom = atom
	return b
}

func (b *ClientBuilder) WithSchedulingPolicy(schedulingPolicy string) *ClientBuilder {
	if schedulingPolicy == "" {
		panic("Cannot use the empty string as a scheduling policy when creating a Client")
	}

	b.schedulingPolicy = schedulingPolicy
	return b
}

func (b *ClientBuilder) WithKernelManager(kernelSessionManager jupyter.KernelSessionManager) *ClientBuilder {
	b.kernelSessionManager = kernelSessionManager
	return b
}

func (b *ClientBuilder) WithTargetTickDurationSeconds(seconds int64) *ClientBuilder {
	b.targetTickDurationSeconds = seconds
	return b
}

func (b *ClientBuilder) WithErrorChan(errorChan chan<- error) *ClientBuilder {
	b.errorChan = errorChan
	return b
}

func (b *ClientBuilder) WithWorkload(workload internalWorkload) *ClientBuilder {
	b.workload = workload
	return b
}

func (b *ClientBuilder) WithSession(session *domain.WorkloadTemplateSession) *ClientBuilder {
	b.session = session
	return b
}

func (b *ClientBuilder) WithNotifyCallback(notifyCallback func(notification *proto.Notification)) *ClientBuilder {
	b.notifyCallback = notifyCallback
	return b
}

func (b *ClientBuilder) WithWaitGroup(waitGroup *sync.WaitGroup) *ClientBuilder {
	b.waitGroup = waitGroup
	return b
}

// Build constructs the Client instance.
func (b *ClientBuilder) Build() *Client {
	sessionMeta := b.sessionReadyEvent.Data.(domain.SessionMetadata)

	client := &Client{
		SessionId:  b.sessionId,
		EventQueue: event_queue.NewSessionEventQueue(b.sessionId),
		WorkloadId: b.workloadId,
		Workload:   b.workload,
		maximumResourceSpec: &domain.ResourceRequest{
			Cpus:     sessionMeta.GetMaxSessionCPUs(),
			MemoryMB: sessionMeta.GetMaxSessionMemory(),
			Gpus:     sessionMeta.GetMaxSessionGPUs(),
			VRAM:     sessionMeta.GetMaxSessionVRAM(),
		},
		currentTick:               clock.NewSimulationClockFromTime(b.startingTick),
		currentTime:               clock.NewSimulationClockFromTime(b.startingTick),
		targetTickDurationSeconds: b.targetTickDurationSeconds,
		targetTickDuration:        time.Second * time.Duration(b.targetTickDurationSeconds),
		clockTrigger:              clock.NewTrigger(),
		errorChan:                 b.errorChan,
		trainingStartedChannel:    make(chan interface{}, 1),
		trainingStoppedChannel:    make(chan interface{}, 1),
		Session:                   b.session,
		kernelSessionManager:      b.kernelSessionManager,
		notifyCallback:            b.notifyCallback,
		waitGroup:                 b.waitGroup,
		schedulingPolicy:          b.schedulingPolicy,
	}

	zapConfig := zap.NewDevelopmentEncoderConfig()
	zapConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	core := zapcore.NewCore(zapcore.NewConsoleEncoder(zapConfig), zapcore.AddSync(colorable.NewColorableStdout()), b.atom)
	logger := zap.New(core, zap.Development())
	if logger == nil {
		panic("failed to create logger for workload driver")
	}
	client.logger = logger

	client.ticker = client.clockTrigger.NewSyncTicker(time.Second*time.Duration(client.targetTickDurationSeconds), fmt.Sprintf("Client-%s", client.SessionId), client.currentTick)

	client.EventQueue.Push(b.sessionReadyEvent)

	return client
}

// Client encapsulates a Session and runs as a dedicated goroutine, processing events for that Session.
type Client struct {
	Workload internalWorkload
	Session  *domain.WorkloadTemplateSession

	SessionId                 string                                 // SessionId is the Jupyter kernel/session ID of this Client
	WorkloadId                string                                 // WorkloadId is the ID of the workload that the Client is a part of.
	errorChan                 chan<- error                           // errorChan is used to notify the WorkloadDriver that an error has occurred.
	kernelConnection          jupyter.KernelConnection               // kernelConnection is the Client's Jupyter kernel connection. The Client uses this to send messages to its kernel.
	sessionConnection         *jupyter.SessionConnection             // sessionConnection is the Client's Jupyter session connection.
	kernelSessionManager      jupyter.KernelSessionManager           // kernelSessionManager is used by the Client to create its sessionConnection and subsequently its kernelConnection.
	schedulingPolicy          string                                 // schedulingPolicy is the name of the scheduling policy that the cluster is configured to use.
	EventQueue                *event_queue.SessionEventQueue         // EventQueue contains events to be processed by this Client.
	maximumResourceSpec       *domain.ResourceRequest                // maximumResourceSpec is the maximum amount of resources this Client may use at any point in its lifetime.
	targetTickDurationSeconds int64                                  // targetTickDurationSeconds is how long each tick was in the trace data used to generate this workload
	targetTickDuration        time.Duration                          // targetTickDuration is how long each tick is supposed to last. This is the tick interval/step rate of the simulation.
	currentTick               domain.SimulationClock                 // currentTick maintains the time for this Client.
	currentTime               domain.SimulationClock                 // currentTime contains the current clock time of the workload, which will be sometime between currentTick and currentTick + TickDuration.
	ticker                    *clock.Ticker                          // ticker delivers ticks, which drives this Client's workload. Each time a tick is received, the Client will process events for that tick.
	clockTrigger              *clock.Trigger                         // clockTrigger is a trigger for the clock ticks
	logger                    *zap.Logger                            // logger is how the Client prints log messages.
	running                   atomic.Int32                           // running indicates whether this Client is actively processing events.
	ticksHandled              atomic.Int64                           // ticksHandled is the number of ticks handled by this Client.
	lastTrainingSubmittedAt   time.Time                              // lastTrainingSubmittedAt is the real-world clock time at which the last training was submitted to the kernel.
	trainingStartedChannel    chan interface{}                       // trainingStartedChannel is used to notify that the last/current training has started.
	trainingStoppedChannel    chan interface{}                       // trainingStoppedChannel is used to notify that the last/current training has ended.
	notifyCallback            func(notification *proto.Notification) // notifyCallback is used to send notifications directly to the frontend.
	waitGroup                 *sync.WaitGroup                        // waitGroup is used to alert the WorkloadDriver that the Client has finished.
}

// Run starts the Client and instructs the Client to begin processing its events in a loop.
func (c *Client) Run() {
	if !c.running.CompareAndSwap(0, 1) {
		c.logger.Warn("Client is already running.",
			zap.String("session_id", c.SessionId),
			zap.String("workload_id", c.WorkloadId))
		return
	}

	err := c.initialize()
	if err != nil {
		c.logger.Error("Failed to initialize client.",
			zap.String("session_id", c.SessionId),
			zap.String("workload_id", c.WorkloadId),
			zap.Error(err))

		c.running.Store(0)
		return
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go c.issueClockTicks(&wg)
	c.run(&wg)

	wg.Wait()
}

// createKernel attempts to create the kernel for the Client, possibly handling any errors that are encountered
// if the errors are something we can deal with. If not, they're returned, and the workload explodes.
func (c *Client) createKernel(evt *domain.Event) (*jupyter.SessionConnection, error) {
	var initialResourceRequest *jupyter.ResourceSpec
	if c.schedulingPolicy == "static" || c.schedulingPolicy == "dynamic-v3" || c.schedulingPolicy == "dynamic-v4" {
		// Try to get the first training event of the session, and just reserve those resources.
		firstTrainingEvent := c.Session.Trainings[0]

		if firstTrainingEvent != nil {
			initialResourceRequest = &jupyter.ResourceSpec{
				Cpu:  int(firstTrainingEvent.Millicpus),
				Mem:  firstTrainingEvent.MemUsageMB,
				Gpu:  firstTrainingEvent.NumGPUs(),
				Vram: firstTrainingEvent.VRamUsageGB,
			}
		} else {
			c.logger.Warn("Could not find first training event of session.",
				zap.String("workload_id", c.WorkloadId),
				zap.String("session_id", c.SessionId))
		}
	}

	if initialResourceRequest == nil {
		initialResourceRequest = &jupyter.ResourceSpec{
			Cpu:  int(c.maximumResourceSpec.Cpus),
			Mem:  c.maximumResourceSpec.MemoryMB,
			Gpu:  c.maximumResourceSpec.Gpus,
			Vram: c.maximumResourceSpec.VRAM,
		}
	}

	backoff := wait.Backoff{
		Duration: time.Second * 5,
		Factor:   2,
		Jitter:   float64(time.Millisecond * 250),
		Steps:    10,
		Cap:      time.Second * 120,
	}

	var (
		sessionConnection *jupyter.SessionConnection
		err               error
	)

	for sessionConnection == nil && backoff.Steps > 0 {
		c.logger.Debug("Issuing create-session request now.",
			zap.String("session_id", c.SessionId),
			zap.String("workload_id", c.WorkloadId),
			zap.String("resource_request", initialResourceRequest.String()))
		sessionConnection, err = c.kernelSessionManager.CreateSession(
			c.SessionId, fmt.Sprintf("%s.ipynb", c.SessionId),
			"notebook", "distributed", initialResourceRequest)
		if err != nil {
			c.logger.Warn("Failed to create session.",
				zap.String("workload_id", c.WorkloadId),
				zap.String("session_id", c.SessionId),
				zap.Error(err))

			if strings.Contains(err.Error(), "insufficient hosts available") {
				sleepInterval := backoff.Step()

				c.logger.Warn("Failed to create session due to insufficient hosts available. Will requeue event and try again later.",
					zap.String("workload_id", c.Workload.GetId()),
					zap.String("workload_name", c.Workload.WorkloadName()),
					zap.String("session_id", c.SessionId),
					zap.Time("original_timestamp", evt.OriginalTimestamp),
					zap.Time("current_timestamp", evt.Timestamp),
					zap.Time("current_tick", c.currentTick.GetClockTime()),
					zap.Int32("num_times_enqueued", evt.GetNumTimesEnqueued()),
					zap.Duration("total_delay", evt.TotalDelay()),
					zap.Int("attempt_number", backoff.Steps-10+1),
					zap.Duration("sleep_interval", sleepInterval))

				// TODO: How to accurately compute the delay here? Since we're using ticks, so one minute is the
				// 		 minimum meaningful delay, really, but we're also using big time compression factors?
				c.incurDelay(sleepInterval)

				time.Sleep(sleepInterval)
				continue
			}

			// Will return nil and a non-nil error.
			return nil, err
		}
	}

	if sessionConnection != nil {
		c.logger.Debug("Successfully created kernel.",
			zap.String("session_id", c.SessionId),
			zap.String("workload_id", c.WorkloadId))
	}

	return sessionConnection, err
}

type parsedIoPubMessage struct {
	Stream string
	Text   string
}

// initialize creates the associated kernel and connects to it.
func (c *Client) initialize() error {
	evt := c.EventQueue.Pop()

	if evt.Name != domain.EventSessionReady {
		c.logger.Error("Received unexpected event for first event.",
			zap.String("session_id", c.SessionId),
			zap.String("event_name", evt.Name.String()),
			zap.String("event", evt.String()),
			zap.String("workload_id", c.WorkloadId))

		return fmt.Errorf("%w: \"%s\"", ErrInvalidFirstEvent, evt.Name.String())
	}

	c.logger.Debug("Initializing client.",
		zap.String("session_id", c.SessionId),
		zap.String("workload_id", c.WorkloadId))

	sessionConnection, err := c.createKernel(evt)
	if err != nil {
		c.logger.Error("Completely failed to create kernel.",
			zap.String("session_id", c.SessionId),
			zap.String("workload_id", c.WorkloadId),
			zap.Error(err))
		return err
	}

	c.sessionConnection = sessionConnection

	// ioPubHandler is a session-specific wrapper around the standard BasicWorkloadDriver::handleIOPubMessage method.
	// This returns true if the received IOPub message is a "stream" message and is parsed successfully.
	// Otherwise, this returns false.
	//
	// The return value is not really used.
	ioPubHandler := func(conn jupyter.KernelConnection, kernelMessage jupyter.KernelMessage) interface{} {
		// Parse the IOPub message.
		// If it is a stream message, this will return a *parsedIoPubMessage variable.
		parsedIoPubMsgVal := c.handleIOPubMessage(conn, kernelMessage)

		if parsedIoPubMsg, ok := parsedIoPubMsgVal.(*parsedIoPubMessage); ok {
			switch parsedIoPubMsg.Stream {
			case "stdout":
				{
					c.Session.AddStdoutIoPubMessage(parsedIoPubMsg.Text)
				}
			case "stderr":
				{
					c.Session.AddStderrIoPubMessage(parsedIoPubMsg.Text)
				}
			default:
				c.logger.Warn("Unexpected stream specified by IOPub message.",
					zap.String("workload_id", c.Workload.GetId()),
					zap.String("workload_name", c.Workload.WorkloadName()),
					zap.String("session_id", c.SessionId),
					zap.String("stream", parsedIoPubMsg.Stream))
				return false
			}
			return true
		}

		return false
	}

	if err = sessionConnection.RegisterIoPubHandler(c.SessionId, ioPubHandler); err != nil {
		c.logger.Error("Failed to register IOPub message handler.",
			zap.String("workload_id", c.Workload.GetId()),
			zap.String("workload_name", c.Workload.WorkloadName()),
			zap.String("session_id", c.SessionId), zap.Error(err))
		return err
	}

	c.sessionConnection = sessionConnection

	return nil
}

// issueClockTicks issues clock ticks for this Client, driving this Client's execution.
//
// issueClockTicks should be executed in its own goroutine.
func (c *Client) issueClockTicks(wg *sync.WaitGroup) {
	defer wg.Done()

	c.logger.Debug("Client is preparing to begin incrementing client-level ticker.",
		zap.String("session_id", c.SessionId),
		zap.String("workload_id", c.WorkloadId))

	for c.Workload.IsInProgress() {
		// Increment the clock.
		tick, err := c.currentTick.IncrementClockBy(c.targetTickDuration)
		if err != nil {
			c.logger.Error("Error while incrementing clock time.",
				zap.Duration("tick-duration", c.targetTickDuration),
				zap.String("workload_id", c.WorkloadId),
				zap.String("workload_name", c.Workload.WorkloadName()),
				zap.Error(err))
			c.errorChan <- err
			break
		}

		c.logger.Debug("Client incremented client-level ticker. Triggering events now.",
			zap.String("session_id", c.SessionId),
			zap.String("workload_id", c.WorkloadId),
			zap.Time("tick", tick))
		c.clockTrigger.Trigger(tick)
	}

	c.logger.Debug("Client has finished issuing clock ticks.",
		zap.String("session_id", c.SessionId),
		zap.String("workload_id", c.WorkloadId),
		zap.Time("final_tick", c.currentTick.GetClockTime()))
}

// run is the private, core implementation of Run.
func (c *Client) run(wg *sync.WaitGroup) {
	defer wg.Done()
	defer func() {
		c.logger.Debug("Client is done running.",
			zap.String("session_id", c.SessionId),
			zap.String("workload_id", c.WorkloadId))

		if swapped := c.running.CompareAndSwap(1, 0); !swapped {
			c.logger.Error("Running was not set to 1.",
				zap.String("session_id", c.SessionId),
				zap.String("workload_id", c.WorkloadId))
		}
	}()

	for c.Workload.IsInProgress() {
		select {
		case tick := <-c.ticker.TickDelivery:
			c.logger.Debug("Client received tick.",
				zap.String("session_id", c.SessionId),
				zap.String("workload_id", c.WorkloadId),
				zap.String("workload_name", c.Workload.WorkloadName()),
				zap.Time("tick", tick))

			err := c.handleTick(tick)

			if err != nil {
				c.errorChan <- err
				return
			}
		}
	}
}

func (c *Client) handleTick(tick time.Time) error {
	c.logger.Debug("Serving tick.",
		zap.String("session_id", c.SessionId),
		zap.String("workload_id", c.WorkloadId),
		zap.Time("tick", tick))

	_, _, err := c.currentTick.IncreaseClockTimeTo(tick)
	if err != nil {
		c.logger.Error("Failed to increase CurrentTick.",
			zap.String("session_id", c.SessionId),
			zap.String("workload_id", c.WorkloadId),
			zap.Time("target_time", tick),
			zap.Error(err))
		return err
	}

	// If there are no events processed this tick, then we still need to increment the clock time so we're in-line with the simulation.
	// Check if the current clock time is earlier than the start of the previous tick. If so, increment the clock time to the beginning of the tick.
	prevTickStart := tick.Add(-c.targetTickDuration)
	if c.currentTime.GetClockTime().Before(prevTickStart) {
		if _, _, err = c.incrementClockTime(prevTickStart); err != nil {
			c.logger.Error("Failed to increase CurrentTime.",
				zap.String("session_id", c.SessionId),
				zap.String("workload_id", c.WorkloadId),
				zap.Time("target_time", tick),
				zap.Error(err))
			return err
		}
	}

	err = c.processEventsForTick(tick)
	if err != nil {
		return err
	}

	c.logger.Debug("Finished serving tick.",
		zap.String("session_id", c.SessionId),
		zap.String("workload_id", c.WorkloadId),
		zap.Time("tick", tick))

	c.ticksHandled.Add(1)
	c.ticker.Done()
	return nil
}

// incrementClockTime sets the c.clockTime clock to the given timestamp, verifying that the new timestamp is either
// equal to or occurs after the old one.
//
// incrementClockTime returns a tuple where the first element is the new time, and the second element is the difference
// between the new time and the old time.
func (c *Client) incrementClockTime(time time.Time) (time.Time, time.Duration, error) {
	newTime, timeDifference, err := c.currentTime.IncreaseClockTimeTo(time)

	if err != nil {
		c.logger.Error("Critical error occurred when attempting to increase clock time.", zap.Error(err))
	}

	return newTime, timeDifference, err // err will be nil if everything was OK.
}

// processEventsForTick processes events in chronological/simulation order.
// This accepts the "current tick" as an argument. The current tick is basically an upper-bound on the times for
// which we'll process an event. For example, if `tick` is 19:05:00, then we will process all cluster and session
// events with timestamps that are (a) equal to 19:05:00 or (b) come before 19:05:00. Any events with timestamps
// that come after 19:05:00 will not be processed until the next tick.
func (c *Client) processEventsForTick(tick time.Time) error {
	numEventsProcessed := 0

	for c.EventQueue.HasEventsForTick(tick) {
		event := c.EventQueue.Pop()

		c.logger.Debug("Handling workload event.",
			zap.String("event_name", event.Name.String()),
			zap.String("session", c.SessionId),
			zap.String("event_name", event.Name.String()),
			zap.String("workload_name", c.Workload.WorkloadName()),
			zap.String("workload_id", c.Workload.GetId()),
			zap.Time("tick", tick))

		err := c.handleEvent(event, tick)

		if err != nil {
			workloadWillAbort := c.handleError(event, err, tick)
			if workloadWillAbort {
				return err
			}
		}

		numEventsProcessed += 1
	}

	c.logger.Debug("Client finished processing events for tick.",
		zap.String("session_id", c.SessionId),
		zap.String("workload_id", c.Workload.GetId()),
		zap.Time("tick", tick),
		zap.Int("num_events_processed", numEventsProcessed))

	return nil
}

// handleError is called by processEventsForTick when handleEvent returns an error.
//
// handleError returns true if the workload is going to be aborted (due to the error being irrecoverable) and
// false if the workload will continue.
func (c *Client) handleError(event *domain.Event, err error, tick time.Time) bool {
	c.logger.Error("Failed to handle event workload event.",
		zap.String("event_name", event.Name.String()),
		zap.String("session", c.SessionId),
		zap.String("event_name", event.Name.String()),
		zap.String("workload_name", c.Workload.WorkloadName()),
		zap.String("workload_id", c.Workload.GetId()),
		zap.Time("tick", tick),
		zap.Error(err))

	// In these cases, we'll just discard the events and continue.
	if errors.Is(err, domain.ErrUnknownSession) || errors.Is(err, ErrUnknownEventType) {
		return false
	}

	return true
}

// handleEvent handles a single *domain.Event.
func (c *Client) handleEvent(event *domain.Event, tick time.Time) error {
	var err error
	switch event.Name {
	case domain.EventSessionTraining:
		err = c.handleTrainingEvent(event, tick)
	case domain.EventSessionStopped:
		err = c.handleSessionStoppedEvent(event)

		// Record it as processed even if there was an error when processing the event.
		c.Workload.ProcessedEvent(domain.NewEmptyWorkloadEvent().
			WithEventId(event.Id()).
			WithSessionId(event.SessionID()).
			WithEventName(event.Name).
			WithEventTimestamp(event.Timestamp).
			WithProcessedAtTime(time.Now()).
			WithError(err))
	default:
		c.logger.Error("Received event of unknown or unexpected type.",
			zap.String("session_id", c.SessionId),
			zap.String("event_name", event.Name.String()),
			zap.String("workload_id", c.Workload.GetId()),
			zap.String("workload_name", c.Workload.WorkloadName()),
			zap.String("event", event.String()))

		err = fmt.Errorf("%w: \"%s\"", ErrUnknownEventType, event.Name.String())
	}

	return err // Will be nil on success
}

// handleTrainingEvent handles a domain.EventSessionTraining *domain.Event.
func (c *Client) handleTrainingEvent(event *domain.Event, tick time.Time) error {
	startedHandlingAt := time.Now()

	timeoutInterval := c.getTimeoutInterval(event)
	ctx, cancel := context.WithTimeout(context.Background(), timeoutInterval)
	defer cancel()

	sentRequestAt, err := c.submitTrainingToKernel(event)
	// Record it as processed even if there was an error when processing the event.
	c.Workload.ProcessedEvent(domain.NewEmptyWorkloadEvent().
		WithEventId(event.Id()).
		WithSessionId(event.SessionID()).
		WithEventName(domain.EventSessionTrainingStarted).
		WithEventTimestamp(event.Timestamp).
		WithProcessedAtTime(time.Now()).
		WithError(err)) // Will be nil on success

	if err != nil {
		c.logger.Error("Failed to submit training to kernel.",
			zap.String("workload_id", c.Workload.GetId()),
			zap.String("workload_name", c.Workload.WorkloadName()),
			zap.String("session_id", c.SessionId),
			zap.String("event", event.StringJson()),
			zap.Error(err))
		return err
	}

	c.logger.Debug("Handled \"training-started\" event.",
		zap.String("workload_id", c.Workload.GetId()),
		zap.String("workload_name", c.Workload.WorkloadName()),
		zap.String("session_id", c.SessionId),
		zap.Time("tick", tick))

	err = c.waitForTrainingToStart(ctx, event, startedHandlingAt, sentRequestAt)
	if err != nil {
		c.logger.Error("Failed to wait for training to start.",
			zap.String("workload_id", c.Workload.GetId()),
			zap.String("workload_name", c.Workload.WorkloadName()),
			zap.String("session_id", c.SessionId),
			zap.String("event", event.StringJson()),
			zap.Error(err))
		return err
	}

	err = c.waitForTrainingToEnd(ctx)
	c.Workload.ProcessedEvent(domain.NewEmptyWorkloadEvent().
		WithEventId(event.Id()).
		WithSessionId(event.SessionID()).
		WithEventName(domain.EventSessionTrainingEnded).
		WithEventTimestamp(event.Timestamp).
		WithProcessedAtTime(time.Now()).
		WithError(err)) // Will be nil on success

	return nil // Will be nil on success
}

// submitTrainingToKernel submits a training event to be processed/executed by the kernel.
func (c *Client) submitTrainingToKernel(evt *domain.Event) (sentRequestAt time.Time, err error) {
	c.logger.Debug("Client received training event.",
		zap.String("session_id", c.SessionId),
		zap.String("workload_id", c.Workload.GetId()),
		zap.String("workload_name", c.Workload.WorkloadName()),
		zap.Duration("training_duration", evt.Duration),
		zap.String("event_id", evt.ID),
		zap.Float64("training_duration_sec", evt.Duration.Seconds()))

	kernelConnection := c.sessionConnection.Kernel()
	if kernelConnection == nil {
		c.logger.Error("No kernel connection found for session connection.",
			zap.String("workload_id", c.Workload.GetId()),
			zap.String("workload_name", c.Workload.WorkloadName()),
			zap.String("session_id", c.SessionId))
		err = ErrNoKernelConnection
		return
	}

	var executeRequestArgs *jupyter.RequestExecuteArgs
	executeRequestArgs, err = c.createExecuteRequestArguments(evt)
	if executeRequestArgs == nil || err != nil {
		c.logger.Error("Failed to create 'execute_request' arguments.",
			zap.String("workload_id", c.Workload.GetId()),
			zap.String("workload_name", c.Workload.WorkloadName()),
			zap.String("session_id", c.SessionId),
			zap.Error(err))
		return time.Time{}, err
	}

	c.logger.Debug("Submitting \"execute_request\" now.",
		zap.String("session_id", c.SessionId),
		zap.String("workload_id", c.Workload.GetId()),
		zap.String("workload_name", c.Workload.WorkloadName()),
		zap.Duration("original_training_duration", evt.Duration),
		zap.String("event_id", evt.ID),
		zap.Float64("training_duration_sec", evt.Duration.Seconds()),
		zap.String("execute_request_args", executeRequestArgs.String()))

	sentRequestAt = time.Now()
	_, err = kernelConnection.RequestExecute(executeRequestArgs)
	if err != nil {
		c.logger.Error("Error while submitting training event to kernel.",
			zap.String("workload_id", c.Workload.GetId()),
			zap.String("workload_name", c.Workload.WorkloadName()),
			zap.String("session_id", c.SessionId))
		return time.Time{}, err
	}

	c.lastTrainingSubmittedAt = time.Now()

	c.Workload.TrainingSubmitted(c.SessionId, evt)

	return sentRequestAt, nil
}

func (c *Client) onReceiveExecuteReply(response jupyter.KernelMessage) {
	responseContent := response.GetContent().(map[string]interface{})
	if responseContent == nil {
		c.logger.Error("\"execute_reply\" message does not have any content...",
			zap.String("workload_id", c.Workload.GetId()),
			zap.String("workload_name", c.Workload.WorkloadName()),
			zap.String("session_id", c.SessionId),
			zap.String("response", response.String()))
		return
	}

	val, ok := responseContent["status"]
	if !ok {
		c.logger.Error("\"execute_reply\" message does not contain a \"status\" field in its content.",
			zap.String("workload_id", c.Workload.GetId()),
			zap.String("workload_name", c.Workload.WorkloadName()),
			zap.String("session_id", c.SessionId),
			zap.String("response", response.String()))
		return
	}

	status := val.(string)

	if status == "error" {
		errorName := responseContent["ename"].(string)
		errorValue := responseContent["evalue"].(string)

		c.logger.Warn("Received \"execute_reply\" message with error status.",
			zap.String("workload_id", c.Workload.GetId()),
			zap.String("workload_name", c.Workload.WorkloadName()),
			zap.String("session_id", c.SessionId),
			zap.String("ename", errorName),
			zap.String("evalue", errorValue),
			zap.String("response", response.String()))

		// Notify the training started channel. There will not be a smr_lead_task sent at this point, since
		// there was an error, so we'll send the notification to the training_started channel.
		c.trainingStartedChannel <- fmt.Errorf("%s: %s", errorName, errorValue)
		return
	}

	c.logger.Debug("Received \"execute_reply\" message with non-error status.",
		zap.String("workload_id", c.Workload.GetId()),
		zap.String("workload_name", c.Workload.WorkloadName()),
		zap.String("session_id", c.SessionId),
		zap.String("response", response.String()))
	c.trainingStoppedChannel <- response
}

// incurDelay is called when we experience some sort of delay and need to delay our future events accordingly.
func (c *Client) incurDelay(delayAmount time.Duration) {
	c.EventQueue.IncurDelay(delayAmount)

	c.Workload.SessionDelayed(c.SessionId, delayAmount)

	if metrics.PrometheusMetricsWrapperInstance != nil {
		metrics.PrometheusMetricsWrapperInstance.SessionDelayedDueToResourceContention.
			With(prometheus.Labels{
				"workload_id": c.Workload.GetId(),
				"session_id":  c.SessionId,
			}).Add(1)
	}
}

// waitForTrainingToStart waits for a training to begin being processed by a kernel replica.
//
// waitForTrainingToStart is called by handleTrainingEvent after submitTrainingToKernel is called.
func (c *Client) waitForTrainingToStart(ctx context.Context, evt *domain.Event, startedHandlingAt time.Time, sentRequestAt time.Time) error {

	c.logger.Debug("Waiting for session to start training before continuing...",
		zap.String("workload_id", c.Workload.GetId()),
		zap.String("workload_name", c.Workload.WorkloadName()),
		zap.String("session_id", c.SessionId))

	select {
	case v := <-c.trainingStartedChannel:
		{
			switch v.(type) {
			case error:
				{
					err := v.(error)
					c.logger.Warn("Session failed to start training",
						zap.String("workload_id", c.Workload.GetId()),
						zap.String("workload_name", c.Workload.WorkloadName()),
						zap.String("session_id", c.SessionId),
						zap.Duration("time_elapsed", time.Since(sentRequestAt)),
						zap.Error(err))

					// If we fail to start training for some reason, then we'll just try again later.
					c.incurDelay(time.Since(startedHandlingAt) + c.targetTickDuration*2)

					// Put the event back in the queue.
					c.EventQueue.Push(evt)
				}
			default:
				{
					startLatency := time.Since(sentRequestAt)
					c.logger.Debug("Session started training",
						zap.String("workload_id", c.Workload.GetId()),
						zap.String("workload_name", c.Workload.WorkloadName()),
						zap.String("session_id", c.SessionId),
						zap.Duration("start_latency", startLatency))
				}
			}
		}
	case <-ctx.Done():
		{
			c.trainingStartTimedOut(sentRequestAt)
		}
	}

	return nil
}

// trainingStartTimedOut is called by waitForTrainingToStart when we don't receive a notification that the submitted
// training event started being processed after the timeout interval elapses.
func (c *Client) trainingStartTimedOut(sentRequestAt time.Time) {
	c.logger.Warn("Have not received 'training started' notification for over 1 minute. Assuming message was lost.",
		zap.String("workload_id", c.Workload.GetId()),
		zap.String("workload_name", c.Workload.WorkloadName()),
		zap.String("session_id", c.SessionId),
		zap.Duration("time_elapsed", time.Since(sentRequestAt)))

	c.notifyCallback(&proto.Notification{
		Id:    uuid.NewString(),
		Title: "Have Spent 1+ Minute(s) Waiting for 'Training Started' Notification",
		Message: fmt.Sprintf("Submitted \"execute_request\" to kernel \"%s\" during workload \"%s\" (ID=\"%s\") "+
			"over 1 minute ago and have not yet received 'smr_lead_task' IOPub message. Time elapsed: %v.",
			c.SessionId, c.Workload.WorkloadName(), c.Workload.GetId(), time.Since(sentRequestAt)),
		Panicked:         false,
		NotificationType: domain.WarningNotification.Int32(),
	})

	// TODO: Resubmit the event?
}

// convertTimestampToTickNumber converts the given tick, which is specified in the form of a time.Time,
// and returns what "tick number" that tick is.
//
// Basically, you just convert the timestamp to its unix epoch timestamp (in seconds), and divide by the
// trace step value (also in seconds).
func (c *Client) convertTimestampToTickNumber(tick time.Time) int64 {
	return tick.Unix() / c.targetTickDurationSeconds
}

// handleIOPubMessage returns the extracted text.
// This is expected to be called within a session-specific wrapper.
//
// If the IOPub message is a "stream" message, then this returns a *parsedIoPubMessage
// wrapping the name of the stream and the message text.
func (c *Client) handleIOPubMessage(conn jupyter.KernelConnection, kernelMessage jupyter.KernelMessage) interface{} {
	// We just want to extract the output from 'stream' IOPub messages.
	// We don't care about non-stream-type IOPub messages here, so we'll just return.
	messageType := kernelMessage.GetHeader().MessageType
	if messageType != "stream" && messageType != "smr_lead_task" {
		return nil
	}

	if messageType == "stream" {
		return c.handleIOPubStreamMessage(conn, kernelMessage)
	}

	return c.handleIOPubSmrLeadTaskMessage(conn, kernelMessage)
}

func (c *Client) handleIOPubSmrLeadTaskMessage(conn jupyter.KernelConnection, kernelMessage jupyter.KernelMessage) string {
	c.logger.Debug("Received 'smr_lead_task' message from kernel.",
		zap.String("workload_id", c.Workload.GetId()),
		zap.String("workload_name", c.Workload.WorkloadName()),
		zap.String("session_id", conn.KernelId()))

	c.Workload.TrainingStarted(c.SessionId, c.convertTimestampToTickNumber(c.currentTick.GetClockTime()))

	// Use the timestamp encoded in the IOPub message to determine when the training actually began,
	// and then delay the session by how long it took for training to begin.
	content := kernelMessage.GetContent().(map[string]interface{})

	var trainingStartedAt int64
	val, ok := content["msg_created_at_unix_milliseconds"]
	if !ok {
		c.logger.Error("Could not recover unix millisecond timestamp from \"smr_lead_task\" IOPub message.",
			zap.String("workload_id", c.Workload.GetId()),
			zap.String("workload_name", c.Workload.WorkloadName()),
			zap.String("session_id", conn.KernelId()),
			zap.Any("message_content", content))

		panic("Could not recover unix millisecond timestamp from \"smr_lead_task\" IOPub message.")
	}

	trainingStartedAt = int64(val.(float64))

	delayMilliseconds := trainingStartedAt - c.lastTrainingSubmittedAt.UnixMilli()
	if delayMilliseconds < 0 {
		c.logger.Error("Computed invalid delay between training submission and training start...",
			zap.String("workload_id", c.Workload.GetId()),
			zap.String("workload_name", c.Workload.WorkloadName()),
			zap.String("session_id", conn.KernelId()),
			zap.Time("sent_execute_request_at", c.lastTrainingSubmittedAt),
			zap.Int64("training_started_at", trainingStartedAt),
			zap.Int64("computed_delay_millis", delayMilliseconds))

		delayMilliseconds = 0
	}

	c.Workload.UpdateStatistics(func(stats *Statistics) {
		stats.JupyterTrainingStartLatenciesDashboardMillis = append(
			stats.JupyterTrainingStartLatenciesDashboardMillis, float64(delayMilliseconds))

		stats.JupyterTrainingStartLatencyDashboardMillis += float64(delayMilliseconds)
	})

	c.logger.Debug("Computed training-started delay for session.",
		zap.String("workload_id", c.Workload.GetId()),
		zap.String("workload_name", c.Workload.WorkloadName()),
		zap.String("session_id", conn.KernelId()),
		zap.Time("sent_execute_request_at", c.lastTrainingSubmittedAt),
		zap.Int64("training_started_at", trainingStartedAt),
		zap.Int64("computed_delay", delayMilliseconds))

	c.incurDelay(time.Millisecond * time.Duration(delayMilliseconds))

	c.trainingStartedChannel <- struct{}{}

	return c.SessionId
}

func (c *Client) handleIOPubStreamMessage(conn jupyter.KernelConnection, kernelMessage jupyter.KernelMessage) interface{} {
	content := kernelMessage.GetContent().(map[string]interface{})

	var (
		stream string
		text   string
		ok     bool
	)

	stream, ok = content["name"].(string)
	if !ok {
		c.logger.Warn("Content of IOPub message did not contain an entry with key \"name\" and value of type string.",
			zap.String("workload_id", c.Workload.GetId()),
			zap.String("workload_name", c.Workload.WorkloadName()),
			zap.Any("content", content), zap.Any("message", kernelMessage),
			zap.String("session_id", conn.KernelId()))
		return nil
	}

	text, ok = content["text"].(string)
	if !ok {
		c.logger.Warn("Content of IOPub message did not contain an entry with key \"text\" and value of type string.", zap.String("workload_id", c.Workload.GetId()), zap.String("workload_name", c.Workload.WorkloadName()), zap.Any("content", content), zap.Any("message", kernelMessage), zap.String("session_id", conn.KernelId()))
		return nil
	}

	return &parsedIoPubMessage{
		Stream: stream,
		Text:   text,
	}
}

// createExecuteRequestArguments creates the arguments for an "execute_request" from the given event.
//
// The event must be of type "training-started", or this will return nil.
func (c *Client) createExecuteRequestArguments(evt *domain.Event) (*jupyter.RequestExecuteArgs, error) {
	if evt.Name != domain.EventSessionTraining {
		c.logger.Error("Attempted to create \"execute_request\" arguments for event of invalid type.",
			zap.String("event_type", evt.Name.String()),
			zap.String("event_id", evt.Id()),
			zap.String("session_id", evt.SessionID()),
			zap.String("workload_id", c.Workload.GetId()),
			zap.String("workload_name", c.Workload.WorkloadName()))

		return nil, fmt.Errorf("invalid event type: %s", evt.Name)
	}

	sessionMetadata := evt.Data.(domain.SessionMetadata)

	if sessionMetadata == nil {
		c.logger.Error("Event has nil data.",
			zap.String("event_type", evt.Name.String()),
			zap.String("event_id", evt.Id()),
			zap.String("session_id", evt.SessionID()),
			zap.String("workload_id", c.Workload.GetId()),
			zap.String("workload_name", c.Workload.WorkloadName()))
		return nil, fmt.Errorf("event has nil data")
	}

	gpus := sessionMetadata.GetCurrentTrainingMaxGPUs()
	if gpus == 0 && sessionMetadata.HasGpus() && sessionMetadata.GetGPUs() > 0 {
		gpus = sessionMetadata.GetGPUs()
	}

	resourceRequest := &domain.ResourceRequest{
		Cpus:     sessionMetadata.GetCurrentTrainingMaxCPUs(),
		MemoryMB: sessionMetadata.GetCurrentTrainingMaxMemory(),
		VRAM:     sessionMetadata.GetVRAM(),
		Gpus:     gpus,
	}

	milliseconds := float64(evt.Duration.Milliseconds())
	if c.Workload.ShouldTimeCompressTrainingDurations() {
		milliseconds = milliseconds * c.Workload.GetTimescaleAdjustmentFactor()
		c.logger.Debug("Applied time-compression to training duration.",
			zap.String("session_id", evt.SessionID()),
			zap.Duration("original_duration", evt.Duration),
			zap.Float64("updated_duration", milliseconds),
			zap.String("event_id", evt.Id()),
			zap.String("workload_id", c.Workload.GetId()),
			zap.String("workload_name", c.Workload.WorkloadName()))
	}

	argsBuilder := jupyter.NewRequestExecuteArgsBuilder().
		Code(TrainingCode).
		Silent(false).
		StoreHistory(true).
		UserExpressions(nil).
		AllowStdin(true).
		StopOnError(false).
		AwaitResponse(false).
		OnResponseCallback(c.onReceiveExecuteReply).
		AddMetadata("resource_request", resourceRequest).
		AddMetadata("training_duration_millis", milliseconds)

	return argsBuilder.Build(), nil
}

// getAdjustedDuration returns the duration of the *domain.Event adjusted based on the timescale adjustment factor
// of the Client's Workload.
func (c *Client) getAdjustedDuration(evt *domain.Event) time.Duration {
	timescaleAdjustmentFactor := c.Workload.GetTimescaleAdjustmentFactor()
	duration := evt.Duration

	if duration == 0 {
		return 0
	}

	return time.Duration(timescaleAdjustmentFactor * float64(evt.Duration))
}

// getTimeoutInterval computes a "meaningful" timeout interval based on the scheduling policy, taking into account
// approximately how long the network I/O before/after training is expected to take and whatnot.
func (c *Client) getTimeoutInterval(evt *domain.Event) time.Duration {
	// Load the scheduling policy.
	schedulingPolicy := c.schedulingPolicy
	if schedulingPolicy == "" {
		c.logger.Warn("Could not compute meaningful timeout interval because scheduling policy is invalid.",
			zap.String("workload_id", c.Workload.GetId()),
			zap.String("workload_name", c.Workload.WorkloadName()),
			zap.String("session_id", c.SessionId),
			zap.String("event", evt.Name.String()))
		return time.Minute + c.getAdjustedDuration(evt)
	}

	if schedulingPolicy == "static" || schedulingPolicy == "dynamic-v3" || schedulingPolicy == "dynamic-v4" {
		// There's no network I/O on the critical path, so stopping the training should be quick.
		return (time.Second * 30) + c.getAdjustedDuration(evt)
	}

	// Get the remote storage definition of the workload.
	remoteStorageDefinition := c.Workload.GetRemoteStorageDefinition()
	if remoteStorageDefinition == nil {
		c.logger.Warn("Could not compute meaningful timeout interval because scheduling policy is invalid.",
			zap.String("workload_id", c.Workload.GetId()),
			zap.String("workload_name", c.Workload.WorkloadName()),
			zap.String("session_id", c.SessionId),
			zap.String("event", evt.Name.String()))
		return (time.Minute * 2) + c.getAdjustedDuration(evt) // We make it a bit higher since we know I/O is on the critical path.
	}

	// Load the session and subsequently its current resource request.
	// We already checked that this existed in handleTrainingEventEnded.
	resourceRequest := c.Session.GetCurrentResourceRequest()
	if resourceRequest == nil {
		c.logger.Warn("Could not compute meaningful timeout interval because scheduling policy is invalid.",
			zap.String("workload_id", c.Workload.GetId()),
			zap.String("workload_name", c.Workload.WorkloadName()),
			zap.String("session_id", c.SessionId),
			zap.String("event", evt.Name.String()))
		return (time.Minute * 2) + c.getAdjustedDuration(evt) // We make it a bit higher since we know I/O is on the critical path.
	}

	vramBytes := resourceRequest.VRAM * 1000000000
	readTime := (vramBytes / float64(remoteStorageDefinition.DownloadRate)) * (1 + float64(remoteStorageDefinition.DownloadRateVariancePercentage))
	writeTime := (vramBytes / float64(remoteStorageDefinition.UploadRate)) * (1 + float64(remoteStorageDefinition.UploadRateVariancePercentage))
	expectedNetworkIoLatency := readTime + writeTime

	// Extra 30 seconds for whatever shenanigans need to occur.
	interval := (time.Second * 30) + (time.Second * time.Duration(expectedNetworkIoLatency)) + c.getAdjustedDuration(evt)

	c.logger.Debug("Computed timeout interval.",
		zap.String("workload_id", c.Workload.GetId()),
		zap.String("workload_name", c.Workload.WorkloadName()),
		zap.String("session_id", c.SessionId),
		zap.Float64("vram_gb", resourceRequest.VRAM),
		zap.Float64("vram_bytes", vramBytes),
		zap.String("remote_storage_definition", remoteStorageDefinition.String()),
		zap.String("event", evt.Name.String()))

	return interval
}

// waitForTrainingToEnd waits until we receive an "execute_request" from the kernel.
func (c *Client) waitForTrainingToEnd(ctx context.Context) error {
	select {
	case v := <-c.trainingStoppedChannel:
		{
			receivedResp := time.Now().UnixMilli()
			e2eLatency := time.Since(time.UnixMilli(c.lastTrainingSubmittedAt.UnixMilli()))

			switch v.(type) {
			case error:
				{
					err := v.(error)
					c.logger.Warn("Session failed to stop training...",
						zap.String("workload_id", c.Workload.GetId()),
						zap.String("workload_name", c.Workload.WorkloadName()),
						zap.String("session_id", c.SessionId),
						zap.Duration("e2e_latency", e2eLatency),
						zap.Error(err))

					return nil // to prevent workload from ending outright
				}
			case jupyter.KernelMessage:
				{
					reply := v.(jupyter.KernelMessage)
					content := reply.GetContent().(map[string]interface{})

					val := content["execution_start_unix_millis"]
					execStartedTimeUnixMillis := int64(val.(float64))

					val = content["execution_finished_unix_millis"]
					execEndedTimeUnixMillis := int64(val.(float64))

					execTimeMillis := execEndedTimeUnixMillis - execStartedTimeUnixMillis

					c.Workload.RecordSessionExecutionTime(c.SessionId, execTimeMillis)

					delay := receivedResp - execEndedTimeUnixMillis

					c.Workload.UpdateStatistics(func(stats *Statistics) {
						stats.TotalReplyLatenciesMillis = append(stats.TotalReplyLatenciesMillis, delay)
						stats.TotalReplyLatencyMillis += delay
					})

					c.logger.Debug("Session stopped training",
						zap.String("session_id", c.SessionId),
						zap.String("workload_id", c.Workload.GetId()),
						zap.String("workload_name", c.Workload.WorkloadName()),
						zap.Int64("exec_time_millis", execTimeMillis),
						zap.Duration("e2e_latency", e2eLatency))

					return nil
				}
			default:
				{
					c.logger.Error("Received unexpected response via 'training-stopped' channel.",
						zap.String("workload_id", c.Workload.GetId()),
						zap.String("workload_name", c.Workload.WorkloadName()),
						zap.String("session_id", c.SessionId),
						zap.Duration("e2e_latency", e2eLatency),
						zap.Any("response", v))

					return fmt.Errorf("unexpected response via 'training-stopped' channel")
				}
			}
		}
	case <-ctx.Done():
		{
			err := ctx.Err()
			if err != nil {
				c.logger.Error("Timed-out waiting for \"execute_reply\" message while stopping training.",
					zap.String("session_id", c.SessionId),
					zap.String("workload_id", c.Workload.GetId()),
					zap.String("workload_name", c.Workload.WorkloadName()),
					zap.Duration("time_elapsed", time.Since(c.lastTrainingSubmittedAt)),
					zap.Error(err))

				// We'll just return (nothing) so that the workload doesn't end.
				return err
			}

			// No error attached to the context. Just log an error message without the error struct
			// and return an error of our own.
			c.logger.Error("Timed-out waiting for \"execute_reply\" message while stopping training.",
				zap.String("session_id", c.SessionId),
				zap.String("workload_id", c.Workload.GetId()),
				zap.String("workload_name", c.Workload.WorkloadName()),
				zap.Duration("time_elapsed", time.Since(c.lastTrainingSubmittedAt)))

			return jupyter.ErrRequestTimedOut
		}
	}
}

// handleSessionStoppedEvent handles a domain.EventSessionStopped *domain.Event.
func (c *Client) handleSessionStoppedEvent(evt *domain.Event) error {
	c.logger.Debug("Stopping session.",
		zap.String("workload_id", c.Workload.GetId()),
		zap.String("workload_name", c.Workload.WorkloadName()),
		zap.String("session_id", c.SessionId))

	err := c.kernelSessionManager.StopKernel(c.SessionId)
	if err != nil {
		c.logger.Error("Error encountered while stopping session.",
			zap.String("workload_id", c.Workload.GetId()),
			zap.String("workload_name", c.Workload.WorkloadName()),
			zap.String("session_id", c.SessionId),
			zap.Error(err))
		return err
	}

	c.logger.Debug("Successfully stopped session.",
		zap.String("workload_id", c.Workload.GetId()),
		zap.String("workload_name", c.Workload.WorkloadName()),
		zap.String("session_id", c.SessionId))

	// Attempt to update the Prometheus metrics for Session lifetime duration (in seconds).
	sessionLifetimeDuration := time.Since(c.Session.GetCreatedAt())
	metrics.PrometheusMetricsWrapperInstance.WorkloadSessionLifetimeSeconds.
		With(prometheus.Labels{"workload_id": c.Workload.GetId()}).
		Observe(sessionLifetimeDuration.Seconds())

	c.Workload.SessionStopped(c.SessionId, evt)
	c.logger.Debug("Handled SessionStopped event.",
		zap.String("workload_id", c.Workload.GetId()),
		zap.String("workload_name", c.Workload.WorkloadName()),
		zap.String("session_id", c.SessionId))

	c.waitGroup.Done()

	return nil
}
