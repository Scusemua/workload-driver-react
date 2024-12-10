package domain

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	configKit "github.com/gookit/config/v2"
	"github.com/gookit/config/v2/yaml"
	"github.com/imdario/mergo"
	"github.com/mitchellh/mapstructure"
	"k8s.io/client-go/util/homedir"
)

const (
	OptionName = "name"
	//OptionDefault = "default"
	OptionDesc = "description"
)

var (
	Config  *Configuration = nil
	Verbose bool
	Months  = []string{"jan", "feb", "mar", "apr", "may", "jun", "jul", "aug", "sep", "oct", "nov", "dec"}
)

// Configuration encapsulates the configuration for the backend.
// These are all parsed and converted into flag arguments using the
// provided 'flag' package (i.e., the one that's part of the standard library).
type Configuration struct {
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
	LastTimestamp                int64  `name:"last-timestamp" description:"Epoch Unix Timestamp denoting the last timestamp for which events will be generated. Any events beyond that point will be discarded."`
	EvictHostOnLastContainerStop int    `name:"evict-host-on-last-container-stop" description:"Override the default settings for whatever Scheduler you're using and force a value for this parameter. -1 to force false, 0 to leave as default for the particular scheduler, and 1 to force true."`
	WorkloadPresetsFilepath      string `name:"workload-presets-file" description:"Path to a .YAML file containing the definitions of one or more Workload Presets."`
	WorkloadTemplatesFilepath    string `name:"workload-templates-file" yaml:"workload-templates-file" json:"workload-templates-file"`
	ExpectedOriginPort           int    `name:"expected-origin-port" description:"Port of the expected origin for messages from the frontend."`
	ExpectedOriginAddresses      string `name:"expected_websocket_origins" json:"expected_websocket_origins" yaml:"expected_websocket_origins" description:"Comma-separated list of addresses (without ports) passed as a single string. These are acceptable/expected origins for the websocket connection upgrader to allow."`
	ClusterDashboardHandlerPort  int    `name:"cluster-dashboard-handler-port" description:"Port for the Cluster Dashboard handler gRPC server to listen on."`

	DriverTimescale float64 `name:"driver-timescale" description:"Multiplier that impacts the timescale at the Driver will operate on with respect to the trace data. For example, if each tick is 60 seconds, then a DriverTimescale value of 0.5 will mean that each tick will take 30 seconds."`

	///////////////////////
	// General execution //
	///////////////////////
	// Options are 0 (i.e., 'pre') and 1 (i.e., 'standard'). With 'pre', the Simulator will not process any events; it will simply parse
	// the trace to extract the CPU, GPU, and Memory readings. With 'standard', the Simulator will actually simulate the workload.
	// ExecutionMode                     int  `name:"execution-mode" description:"Options are 0 (i.e., 'pre') and 1 (i.e., 'standard'). With 'pre', the Simulator will not process any events; it will simply parse the trace to extract the CPU, GPU, and Memory readings. With 'standard', the Simulator will actually simulate the workload."`
	MaxTaskDurationSec                int  `name:"max-task-duration-seconds" description:"The maximum length of a task. If a task with length >= this value is executed, then the associated Session will be terminated once the event completes."`
	ContinueUntilExplicitlyTerminated bool `name:"continue-until-explicit-termination" description:""`

	//////////////////////
	// Resource Credits //
	//////////////////////
	UseResourceCredits      bool    `name:"use-resource-credits" description:"If true, then use resource credits as the billing model, rather than raw USD."`
	InitialCreditBalance    float64 `name:"initial-resource-credit-balance" description:"The number of resource credits a new user will initially have available."`
	ResourceCreditCPU       float64 `name:"resource-credit-vcpus" description:"The amount of vCPUs made available to a user for one hour by a single resource credit."`
	ResourceCreditGPU       float64 `name:"resource-credit-gpus" description:"The amount of GPUs made available to a user for one hour by a single resource credit."`
	ResourceCreditMemMB     float64 `name:"resource-credit-mem-mb" description:"The amount of memory (i.e., RAM), in megabytes (MB), made available to a user for one hour by a single resource credit."`
	ResourceCreditCostInUSD float64 `name:"resource-credit-cost-usd" description:"The cost-equivalent of a single resource credit in USD."`

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

	PushUpdateInterval           int    `name:"push-update-interval" description:"How frequently the server should push updates about active workloads to the frontend"`
	WorkloadDriverKernelSpec     string `name:"workload-driver-kernel-spec-name" description:"The name of the Jupyter kernel spec to be used by the workload driver when creating new kernels."`
	ConnectToKernelTimeoutMillis int64  `name:"kernel-connection-timeout-milliseconds" description:"The amount of time, in milliseconds, to wait while establishing a connection to a new kernel (from the workload driver) before returning with an error. Defaults to 60,000 milliseconds (i.e., 60 seconds, or 1 minute)."` // The amount of time, in milliseconds, to wait while establishing a connection to a new kernel (from the workload driver) before returning with an error. Defaults to 60,000 milliseconds (i.e., 60 seconds, or 1 minute).

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

	OnlyChargeMixedContainersWhileTraining bool   `name:"only-charge-mixed-containers-while-training" description:"When the training-mode is set to \"local\", only charge the user (using the configured pricing model, such as GPU-Proportional) when the notebook server is actively training. When the notebook server is not actively training, fall back to CPU/Memory-proportional."`
	LogOutputFile                          string `name:"log-output-file" description:"If specified, then log output will go to this file, rather than to STDOUT."` // If specified, then log output will go to this file, rather than to STDOUT.

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

	// SpoofKubeNodes          bool   `name:"spoof-nodes" yaml:"spoof-nodes" json:"spoof-nodes" description:"If true, spoof the Kubernetes nodes."`
	// SpoofKernels            bool   `name:"spoof-kernels" yaml:"spoof-kernels" json:"spoof-kernels" description:"If true, spoof the kernels."`
	// SpoofKernelSpecs        bool   `name:"spoof-specs" yaml:"spoof-specs" json:"spoof-specs" description:"If true, spoof the kernel specs."`
	InCluster                     bool   `name:"in-cluster" yaml:"in-cluster" json:"in-cluster" description:"Should be true if running from within the kubernetes cluster."`
	KernelQueryInterval           string `name:"kernel-query-interval" yaml:"kernel-query-interval" json:"kernel-query-interval" default:"5s" description:"How frequently to query the Cluster for updated kernel information."`
	NodeQueryInterval             string `name:"node-query-interval" yaml:"node-query-interval" json:"node-query-interval" default:"10s" description:"How frequently to query the Cluster for updated Kubernetes node information."`
	KernelSpecQueryInterval       string `name:"kernel-spec-query-interval" yaml:"kernel-spec-query-interval" json:"kernel-spec-query-interval" default:"600s" description:"How frequently to query the Cluster for updated Jupyter kernel spec information."`
	KubeConfig                    string `name:"kubeconfig" yaml:"kubeconfig" json:"kubeconfig" description:"Absolute path to the kubeconfig file."`
	GatewayAddress                string `name:"gateway-address" yaml:"gateway-address" json:"gateway-address" description:"The IP address that the front-end should use to connect to the Gateway."`
	FrontendJupyterServerAddress  string `name:"frontend-jupyter-server-address" yaml:"frontend-jupyter-server-address" json:"frontend-jupyter-server-address" description:"The IP address of the Jupyter Server to return to the frontend."`
	InternalJupyterServerAddress  string `name:"internal-jupyter-server-address" yaml:"internal-jupyter-server-address" json:"internal-jupyter-server-address" description:"The IP address of the Jupyter Server to use internally within the backend."`
	JupyterServerBasePath         string `name:"jupyter-server-base-path" json:"jupyter-server-base-path" yaml:"jupyter-server-base-path" description:"The base path on which the Jupyter Server is listening."`
	ServerPort                    int    `name:"server-port" yaml:"server-port" json:"server-port" description:"Port of the backend server."`
	WebsocketProxyPort            int    `name:"websocket-proxy-port" yaml:"websocket-proxy-port" json:"websocket-proxy-port" description:"Port of the backend websocket proxy server, which reverse-proxies websocket connections to the Jupyter server."`
	AdminUser                     string `name:"admin_username" yaml:"admin_username" json:"admin_username"`
	AdminPassword                 string `name:"admin_password" yaml:"admin_password" json:"admin_password"`
	TokenValidDurationSec         int    `name:"token_valid_duration_sec" yaml:"token_valid_duration_sec" json:"token_valid_duration_sec"`
	TokenRefreshIntervalSec       int    `name:"token_refresh_interval_sec" yaml:"token_refresh_interval_sec" json:"token_refresh_interval_sec"`
	BaseUrl                       string `name:"base-url" yaml:"base-url" json:"base-url" default:"/"`
	PrometheusEndpoint            string `name:"prometheus-endpoint" yaml:"prometheus-endpoint" json:"prometheus-endpoint" default:"/metrics"`
	WorkloadOutputDirectory       string `name:"workload_output_directory" json:"workload_output_directory" yaml:"workload_output_directory" default:"./workload_output_directory"`
	WorkloadOutputIntervalSec     int    `name:"workload-output-interval-seconds" json:"workload-output-interval-seconds" yaml:"workload-output-interval-seconds"`
	TimeCompressTrainingDurations bool   `name:"apply-time-compression-to-training-durations" json:"apply-time-compression-to-training-durations" yaml:"apply-time-compression-to-training-durations"`
}

func GetDefaultConfig() *Configuration {
	var kubeconfigDefaultValue string
	if home := homedir.HomeDir(); home != "" {
		kubeconfigDefaultValue = filepath.Join(home, ".kube", "config")
	} else {
		kubeconfigDefaultValue = ""
	}

	return &Configuration{
		InCluster:                     false,
		KernelQueryInterval:           "60s",
		NodeQueryInterval:             "120s",
		KubeConfig:                    kubeconfigDefaultValue,
		GatewayAddress:                "localhost:8079",
		KernelSpecQueryInterval:       "600s",
		FrontendJupyterServerAddress:  "localhost:8888",
		InternalJupyterServerAddress:  "localhost:8888",
		JupyterServerBasePath:         "/",
		ServerPort:                    8000,
		WorkloadDriverKernelSpec:      "distributed",
		PushUpdateInterval:            1,
		ConnectToKernelTimeoutMillis:  60000,
		WebsocketProxyPort:            8001,
		ClusterDashboardHandlerPort:   8078,
		ExpectedOriginPort:            9001,
		ExpectedOriginAddresses:       "localhost,127.0.0.1",
		TraceStep:                     60,
		WorkloadOutputDirectory:       "./workload_output_directory",
		WorkloadOutputIntervalSec:     2,
		TimeCompressTrainingDurations: true,
	}
}

func (opts *Configuration) String() string {
	out, err := json.MarshalIndent(opts, "", "  ")
	if err != nil {
		panic(err)
	}

	return string(out)
}

func (opts *Configuration) CheckUsage() {
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
		_, _ = fmt.Fprintf(os.Stderr, "Usage: ./play [options] data_base_path\n")
		_, _ = fmt.Fprintf(os.Stderr, "Available options:\n")
		flag.PrintDefaults()
		os.Exit(0)
	}

	if opts.YAML != "" {
		fmt.Printf("Reading configuration from file: \"%s\"\n", opts.YAML)
		configKit.WithOptions(func(opt *configKit.Options) {
			opt.SetTagName(OptionName)
			// DecoderConfig initialization is due a bug in configKit: no TagName will be applied if DecoderConfig is nil.
			// TODO: Fix the bug
			opt.DecoderConfig = &mapstructure.DecoderConfig{}
		})
		configKit.AddDriver(yaml.Driver)
		if err := configKit.LoadFiles(opts.YAML); err != nil {
			panic(err)
		}
		fileOpts := &Configuration{}
		if err := configKit.BindStruct("", fileOpts); err != nil {
			panic(err)
		}

		if err := mergo.Merge(opts, fileOpts, mergo.WithOverride); err != nil {
			panic(err)
		}
	} else {
		fmt.Printf("[WARNING] No YAML configuration file specified...\n")
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

	fmt.Printf("Server configuration:\n%v\n", opts)
}
