import { Label } from '@patternfly/react-core';
import {
    CheckCircleIcon,
    ExclamationTriangleIcon,
    HourglassStartIcon,
    PausedIcon,
    PendingIcon,
    QuestionIcon,
    SpinnerIcon,
    TimesCircleIcon,
} from '@patternfly/react-icons';
import text from '@patternfly/react-styles/css/utilities/Text/text';
import React from 'react';

export const WorkloadStateReady: string = 'WorkloadReady'; // Workload is registered and ready to be started.
export const WorkloadStateRunning: string = 'WorkloadRunning'; // Workload is actively running/in-progress.
export const WorkloadStatePausing: string = 'WorkloadPausing'; // Workload is finishing processing the current tick and then will pause.
export const WorkloadStatePaused: string = 'WorkloadPaused'; // Workload is paused.
export const WorkloadStateFinished: string = 'WorkloadFinished'; // Workload stopped naturally/successfully after processing all events.
export const WorkloadStateErred: string = 'WorkloadErred'; // Workload stopped due to an error.
export const WorkloadStateTerminated: string = 'WorkloadTerminated'; // Workload stopped because it was explicitly terminated early/premature.

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
    return (
        workload.workload_state == WorkloadStateFinished ||
        workload.workload_state == WorkloadStateErred ||
        workload.workload_state == WorkloadStateTerminated
    );
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
    tick_durations_milliseconds: number[];
    sum_tick_durations_millis: number;
}

export function IsPaused(workload: Workload) {
    return workload.workload_state == WorkloadStatePaused;
}

export function IsPausing(workload: Workload) {
    return workload.workload_state == WorkloadStatePausing;
}

export function IsActivelyRunning(workload: Workload) {
    return workload.workload_state == WorkloadStateRunning;
}

export function IsTerminated(workload: Workload) {
    return workload.workload_state == WorkloadStateTerminated;
}

/**
 * Alias for IsFinished.
 *
 * Returns true if the workload finished successfully.
 */
export function IsComplete(workload: Workload) {
    return workload.workload_state == WorkloadStateFinished;
}

/**
 * Alias for IsComplete.
 *
 * Returns true if the workload finished successfully.
 */
export function IsFinished(workload: Workload) {
    return workload.workload_state == WorkloadStateFinished;
}

export function IsReadyAndWaiting(workload: Workload) {
    return workload.workload_state == WorkloadStateReady;
}

export function IsErred(workload: Workload) {
    return workload.workload_state == WorkloadStateErred;
}

export function IsInProgress(workload: Workload) {
    return IsPaused(workload) || IsPausing(workload) || IsActivelyRunning(workload);
}

/**
 * GetNumActiveSessionsInWorkload returns the current number of actively-running (i.e., idle or executing code)
 * sessions from the given workload.
 */
export function GetNumActiveSessionsInWorkload(workload: Workload): number {
    let num_active_sessions: number = 0;

    workload.sessions.forEach(function (session: Session) {
        if (session.state == 'idle' || session.state == 'training') {
            num_active_sessions += 1;
        }
    });

    return num_active_sessions;
}

export const GetWorkloadStatusLabel = (workload: Workload) => {
    if (IsReadyAndWaiting(workload)) {
        return (
            <Label icon={<HourglassStartIcon className={text.infoColor_100} />} color="blue">
                Ready
            </Label>
        );
    }

    if (IsActivelyRunning(workload)) {
        return (
            <Label icon={<SpinnerIcon className={'loading-icon-spin ' + text.successColor_100} />} color="green">
                Running
            </Label>
        );
    }

    if (IsPausing(workload)) {
        return (
            <Label icon={<PendingIcon />} color="cyan">
                Pausing
            </Label>
        );
    }

    if (IsPaused(workload)) {
        return (
            <Label icon={<PausedIcon />} color="cyan">
                Paused
            </Label>
        );
    }

    if (IsFinished(workload)) {
        return (
            <Label icon={<CheckCircleIcon className={text.successColor_100} />} color="green">
                Complete
            </Label>
        );
    }

    if (IsErred(workload)) {
        return (
            <Label icon={<TimesCircleIcon className={text.dangerColor_100} />} color="red">
                Erred
            </Label>
        );
    }

    if (IsTerminated(workload)) {
        return (
            <Label icon={<ExclamationTriangleIcon className={text.warningColor_100} />} color="orange">
                Terminated
            </Label>
        );
    }

    return (
        <Label icon={<QuestionIcon className={text.warningColor_100} />} color="orange">
            Unknown
        </Label>
    );
};

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
    cpus: number; // millicpus (1/1000 CPU cores)
    gpus: number;
    vram: number; // GPU memory in gigabytes (GB)
    memory_mb: number; // megabytes (MB)
    gpu_type: string;
}

interface RemoteStorageDefinition {
    name: string;
    downloadRate: number;
    uploadRate: number;
    downloadVariancePercent: number;
    uploadVariancePercent: number;
    readFailureChancePercentage: number;
    writeFailureChancePercentage: number;
}

interface Session {
    form_id: string;
    id: string;
    max_resource_request: ResourceRequest;
    current_resource_request: ResourceRequest;
    start_tick: number;
    stop_tick: number;
    trainings: TrainingEvent[];
    trainings_completed: number;
    state: string;
    error_message: string; // If the session encountered an error message, then we can store it here.
    stderr_io_pub_messages: string[];
    stdout_io_pub_messages: string[];
}

interface TrainingEvent {
    training_index: number;
    millicpus: number;
    mem_usage_mb: number;
    vram_usage_gb: number;
    gpu_utilizations: GpuUtilization[];
    start_tick: number;
    duration_in_ticks: number;
}

interface GpuUtilization {
    utilization: number;
}

interface PatchedWorkload {
    workloadId: string;
    patch: string;
}

interface BaseWorkloadResponse {
    msg_id: string;
    op: string;
    status: string;
}

interface ErrorResponse {
    Description: string;
    ErrorMessage: string;
    Valid: boolean;
    op: string;
    status: string;
    msg_id: string;
}

// Response for a 'get workloads' request.
// Sent to the front-end by the back-end.
interface WorkloadResponse {
    msg_id: string;
    op: string;
    status: string;
    new_workloads: Workload[];
    modified_workloads: Workload[];
    deleted_workloads: Workload[];
    patched_workloads: PatchedWorkload[];
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
        case WorkloadStateReady:
            return 'The workload has been registered and is ready to begin.';
        case WorkloadStateRunning:
            return 'The workload is actively-running.';
        case WorkloadStateFinished:
            return 'The workload has completed successfully.';
        case WorkloadStateErred:
            return 'The workload has been aborted due to a critical error: ' + workload.error_message;
        case WorkloadStateTerminated:
            return 'The workload has been explicitly/manually terminated.';
    }

    console.error(
        `Workload ${workload.name} (ID=${workload.id}) is in an unsupported/unknown state: "${workload.workload_state}"`,
    );
    return `The workload is currently in an unknown/unsupported state: "${workload.workload_state}"`;
}

export { IsWorkloadFinished as IsWorkloadFinished };

export { GetWorkloadStatusTooltip as GetWorkloadStatusTooltip };

export type { Workload as Workload };
export type { WorkloadPreset as WorkloadPreset };
export type { BaseWorkloadResponse as BaseWorkloadResponse };
export type { WorkloadResponse as WorkloadResponse };
export type { WorkloadEvent as WorkloadEvent };
export type { Session as Session };
export type { TrainingEvent as TrainingEvent };
export type { WorkloadTemplate as WorkloadTemplate };
export type { ResourceRequest as ResourceRequest };
export type { PatchedWorkload as PatchedWorkload };
export type { ErrorResponse as ErrorResponse };
export type { RemoteStorageDefinition as RemoteStorageDefinition };
