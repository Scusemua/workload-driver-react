package domain

import (
	"flag"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"

	configKit "github.com/gookit/config/v2"
	"github.com/gookit/config/v2/yaml"
	"github.com/imdario/mergo"
	"github.com/mitchellh/mapstructure"
)

const (
	OptionName    = "name"
	OptionDefault = "default"
	OptionDesc    = "description"
)

var (
	Config  *WorkloadConfig = nil
	Verbose bool
	Months  = []string{"jan", "feb", "mar", "apr", "may", "jun", "jul", "aug", "sep", "oct", "nov", "dec"}
)

type WorkloadConfig struct {
	YAML                         string `name:"yaml" description:"Path to config file in the yml format."`
	TraceStep                    int64  `name:"trace-step" description:"Default interval, in seconds, of two consecutive trace readings."`
	GPUTraceFile                 string `name:"gputrace" description:"File path of GPU utilization trace."`
	GPUTraceStep                 int64  `name:"gputrace-step" description:"Interval, in seconds, of two consecutive trace readings of GPU."`
	GPUMappingFile               string `name:"gpumap" description:"File path of GPU idx/pod map."`
	MaxSessionCpuFile            string `name:"max-session-cpu-file" description:"File obtained during a 'pre-run' of the simulator that simply reads thru the trace data. Contains the lifetime-maximum CPU utilization of each session."`        // File obtained during a 'pre-run' of the simulator that simply reads thru the trace data. Contains the max CPU used by each session.
	MaxSessionMemFile            string `name:"max-session-mem-file" description:"File obtained during a 'pre-run' of the simulator that simply reads thru the trace data. Contains the lifetime-maximum memory (in bytes) used by each session."` // // File obtained during a 'pre-run' of the simulator that simply reads thru the trace data. Contains the max memory (in bytes) used by each session.
	MaxSessionGpuFile            string `name:"max-session-gpu-file" desciption:"File obtained during a 'pre-run' of the simulator that simply reads thru the trace data. Contains the lifetime-maximum GPU utilization of each session."`         // File obtained during a 'pre-run' of the simulator that simply reads thru the trace data. Contains the max CPU used by each session.
	MaxTaskCpuFile               string `name:"max-task-cpu-file" description:"File obtained during a 'pre-run' of the simulator that simply reads thru the trace data. Contains the max CPU utilization achieved within each individual training task."`
	MaxTaskMemFile               string `name:"max-task-mem-file" description:"File obtained during a 'pre-run' of the simulator that simply reads thru the trace data. Contains the max memory (in bytes) used within each individual training task."`
	MaxTaskGpuFile               string `name:"max-task-gpu-file" description:"File obtained during a 'pre-run' of the simulator that simply reads thru the trace data. Contains the max GPU utilization achieved within each individual training task."`
	CPUTraceFile                 string `name:"cputrace" description:"File path of CPU counter trace."`
	CPUTraceStep                 int64  `name:"cputrace-step" description:"Interval, in seconds, of two consecutive trace readings of CPU."`
	CPUMappingFile               string `name:"cpumap" description:"File path of CPU idx/pod map."`
	CPUDowntime                  string `name:"cpudown" description:"CPU trace downtime."`
	MemTraceFile                 string `name:"memtrace" description:"File path of memory usage trace."`
	MemTraceStep                 int64  `name:"memtrace-step" description:"Interval, in seconds, of two consecutive trace readings of memory."`
	MemMappingFile               string `name:"memmap" description:"File path of memory idx/pod map."`
	FromMonth                    string `name:"from-month" description:"Month the trace starts if the path of trace file contains placeholder."`
	ToMonth                      string `name:"to-month" description:"Month the trace ends if the path of trace file contains placeholder."`
	Output                       string `name:"o" description:"Path to output csv data."`
	ClusterStatsOutput           string `name:"cluster-stats-output" description:"File to output cluster stats as a CSV"`
	OutputSessions               string `name:"output-sessions" description:"Path to output all sessions."`
	OutputTasks                  string `name:"output-tasks" description:"Path to output all tasks."`
	OutputTaskIntervals          string `name:"output-task-intervals" description:"Path to output all task intervals."`
	OutputReschedIntervals       string `name:"output-reschedule-intervals" description:"Path to output reschedule intervals."`
	InspectPod                   string `name:"pod" description:"Inspect the behavior of specified pod."`
	InspectGPU                   bool   `name:"pod-gpu" description:"Inspect the gpu readings of specified pod."`
	InspectionClass              string `name:"pod-class" description:"The type of behavior will be inspected. Choose from trace or entity"`
	Debug                        bool   `name:"debug" description:"Display debug logs."`
	Verbose                      bool   `name:"v" description:"Display verbose logs."`
	Seed                         int64  `name:"seed" description:"Random seed to reproduce simulation."`
	LastTimestamp                int64  `name:"last-timestamp" description:"Epoch Unix Timestamp denotating the last timestamp for which events will be generated. Any events beyond that point will be discarded."`
	EvictHostOnLastContainerStop int    `name:"evict-host-on-last-container-stop" description:"Override the default settings for whatever Scheduler you're using and force a value for this parameter. -1 to force false, 0 to leave as default for the particular scheduler, and 1 to force true."`

	/////////////////////////////
	// Compute resource policy //
	/////////////////////////////
	ComputeResourcePolicy string `name:"compute-resource-policy" description:"Defines the different compute resources available to the cluster."`
	FractionServerless    string `name:"fraction-serverless" description:"Fraction of compute resource served by cloud functions."`
	FractionOnDemand      string `name:"fraction-on-demand" description:"Fraction of the serverful compute resource that is composed of on-demand virtual machines."`

	///////////////////////
	// Spot reclamations //
	///////////////////////
	// Determines what happens when a spot reclamation occurs. Options: 'terminate', 'migrate'
	SpotReclamationHandler string `name:"spot-reclaimation-handler" description:"Determines what happens when a spot reclamation occurs. Options: 'terminate', 'migrate'"`
	SpotReclamationConfig  string `name:"spot-reclamation-policy" description:"Defines how spot instances are reclaimed."`

	///////////////////////
	// General execution //
	///////////////////////
	// Options are 0 (i.e., 'pre') and 1 (i.e., 'standard'). With 'pre', the Simulator will not process any events; it will simply parse
	// the trace to extract the CPU, GPU, and Memory readings. With 'standard', the Simulator will actually simulate the workload.
	ExecutionMode                     int  `name:"execution-mode" description:"Options are 0 (i.e., 'pre') and 1 (i.e., 'standard'). With 'pre', the Simulator will not process any events; it will simply parse the trace to extract the CPU, GPU, and Memory readings. With 'standard', the Simulator will actually simulate the workload."`
	MaxTaskDurationSec                int  `name:"max-task-duration-seconds" description:"The maximum length of a task. If a task with length >= this value is executed, then the associated Session will be terminated once the event completes."`
	ContinueUntilExplicitlyTerminated bool `name:"continue-until-explicit-termination" description:""`
	// TrainingMode                      string `name:"training-mode" description:"Determines where training events are processed. Options include: 'local', 'offload-faas', 'offload-serverful', or 'offload-hybrid'."` // Determines where training events are processed. Options include: 'local', 'offload-faas', 'offload-serverful', or 'offload-hybrid'.
	NotebookServerMemoryGB string `name:"notebook-server-memory-gb" description:"The number of gigabytes allocated to notebook servers (when they're not permitted to train directly/locally)."` // The number of gigabytes allocated to notebook servers (when they're not permitted to train directly/locally).
	// If true, then the PendingHostQueue is enabled, which keeps tracks of pending hosts and allocates resources to them prior to when they're actually provisioned.
	// TrackResourcesOnPendingHosts bool `name:"track-resources-on-pending-hosts" description:"If true, then the PendingHostQueue is enabled, which keeps tracks of pending hosts and allocates resources to them prior to when they're actually provisioned."`

	///////////////////
	// Checkpointing //
	///////////////////
	UseCheckpointing         bool   `name:"use-checkpointing" description:"If set to true, then simulate the use of checkpointing when migrating containers from one host to another following a reclamation event. Without checkpointing, all state is lost and a container's events must be replayed from the beginning once the container is rescheduled onto another (possibly new) host."`
	CheckpointMinDelayMillis string `name:"checkpoint-min-delay-millis" description:"The minimum delay, in milliseconds, that occurs when using checkpointing. This delay simulates the time spent retrieving the checkpointed state for a rescheduled container following a spot instance reclamation."`
	CheckpointMaxDelayMillis string `name:"checkpoint-max-delay-millis" description:"The maximum delay, in milliseconds, that occurs when using checkpointing. This delay simulates the time spent retrieving the checkpointed state for a rescheduled container following a spot instance reclamation"`

	// MinimumHosts            int `name:"cluster-min-hosts" description:"The minimum number of actively-running hosts that this Cluster maintains. If a host is reclaimed or stopped and the number of actively-running hosts falls below this number, then a new host will be provisioned."`
	MinHostProvisionDelayMs  string `name:"min-host-provisioning-delay" description:"The minimum amount of time required to provision a new host (in milliseconds)."`
	MaxHostProvisionDelayMs  string `name:"max-host-provisioning-delay" description:"The maximum amount of time required to provision a new host (in milliseconds)."`
	KeepAliveIntervalSeconds int64  `name:"keep-alive-interval-seconds" description:"The duration, in milliseconds, of the interval after which an idle Session is evicted."`

	//////////////////////
	// Resource Credits //
	//////////////////////
	UseResourceCredits      bool    `name:"use-resource-credits" description:"If true, then use resource credits as the billing model, rather than raw USD."`
	InitialCreditBalance    float64 `name:"initial-resource-credit-balance" description:"The number of resource credits a new user will initially have available."`
	ResourceCreditCPU       float64 `name:"resource-credit-vcpus" description:"The amount of vCPUs made available to a user for one hour by a single resource credit."`
	ResourceCreditGPU       float64 `name:"resource-credit-gpus" description:"The amount of GPUs made available to a user for one hour by a single resource credit."`
	ResourceCreditMemMB     float64 `name:"resource-credit-mem-mb" description:"The amount of memory (i.e., RAM), in megabytes (MB), made available to a user for one hour by a single resource credit."`
	ResourceCreditCostInUSD float64 `name:"resource-credit-cost-usd" description:"The cost-equivalent of a single resource credit in USD."`

	////////////////
	// Scheduling //
	////////////////
	CpuSchedulingWeight     float64 `name:"cpu-schedule-weight" description:"Value from 1.0 to 100.0 indicating how much weight to assign to CPU when scheduling Sessions onto Hosts."`
	GpuSchedulingWeight     float64 `name:"gpu-schedule-weight" description:"Value from 1.0 to 100.0 indicating how much weight to assign to GPU when scheduling Sessions onto Hosts."`
	MemorySchedulingWeight  float64 `name:"memory-schedule-weight" description:"Value from 1.0 to 100.0 indicating how much weight to assign to memory when scheduling Sessions onto Hosts."`
	HostScoreMetric         string  `name:"host-score-metric" description:"Scoring method to use for potential hosts when scheduling sessions. Valid options include 'LeastAllocated', 'MostAllocated', 'LeastAllocatable', and 'MostAllocatable'."` // Scoring metric to use for potential hosts when scheduling sessions. Valid options include "least-allocated" and "most-allocated".
	MinMigrationDelayMillis string  `name:"min-migration-delay" description:"The minimum delay (in milliseconds) incurred when migrating a Container."`
	MaxMigrationDelayMillis string  `name:"max-migration-delay" description:"The maximum delay (in milliseconds) incurred when migrating a Container."`
	PreemptionEnabled       bool    `name:"preemption-enabled" description:"Basically enables Jingyuan's dynamic policy."`
	Scheduler               string  `name:"scheduler" description:"Scheduler to use. Options include: 'non-replica', 'faas', 'static', and 'dynamic'."`
	RandomHostSpace         int     `name:"random-host-space" description:"Number of hosts to select from when finding a reschedule host in the dynamic scheduler."`
	// FindRescheduleHostMethod string  `name:"find-reschedule-host-method" description:"The method used to find reschedule hosts when migrating Containers. Options are 'v3' (Dynamic V3) or 'v4' (Dynamic V4)."`

	////////////////////////////////////////////
	// Instance type & host pool config paths //
	////////////////////////////////////////////
	ServerlessFunctionPoolConfigPath        string `name:"serverless-function-pool-config-path" description:"Path to the configuration file defining the serverless function pool to be used by the Cluster in the Simulator."`
	ServerlessFunctionDefinitionsConfigPath string `name:"serverless-function-definitions-config-path" description:"Path to the configuration file defining the different Serverless Functions available during the simulation."`
	ServerfulHostPoolConfigPath             string `name:"host-pool-config-path" description:"Path to the configuration file defining the serverful host pool to be used by the Cluster in the Simulator."`
	ServerfulInstanceTypesConfigPath        string `name:"serverful-instance-types-config-path" description:"Path to the configuration file defining the Instance Types to be available during the simulation."`

	//////////////////////////////
	// Warm & Cold Start Delays //
	//////////////////////////////
	MinRetentionPeriodMillis string `name:"serverless-min-retention" description:"The minimum amount of time that a warm serverless function will remain provisioned before being reclaimed by the cloud provider in milliseconds."` // The minimum amount of time that a warm serverless function will remain provisioned before being reclaimed by the cloud provider in milliseconds.
	MaxRetentionPeriodMillis string `name:"serverless-max-retention" description:"The maximum amount of time that a warm serverless function will remain provisioned before being reclaimed by the cloud provider in milliseconds."` // The maximum amount of time that a warm serverless function will remain provisioned before being reclaimed by the cloud provider in milliseconds.

	/////////////
	// Logging //
	/////////////
	DoHostLevelLogging       bool   `name:"do-host-level-logging" description:"If enabled, output host-level CSV files for every host."` // If enabled, output host-level CSV files for every host.
	DoHostPoolLevelLogging   bool   `name:"do-host-pool-level-logging" description:"If enabled, output host-pool-level CSV files for every host."`
	HostLogsEveryNTicks      int64  `name:"host-logs-every-n-ticks" description:"Output host-level CSV logs every N ticks. This parameter is N."`     // Output host-level CSV logs every N ticks. This parameter is N."
	LogOutputFile            string `name:"log-output-file" description:"If specified, then log output will go to this file, rather than to STDOUT."` // If specified, then log output will go to this file, rather than to STDOUT.
	DisplayCostIntervalTicks int64  `name:"display-cost-interval" description:"Defines the frequency (every N ticks) at which the simulator outputs the running provider-side and tenant-side cost. The cost is logged at the very end of a tick."`
	// If true, then sessions will ALWAYS be migrated to an on-demand host following a spot migration.
	// AlwaysMigrateToOnDemandAfterSpotReclamation bool `name:"spot-migration-force-on-demand" description:"If true, then sessions will ALWAYS be migrated to an on-demand host following a spot migration."`
	// If specified, will write CPU profile to the specified file in the output directory.
	CpuProfileFile string `name:"cpu-profile-file" description:"If specified, will write CPU profile to the specified file in the output directory."`
	// Deprecated.
	PreTaskFile string `name:"pre-task-file" description:"Task CSV file containing a 'MaxTaskCPU' column of max CPU utilization and 'MaxSessionMemory' column of max memory usage during each task."`
	// Deprecated.
	PreSessionFile              string `name:"pre-session-file" description:"Session CSV file containing a 'MaxSessionCPU' column of max CPU utilization and 'MaxSessionMemory' column of max memory usage during each session."`
	UtilizationSamplingInterval int64  `name:"utilization-sampling-interval" description:"Sample utilizations from running Sessions every 'UtilizationSamplingInterval' ticks."`
	EnableDebugLogAt            int64  `name:"enable-debug-log-at-time" description:"Enable debug logging at a certain timestamp."`

	////////////////////
	// Host Reclaimer //
	////////////////////
	// Specify which host reclaimer to use. Options: 'idle', 'variable-idle'
	HostReclaimer string `name:"host-reclaimer" description:"Specify which host reclaimer to use. Options: 'idle', 'variable-idle'"`
	// The base interval of time a ServerfulHost can be idle after which it will be reclaimed by an IdleHostReclaimer.
	IdleHostReclaimerBaseIntervalSec int `name:"idle-host-reclaimer-base-interval-seconds" description:"The base interval of time a ServerfulHost can be idle after which it will be reclaimed by an IdleHostReclaimer."`
	// Hard-cap on the idle interval. Used for variable host reclaimers.
	VariableIdleHostReclaimerMaxIntervalSec int `name:"variable-idle-host-reclaimer-max-interval-seconds" description:"Hard-cap on the idle interval. Used for variable host reclaimers."`
	// Target resource that is used to scale the idle interval. Valid options are 'gpu', 'cpu', or 'memory'.
	VariableIdleHostReclaimerTargetResource string `name:"variable-idle-host-reclaimer-target-resource" description:"Target resource that is used to scale the idle interval. Valid options are 'gpu', 'cpu', or 'memory'."`
	// The idle interval for an instance is scaled-up for every `baseResourceAmount` of the target resource that it has.
	VariableIdleHostReclaimerBaseResourceAmount int64 `name:"variable-idle-host-reclaimer-base-resource-amount" description:"The idle interval for an instance is scaled-up for every 'baseResourceAmount' of the target resource that it has."`
	// If enabled, then ables the base idle interval to be scaled down. This can occur if an instance type has less of a resource than the configured 'baseResourceAmount' parameter.
	VariableIdleHostReclaimerAllowScalingDown bool `name:"variable-idle-host-reclaimer-allow-scaling-down" description:"If enabled, then ables the base idle interval to be scaled down. This can occur if an instance type has less of a resource than the configured 'baseResourceAmount' parameter."`

	////////////////////////////
	// Instance type selector //
	////////////////////////////
	// Used to determine which instance type to use when creating a new instance. Options are 'largest-cpu', 'smallest-cpu', 'smallest-gpu', and 'most-least-cpu-gpu-threshold'.
	InstanceTypeSelector           string `name:"instance-type-selector" description:"Used to determine which instance type to use when creating a new serverful instance. Options are 'largest-cpu', 'smallest-cpu', 'smallest-gpu', and 'most-least-cpu-gpu-threshold'."`
	ServerlessInstanceTypeSelector string `name:"serverless-instance-type-selector" description:"Used to determine which instance type to use when creating a new serverless function. Options are 'largest-cpu', 'smallest-cpu', 'smallest-gpu', and 'most-least-cpu-gpu-threshold'."`
	// This should be specified when using the 'most-least-cpu-gpu-threshold' setting for the 'InstanceTypeSelector' parameter.
	// Sessions requesting >= this many GPUs will use the 'largest-cpu' method for instance type selection, whereas sessions selecting < this will use the 'smallest-cpu' method.
	NewMostLeastCPU_GPUThresholdSelector_Threshold int64 `name:"most-least-cpu-gpu-threshold" description:"This should be specified when using the 'most-least-cpu-gpu-threshold' setting for the 'InstanceTypeSelector' parameter. Sessions requesting >= this many GPUs will use the 'largest-cpu' method for instance type selection, whereas sessions selecting < this will use the 'smallest-cpu' method."`

	///////////////////
	// Monetary cost //
	///////////////////
	// By default, users are charged the price-per-hour of the instance that their session is running on. This parameter adjusts/scales the rate at which users incurred cost. For example, 2.0 would charge users double the hourly rate.
	ServerfulUserCostMultiplier string `name:"serverful-user-cost-multiplier" description:"By default, users are charged the price-per-hour of the instance that their session is running on. This parameter adjusts/scales the rate at which users incurred cost. For example, 2.0 would charge users double the hourly rate."`
	// Adjusts how much the provider is billed for a host relative to the cost-per-hour of the host's instance type.
	ServerfulProviderCostMultiplier string `name:"serverful-provider-cost-modifier" description:"Adjusts how much the provider is billed for a host relative to the cost-per-hour of the host's instance type."`
	// Adjusts the rate at which users are charged when using serverless functions.
	ServerlessUserCostMultiplier string `name:"faas-user-cost-multiplier" description:"Adjusts the rate at which users are charged when using serverless functions."`
	// Adjusts the rate at which providers are charged for running serverless functions.
	ServerlessProviderCostMultiplier string `name:"faas-provider-cost-multiplier" description:"Adjusts the rate at which providers are charged for running serverless functions."`
	// Determines the billing model by which users are charged for serverful hosts or virtual machines. Options include 'gpu-fraction' and 'host-cph'.
	ServerfulCostModel string `name:"serverful-cost-model" description:"Determines the billing model by which users are charged for serverful hosts or virtual machines. Options include 'gpu-fraction' and 'host-cph'."`
	// Determines the billing model by which users are charged for serverless functions. Options include 'alibaba' and 'aws-lambda'.
	FaasCostModel string `name:"faas-cost-model" description:"Determines the billing model by which users are charged for serverless functions. Options include 'alibaba' and 'aws-lambda'."`
	// By default, sessions reserve 'NUM_GPUS' GPUs when being scheduled. If this property is enabled, then sessions will instead reserve 'NUM_GPUs' * 'MAX_GPU_UTIL'.
	// This will lead to many sessions reserving fewer GPUs than when this property is disabled (default).
	AdjustGpuReservations bool `name:"adjust-gpu-reservations" description:"By default, sessions reserve 'NUM_GPUS' GPUs when being scheduled. If this property is enabled, then sessions will instead reserve 'NUM_GPUs' * 'MAX_GPU_UTIL'. This will lead to many sessions reserving fewer GPUs than when this property is disabled (default)."`
	// If true, then use one-year reserved pricing for VMs satisfying the 'minimum capacity' requirement of the resource pool.
	// If the configuration specifies "true" for both this property and the `UseThreeYearReservedPricingForMinimumCapacity` property, then three-year reserved pricing will be used.
	UseOneYearReservedPricingForMinimumCapacity bool `name:"use-one-year-reserved-pricing-for-min-capacity" description:"If true, then use one-year reserved pricing for VMs satisfying the 'minimum capacity' requirement of the resource pool. If the configuration specifies 'true' for both this property and the 'UseThreeYearReservedPricingForMinimumCapacity' property, then three-year reserved pricing will be used."`
	// If true, then use three-year reserved pricing for VMs satisfying the 'minimum capacity' requirement of the resource pool.
	// If the configuration specifies "true" for both this property and the `UseThreeYearReservedPricingForMinimumCapacity` property, then three-year reserved pricing will be used.
	UseThreeYearReservedPricingForMinimumCapacity bool `name:"use-three-year-reserved-pricing-for-min-capacity" description:"If true, then use three-year reserved pricing for VMs satisfying the 'minimum capacity' requirement of the resource pool. If the configuration specifies 'true' for both this property and the 'UseThreeYearReservedPricingForMinimumCapacity' property, then three-year reserved pricing will be used."`
	// If true, increase the duration of training events processed on serverless functions (relative to the maximum RAM utilization of the given training event) in order to simulate the overhead of reading data from cloud storage.
	SimulateDataTransferLatency bool `name:"simulate-data-transfer-latency" description:"If true, increase the duration of training events processed on serverless functions (relative to the maximum RAM utilization of the given training event) in order to simulate the overhead of reading data from cloud storage."`
	// If true, bill users for data transfer costs for reading data from cloud storage when training using serverless functions. The cost incurred is a function of the maximum RAM utilization of the given training event.
	SimulateDataTransferCost       bool   `name:"simulate-data-transfer-cost" description:"If true, bill users for data transfer costs for reading data from cloud storage when training using serverless functions. The cost incurred is a function of the maximum RAM utilization of the given training event."`
	BillUsersForNonActiveReplicas  bool   `name:"bill-inactive-replicas" description:"If true, then charge users for actively-scheduled training replicas that are not processing training events."`
	NonActiveReplicaCostMultiplier string `name:"inactive-replica-cost-multiplier" description:"Adjusts the rate at which we bill users for actively-scheduled training replicas that are not processing training events. This is only used when 'BillUsersForNonActiveReplicas' is true."`

	//////////////////////
	// Recovery handler //
	//////////////////////
	// The method for computing the replay delay when evicting a container. Options include 'last-n', 'all', 'replay-if-idle-less-than', and 'replay-if-training'.
	// The 'last-n' option uses the previous n training times as the replay delay, whereas the 'all' option uses all of the completed training times.
	// The 'replay-if-training' policy will only prompt a container to replay events if it was evicted while actively training. It is combined with another base policy to handle the replay interval when the criterion is satifised.
	// The 'replay-if-idle-less-than' policy will only prompt a container to replay events if it was evicted while actively training OR if it had been idle for < a configurable interval. It is combined with another base policy to handle the replay interval when the criteria are satifised.
	RecoveryHandler string `name:"recovery-handler" description:"The method for computing the replay delay when evicting a container. Options include 'last-n' and 'all'. The 'last-n' option uses the previous n training times as the replay delay, whereas the 'all' option uses all of the completed training times."`
	// Used when the 'last-n' replay delay method is selected. Sets the value of n for that method.
	RecoveryHandlerLastN int `name:"recovery-handler-last-n" description:"Used when the 'last-n' replay delay method is selected. Sets the value of n for that method."`
	// The recovery handler used by one of the 'composite' handlers (such as '' or '') when the replay criteria are satisfied.
	UnderlyingRecoveryHandler string `name:"underlying-recovery-handler" description:"The recovery handler used by one of the 'composite' handlers (such as 'replay-if-training' or 'replay-if-idle-less-than') when the replay criteria are satisfied."`
	// When using the 'replay-if-idle-less-than' recovery handlers, this is the threshold (in seconds) after which an idle Session will NOT have to replay events if rescheduled.
	RecoveryIdleThresholdSeconds int `name:"recovery-idle-threshold" description:"When using the 'replay-if-idle-less-than' recovery handlers, this is the threshold (in seconds) after which an idle Session will NOT have to replay events if rescheduled."`

	OnlyChargeMixedContainersWhileTraining bool `name:"only-charge-mixed-containers-while-training" description:"When the training-mode is set to \"local\", only charge the user (using the configured pricing model, such as GPU-Proportional) when the notebook server is actively training. When the notebook server is not actively training, fall back to CPU/Memory-proportional."`

	/////////////////////////
	// Jingyuan's Policies //
	/////////////////////////
	// OversubscriptionEnabled       bool    `name:"oversubscription-enabled" description:"Allows resources on VCRs to be oversubscribed."`
	ScalingPolicy       string `name:"scaling-policy" description:"Determines how the cluster adds and removes resources. Options include 'basic' and 'jingyuan'. The basic policy simply relies on the configured host reclaimer and the Placer3D to provision new hosts in response to changes in demand."`
	NumTrainingReplicas int    `name:"num-training-replicas" description:"Defines the weighted number of replica training containers per Session. Weight = GPUs of Hosts / Required GPUs."`
	// MaxSubscribedRatio            float64 `name:"max-subscribed-ratio" description:"Maximum oversubscription ratio permitted."`
	SubscribedRatioUpdateInterval float64 `name:"subscribed-ratio-update-interval" description:"The interval to update the subscribed ratio."`
	ScalingFactor                 float64 `name:"scaling-factor" description:"Defines how many hosts the cluster will provision based on busy resources"`
	ScalingInterval               int     `name:"scaling-interval" description:"Interval to call validateCapacity, 0 to disable routing scaling."`
	ScalingLimit                  float64 `name:"scaling-limit" description:"Defines how many hosts the cluster will provision at maximum based on busy resources"`
	MaximumHostsToReleaseAtOnce   int     `name:"scaling-in-limit" description:"Sort of the inverse of the ScalingLimit parameter (maybe?)"`
	ScalingOutEnaled              bool    `name:"scaling-out-enabled" description:"If enabled, the scaling manager will attempt to over-provision hosts slightly so as to leave room for fluctation. If disabled, then the Cluster will exclusivel scale-out in response to real-time demand, rather than attempt to have some hosts available in the case that demand surges."`
	ScalingBufferSize             int     `name:"scaling-buffer-size" description:"Buffer size is how many extra hosts we provision so that we can quickly scale if needed."`

	/////////////////
	// Replication //
	/////////////////
	MinScheduledReplicaQuantity     int     `name:"min-scheduled-replica-quantity" description:"The minimum number of actively-scheduled training replicas that a Session must have in order in ordered to be considered fit to process training-events. This options is only used if 'quantity' is specified for the 'MinScheduledReplicaMetric' config parameter."` // The minimum number of actively-scheduled training replicas that a Session must have in order in ordered to be considered fit to process training-events. This options is only used if "quantity" is specified for the 'MinScheduledReplicaMetric' config parameter.
	MinScheduledReplicaFraction     float64 `name:"min-scheduled-replica-fraction" description:"The fraction of actively-scheduled training replicas that a Session must have in order in ordered to be considered fit to process training-events. This options is only used if 'fraction' is specified for the 'MinScheduledReplicaMetric' config parameter."`       // The fraction of actively-scheduled training replicas that a Session must have in order in ordered to be considered fit to process training-events. This options is only used if "fraction" is specified for the 'MinScheduledReplicaMetric' config parameter.
	MinScheduledReplicaFitnessCheck string  `name:"min-scheduled-replica-metric" description:"Indicates whether to go off of a numerical quantity ('quantity') of actively-scheduled training replicas or a fraction ('fraction') of actively-scheduled training replicas when determining if a Session is fit to process training-events."`                          // Indicates whether to go off of a numerical quantity ("quantity") of actively-scheduled training replicas or a fraction ("fraction") of actively-scheduled training replicas when determining if a Session is fit to process training-events.
}

func (opts *WorkloadConfig) CheckUsage() {
	var printInfo bool
	flag.BoolVar(&printInfo, "h", false, "help info?")

	oType := reflect.TypeOf(opts).Elem()
	oVal := reflect.ValueOf(opts).Elem()
	numField := oType.NumField()
	for i := 0; i < numField; i++ {
		field := oType.Field(i)
		if field.PkgPath != "" {
			continue
		}

		name := field.Tag.Get(OptionName)
		if name == "" {
			continue
		}
		desc := field.Tag.Get(OptionDesc)
		opt := oVal.Field(i)
		switch field.Type.Kind() {
		case reflect.Bool:
			flag.BoolVar(opt.Addr().Interface().(*bool), name, opt.Bool(), desc)
		case reflect.Int:
			flag.IntVar(opt.Addr().Interface().(*int), name, int(opt.Int()), desc)
		case reflect.Int64:
			flag.Int64Var(opt.Addr().Interface().(*int64), name, opt.Int(), desc)
		case reflect.Uint:
			flag.UintVar(opt.Addr().Interface().(*uint), name, uint(opt.Uint()), desc)
		case reflect.Uint64:
			flag.Uint64Var(opt.Addr().Interface().(*uint64), name, opt.Uint(), desc)
		case reflect.Float64:
			flag.Float64Var(opt.Addr().Interface().(*float64), name, opt.Float(), desc)
		case reflect.String:
			flag.StringVar(opt.Addr().Interface().(*string), name, opt.String(), desc)
		default:
			panic(fmt.Errorf("unsupprted config type: %v", field.Type.Kind()))
		}
	}

	flag.Parse()

	if printInfo {
		fmt.Fprintf(os.Stderr, "Usage: ./play [options] data_base_path\n")
		fmt.Fprintf(os.Stderr, "Available options:\n")
		flag.PrintDefaults()
		os.Exit(0)
	}

	if opts.YAML != "" {
		configKit.WithOptions(func(opt *configKit.Options) {
			opt.TagName = OptionName
			// DecoderConfig initialization is due a bug in configKit: no TagName will be applied if DecoderConfig is nil.
			// TODO: Fix the bug
			opt.DecoderConfig = &mapstructure.DecoderConfig{}
		})
		configKit.AddDriver(yaml.Driver)
		if err := configKit.LoadFiles(opts.YAML); err != nil {
			panic(err)
		}
		fileOpts := &WorkloadConfig{}
		if err := configKit.BindStruct("", fileOpts); err != nil {
			panic(err)
		}

		if err := mergo.Merge(opts, fileOpts, mergo.WithOverride); err != nil {
			panic(err)
		}
	}

	if opts.GPUTraceStep == 0 {
		opts.GPUTraceStep = opts.TraceStep
	}
	if opts.CPUTraceStep == 0 {
		opts.CPUTraceStep = opts.TraceStep
	}
	if opts.MemTraceStep == 0 {
		opts.MemTraceStep = opts.TraceStep
	}
	if opts.FromMonth != "" {
		opts.FromMonth = strings.ToLower(opts.FromMonth[:3])
	}
	if opts.ToMonth != "" {
		opts.ToMonth = strings.ToLower(opts.ToMonth[:3])
	}
}

func (opts *WorkloadConfig) NormalizeTracePaths(path string) []string {
	if opts.FromMonth == "" {
		return []string{path}
	}

	paths := make([]string, 0, len(Months))
	fromMonth := 0
	// Match the start month
	if opts.FromMonth != "" {
		for i := 0; i < len(Months); i++ {
			if Months[i] == opts.FromMonth {
				fromMonth = i
			}
		}
	}
	// Match the end month
	for i := 0; i < len(Months); i++ {
		idx := (fromMonth + i) % len(Months)
		paths = append(paths, fmt.Sprintf(path, Months[idx]))
		if Months[idx] == opts.ToMonth {
			return paths
		}
	}
	return paths
}

func (opts *WorkloadConfig) NormalizeDowntime(downtime string) []int64 {
	if downtime == "" {
		return nil
	}

	startEnds := strings.Split(downtime, ",")
	downtimes := make([]int64, len(startEnds))
	for i, startEnd := range startEnds {
		downtimes[i], _ = strconv.ParseInt(startEnd, 10, 64)
	}
	return downtimes
}
