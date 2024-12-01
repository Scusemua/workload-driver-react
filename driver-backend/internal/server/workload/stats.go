package workload

import (
	"github.com/scusemua/workload-driver-react/m/v2/internal/domain"
	"time"
)

// Statistics encapsulates runtime statistics and metrics about a workload that are maintained within the
// dashboard backend (rather than within Prometheus).
type Statistics struct {
	*ClusterStatistics

	RegisteredTime time.Time `json:"registered_time" csv:"-"`
	StartTime      time.Time `json:"start_time" csv:"-"`
	EndTime        time.Time `json:"end_time" csv:"-"`

	Id                                  string                  `json:"id" csv:"id" `
	Name                                string                  `json:"name" csv:"name" `
	AggregateSessionDelayMillis         int64                   `json:"aggregate_session_delay_ms" csv:"aggregate_session_delay_ms"`
	CumulativeNumStaticTrainingReplicas int                     `json:"cumulative_num_static_training_replicas" csv:"cumulative_num_static_training_replicas"  csv:""`
	CurrentTick                         int64                   `json:"current_tick" csv:"current_tick"`
	EventsProcessed                     []*domain.WorkloadEvent `json:"events_processed"  csv:"events_processed"`
	NextEventExpectedTick               int64                   `json:"next_event_expected_tick"  csv:"next_event_expected_tick"`
	NextExpectedEventName               domain.EventName        `json:"next_expected_event_name"  csv:"next_expected_event_name"`
	NextExpectedEventTarget             string                  `json:"next_expected_event_target"  csv:"next_expected_event_target"`
	NumActiveSessions                   int64                   `json:"num_active_sessions"  csv:"num_active_sessions"`
	NumActiveTrainings                  int64                   `json:"num_active_trainings"  csv:"num_active_trainings"`
	NumDiscardedSessions                int                     `json:"num_discarded_sessions"  csv:"num_discarded_sessions"`
	NumEventsProcessed                  int64                   `json:"num_events_processed"  csv:"num_events_processed"`
	NumSampledSessions                  int                     `json:"num_sampled_sessions"  csv:"num_sampled_sessions"`
	NumSessionsCreated                  int64                   `json:"num_sessions_created"  csv:"num_sessions_created"`
	NumSubmittedTrainings               int64                   `json:"num_submitted_trainings"  csv:"num_submitted_trainings"` // NumSubmittedTrainings is the number of trainings that have been submitted but not yet started.
	NumTasksExecuted                    int64                   `json:"num_tasks_executed"  csv:"num_tasks_executed"`
	SessionsSamplePercentage            float64                 `json:"sessions_sample_percentage"  csv:"sessions_sample_percentage"`
	SimulationClockTimeStr              string                  `json:"simulation_clock_time"  csv:"simulation_clock_time"`
	TickDurationsMillis                 []int64                 `json:"tick_durations_milliseconds"  csv:"tick_durations_milliseconds"`
	TimeElapsed                         time.Duration           `json:"time_elapsed"  csv:"time_elapsed"` // Computed at the time that the data is requested by the user. This is the time elapsed SO far.
	TimeElapsedStr                      string                  `json:"time_elapsed_str"  csv:"time_elapsed_str"`
	TimeSpentPausedMillis               int64                   `json:"time_spent_paused_milliseconds"  csv:"time_spent_paused_milliseconds"`
	TotalNumSessions                    int                     `json:"total_num_sessions" csv:"total_num_sessions"  csv:"total_num_sessions"`
	TotalNumTicks                       int64                   `json:"total_num_ticks"  csv:"total_num_ticks"`
	WorkloadDuration                    time.Duration           `json:"workload_duration"  csv:"workload_duration"` // The total time that the workload executed for. This is only set once the workload has completed.
	WorkloadState                       State                   `json:"workload_state"  csv:"workload_state"`
	WorkloadType                        Kind                    `json:"workload_type"  csv:"workload_type"`
}

type ClusterStatistics struct {
	///////////
	// Hosts //
	///////////

	Hosts            int `json:"hosts" csv:"hosts"`
	NumDisabledHosts int `json:"num_disabled_hosts" csv:"num_disabled_hosts"`
	NumEmptyHosts    int `csv:"NumEmptyHosts" json:"NumEmptyHosts"` // The number of Hosts with 0 sessions/containers scheduled on them.

	// The amount of time hosts have spent not idling throughout the entire simulation
	CumulativeHostActiveTime float64 `csv:"CumulativeHostActiveTimeSec" json:"CumulativeHostActiveTimeSec"`
	// The amount of time hosts have spent idling throughout the entire simulation.
	CumulativeHostIdleTime float64 `csv:"CumulativeHostIdleTimeSec" json:"CumulativeHostIdleTimeSec"`
	// The aggregate, cumulative lifetime of ALL hosts provisioned at some point during the simulation.
	AggregateHostLifetime float64 `csv:"AggregateHostLifetimeSec" json:"AggregateHostLifetimeSec"`
	// The aggregate, cumulative lifetime of the hosts that are currently running.
	AggregateHostLifetimeOfRunningHosts float64 `csv:"AggregateHostLifetimeOfRunningHostsSec" json:"AggregateHostLifetimeOfRunningHostsSec"`

	// The total (cumulative) number of hosts provisioned during the simulation run.
	CumulativeNumHostsProvisioned int `csv:"CumulativeNumHostsProvisioned" json:"CumulativeNumHostsProvisioned"`
	// The total amount of time spent provisioning hosts.
	CumulativeTimeProvisioningHosts float64 `csv:"CumulativeTimeProvisioningHostsSec" json:"CumulativeTimeProvisioningHostsSec"`

	///////////////
	// Resources //
	///////////////

	SpecCPUs        float64 `csv:"SpecCPUs" json:"SpecCPUs"`
	SpecGPUs        float64 `csv:"SpecGPUs" json:"SpecGPUs"`
	SpecMemory      float64 `csv:"SpecMemory" json:"SpecMemory"`
	SpecVRAM        float64 `csv:"SpecVRAM" json:"SpecVRAM"`
	IdleCPUs        float64 `csv:"IdleCPUs" json:"IdleCPUs"`
	IdleGPUs        float64 `csv:"IdleGPUs" json:"IdleGPUs"`
	IdleMemory      float64 `csv:"IdleMemory" json:"IdleMemory"`
	IdleVRAM        float64 `csv:"IdleVRAM" json:"IdleVRAM"`
	PendingCPUs     float64 `csv:"PendingCPUs" json:"PendingCPUs"`
	PendingGPUs     float64 `csv:"PendingGPUs" json:"PendingGPUs"`
	PendingMemory   float64 `csv:"PendingMemory" json:"PendingMemory"`
	PendingVRAM     float64 `csv:"PendingVRAM" json:"PendingVRAM"`
	CommittedCPUs   float64 `csv:"CommittedCPUs" json:"CommittedCPUs"`
	CommittedGPUs   float64 `csv:"CommittedGPUs" json:"CommittedGPUs"`
	CommittedMemory float64 `csv:"CommittedMemory" json:"CommittedMemory"`
	CommittedVRAM   float64 `csv:"CommittedVRAM" json:"CommittedVRAM"`

	DemandCPUs   float64 `csv:"DemandCPUs" json:"DemandCPUs"`
	DemandMemMb  float64 `csv:"DemandMemMb" json:"DemandMemMb"`
	DemandGPUs   float64 `csv:"DemandGPUs" json:"DemandGPUs"`
	DemandVRAMGb float64 `csv:"DemandVRAMGb" json:"DemandVRAMGb"`
	//GPUUtil    float64 `csv:"GPUUtil" json:"GPUUtil"`
	//CPUUtil    float64 `csv:"CPUUtil" json:"CPUUtil"`
	//MemUtil    float64 `csv:"MemUtil" json:"MemUtil"`
	//VRAMUtil   float64 `csv:"VRAMUtil" json:"VRAMUtil"`
	//CPUOverload int `csv:"CPUOverload" json:"CPUOverload"`

	/////////////////////////////////
	// Static & Dynamic Scheduling //
	/////////////////////////////////

	SubscriptionRatio float64 `csv:"SubscriptionRatio" json:"SubscriptionRatio"`

	////////////////////////
	// Dynamic Scheduling //
	////////////////////////

	Rescheduled       int32 `csv:"Rescheduled" json:"Rescheduled"`
	Resched2Ready     int32 `csv:"Resched2Ready" json:"Resched2Ready"`
	Migrated          int32 `csv:"Migrated" json:"Migrated"`
	Preempted         int32 `csv:"Preempted" json:"Preempted"`
	OnDemandContainer int   `csv:"OnDemandContainers" json:"OnDemandContainers"`
	IdleHostsPerClass int32 `csv:"IdleHosts" json:"IdleHosts"`

	//////////////
	// Sessions //
	//////////////

	CompletedTrainings int32 `csv:"CompletedTrainings" json:"CompletedTrainings"`
	// The Len of Cluster::Sessions (which is of type *SessionManager).
	// This includes all Sessions that have not been permanently stopped.
	NumNonTerminatedSessions int `csv:"NumNonTerminatedSessions" json:"NumNonTerminatedSessions"`
	// The number of Sessions that are presently idle, not training.
	NumIdleSessions int `csv:"NumIdleSessions" json:"NumIdleSessions"`
	// The number of Sessions that are presently actively-training.
	NumTrainingSessions int `csv:"NumTrainingSessions" json:"NumTrainingSessions"`
	// The number of Sessions in the STOPPED state.
	NumStoppedSessions int `csv:"NumStoppedSessions" json:"NumStoppedSessions"`
	// The number of Sessions that are actively running (but not necessarily training), so includes idle sessions.
	// Does not include evicted, init, or stopped sessions.
	NumRunningSessions int `csv:"NumRunningSessions" json:"NumRunningSessions"`

	// The amount of time that Sessions have spent idling throughout the entire simulation.
	CumulativeSessionIdleTime float64 `csv:"CumulativeSessionIdleTimeSec" json:"CumulativeSessionIdleTimeSec"`
	// The amount of time that Sessions have spent training throughout the entire simulation. This does NOT include replaying events.
	CumulativeSessionTrainingTime float64 `csv:"CumulativeSessionTrainingTimeSec" json:"CumulativeSessionTrainingTimeSec"`
	// The aggregate lifetime of all sessions created during the simulation (before being suspended).
	AggregateSessionLifetime float64 `csv:"AggregateSessionLifetimeSec" json:"AggregateSessionLifetimeSec"`
}
