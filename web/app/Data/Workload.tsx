const WORKLOAD_STATE_READY: string = "WorkloadReady"   ; // Workload is registered and ready to be started.
const WORKLOAD_STATE_RUNNING: string = "WorkloadRunning"; // Workload is actively running/in-progress.
const WORKLOAD_STATE_FINISHED: string = "WorkloadFinished"; // Workload stopped naturally/successfully after processing all events.
const WORKLOAD_STATE_ERRED: string = "WorkloadErred"; // Workload stopped due to an error.
const WORKLOAD_STATE_TERMINATED: string = "WorkloadTerminated"; // Workload stopped because it was explicitly terminated early/premature.

interface WorkloadPreset {
    name: string; // Human-readable name for this particular workload preset.
    description: string; // Human-readable description of the workload.
    key: string; // Key for code-use only (i.e., we don't intend to display this to the user for the most part).
    months: string[]; // The months of data used by the workload.
    months_description: string; // Formatted, human-readable text of the form (StartMonth) - (EndMonth) or (Month) if there is only one month included in the trace.
}

interface WorkloadPreset {
    name: string; // Human-readable name for this particular workload preset.
    description: string; // Human-readable description of the workload.
    key: string; // Key for code-use only (i.e., we don't intend to display this to the user for the most part).
    preset_type: string; // The type of preset ("XML" or "CSV").
    months: string[]; // The months of data used by the workload.
    months_description: string; // Formatted, human-readable text of the form (StartMonth) - (EndMonth) or (Month) if there is only one month included in the trace.
    svg_content: string[]; // For XML presets, their events can be rendered/displayed as an SVG.
}

// Return true if the workload is in the 'finished', 'erred', or 'terminated' states.
function IsWorkloadFinished(workload: Workload) {
    return (workload.workload_state == WORKLOAD_STATE_FINISHED || workload.workload_state == WORKLOAD_STATE_ERRED || workload.workload_state == WORKLOAD_STATE_TERMINATED)
}

interface Workload {
    id: string;
    name: string;
    workload_state: string;
    workload_preset: WorkloadPreset;
    workload_preset_name: string;
    workload_preset_key: string;
    workload_template: WorkloadTemplate;
    registered_time: string; // Timestamp of when the workload was registered.
    start_time: string;
    time_elapsed: number;
    time_elapsed_str: string;
    num_tasks_executed: number;
    seed: number;
    num_active_sessions: number;
    num_sessions_created: number;
    num_events_processed: number;
    num_active_trainings: number;
    debug_logging_enabled: boolean;
    error_message: string;
    timescale_adjustment_factor: number;
    events_processed: WorkloadEvent[];
    sessions: Session[];
    simulation_clock_time: string; 
    current_tick: number;
}

interface WorkloadEvent {
    idx: number; 
    id: string;
    name: string;
    session: string;
    timestamp: string;
    processed_at: string;
    processed_successfully: boolean;
    error_message: string;
}

interface ResourceRequest {
    cpus: number;
    gpus: number;
    mem_gb: number;
    gpu_type: string;
}

interface Session {
    form_id: string;
    id: string;
    resource_request: ResourceRequest;
    start_tick: number;
    stop_tick: number;
    trainings: TrainingEvent[];
    trainings_completed: number;
    state: string; 
    error_message: string; // If the session encountered an error message, then we can store it here.
}

interface TrainingEvent {
    sessionId: string;
    trainingId: string;
    cpuUtil: number;
    memUsageGb: number;
    gpuUtil: number[];
    startTick: number;
    durationInTicks: number;
}

// Response for a 'get workloads' request.
// Sent to the front-end by the back-end.
interface WorkloadResponse {
    msg_id: string;
    new_workloads: Workload[];
    modified_workloads: Workload[];
    deleted_workloads: Workload[];
}

// Wraps a workload created using a template.
interface WorkloadTemplate {
    // name: string;
    sessions: Session[];
}

function GetWorkloadStatusTooltip(workload: Workload | null) {
    if (workload === null) {
        return 'N/A';
    }

    switch (workload.workload_state) {
        case WORKLOAD_STATE_READY:
            return 'The workload has been registered and is ready to begin.';
        case WORKLOAD_STATE_RUNNING:
            return 'The workload is actively-running.';
        case WORKLOAD_STATE_FINISHED:
            return 'The workload has completed successfully.';
        case WORKLOAD_STATE_ERRED:
            return 'The workload has been aborted due to a critical error: ' + workload.error_message;
        case WORKLOAD_STATE_TERMINATED:
            return 'The workload has been explicitly/manually terminated.';
    }

    console.error(
        `Workload ${workload.name} (ID=${workload.id}) is in an unsupported/unknown state: ${workload.workload_state}`,
    );
    return 'The workload is currently in an unknown/unsupported state.';
};

export { WORKLOAD_STATE_READY as WORKLOAD_STATE_READY };
export { WORKLOAD_STATE_RUNNING as WORKLOAD_STATE_RUNNING };
export { WORKLOAD_STATE_FINISHED as WORKLOAD_STATE_FINISHED };
export { WORKLOAD_STATE_ERRED as WORKLOAD_STATE_ERRED };
export { WORKLOAD_STATE_TERMINATED as WORKLOAD_STATE_TERMINATED };

export { IsWorkloadFinished as IsWorkloadFinished };

export { GetWorkloadStatusTooltip as GetWorkloadStatusTooltip };

export type { Workload as Workload };
export type { WorkloadPreset as WorkloadPreset };
export type { WorkloadResponse as WorkloadResponse };
export type { WorkloadEvent as WorkloadEvent };
export type { Session as Session };
export type { TrainingEvent as TrainingEvent };
export type { WorkloadTemplate as WorkloadTemplate };
export type { ResourceRequest as ResourceRequest };