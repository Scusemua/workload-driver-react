interface WorkloadPreset {
    name: string; // Human-readable name for this particular workload preset.
    description: string; // Human-readable description of the workload.
    key: string; // Key for code-use only (i.e., we don't intend to display this to the user for the most part).
    months: string[]; // The months of data used by the workload.
}

interface Workload {
    id: string;
    name: string;
    started: boolean;
    workload_preset_name: string;
    workload_preset_key: string;
    start_time: string;
    time_elapsed: string;
    num_tasks_executed: number;
    finished: boolean;
}

export type { Workload as Workload };
export type { WorkloadPreset as WorkloadPreset };
