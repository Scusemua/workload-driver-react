interface WorkloadPreset {
    name: string; // Human-readable name for this particular workload preset.
    description: string; // Human-readable description of the workload.
    key: string; // Key for code-use only (i.e., we don't intend to display this to the user for the most part).
    months: string[]; // The months of data used by the workload.
    months_description: string; // Formatted, human-readable text of the form (StartMonth) - (EndMonth) or (Month) if there is only one month included in the trace.
}

interface Workload {
    id: string;
    name: string;
    workload_state: number;
    workload_preset: WorkloadPreset;
    workload_preset_name: string;
    workload_preset_key: string;
    start_time: string;
    time_elapsed: string;
    num_tasks_executed: number;
    seed: number;
    num_active_sessions: number;
    num_sessions_created: number;
    num_events_processed: number;
    num_active_trainings: number;
    debug_logging_enabled: boolean;
}

const WORKLOAD_STATE_READY: number = 0; // Workload is registered and ready to be started.
const WORKLOAD_STATE_RUNNING: number = 1; // Workload is actively running/in-progress.
const WORKLOAD_STATE_FINISHED: number = 2; // Workload stopped naturally/successfully after processing all events.
const WORKLOAD_STATE_ERRED: number = 3; // Workload stopped due to an error.
const WORKLOAD_STATE_TERMINATED: number = 4; // Workload stopped because it was explicitly terminated early/premature.

export { WORKLOAD_STATE_READY as WORKLOAD_STATE_READY };
export { WORKLOAD_STATE_RUNNING as WORKLOAD_STATE_RUNNING };
export { WORKLOAD_STATE_FINISHED as WORKLOAD_STATE_FINISHED };
export { WORKLOAD_STATE_ERRED as WORKLOAD_STATE_ERRED };
export { WORKLOAD_STATE_TERMINATED as WORKLOAD_STATE_TERMINATED };

export type { Workload as Workload };
export type { WorkloadPreset as WorkloadPreset };
