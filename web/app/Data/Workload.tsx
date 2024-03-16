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
    started: boolean;
    workload_preset: WorkloadPreset;
    workload_preset_name: string;
    workload_preset_key: string;
    start_time: string;
    time_elapsed: string;
    num_tasks_executed: number;
    finished: boolean;
    seed: number;
}

export type { Workload as Workload };
export type { WorkloadPreset as WorkloadPreset };
