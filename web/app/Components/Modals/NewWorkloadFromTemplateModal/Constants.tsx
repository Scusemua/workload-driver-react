import { v4 as uuidv4 } from 'uuid';

// How much to adjust the timescale adjustment factor when using the 'plus' and 'minus' buttons to adjust the field's value.
const TimescaleAdjustmentFactorDelta: number = 0.1;
const TimescaleAdjustmentFactorMax: number = 10;
const TimescaleAdjustmentFactorMin: number = 1.0e-3;
const TimeAdjustmentFactorDefault: number = 0.1;

// How much to adjust the workload seed when using the 'plus' and 'minus' buttons to adjust the field's value.
const WorkloadSeedDelta: number = 1.0;
const WorkloadSeedMax: number = 2147483647.0;
const WorkloadSeedMin: number = 0.0;
const WorkloadSeedDefault: number = 0.0;

const SessionStartTickDefault: number = 1;
const SessionStopTickDefault: number = 6;
const TrainingStartTickDefault: number = 2;
const TrainingDurationInTicksDefault: number = 2;
const TrainingCpuPercentUtilDefault: number = 10;
const TrainingGpuPercentUtilDefault: number = 50;
const TrainingMemUsageGbDefault: number = 0.25;
const NumberOfGpusDefault: number = 1;
const DefaultNumTrainingEvents: number = 1;
const DefaultSelectedTrainingEvent: number = 0;

const DefaultTrainingEventField = {
    training_start_tick: TrainingStartTickDefault,
    training_duration_ticks: TrainingDurationInTicksDefault,
    cpu_percent_util: TrainingCpuPercentUtilDefault,
    mem_usage_gb_util: TrainingMemUsageGbDefault,
    num_gpus: NumberOfGpusDefault,
    gpu_utilizations: [{
        utilization: TrainingGpuPercentUtilDefault
    }]
}

const DefaultSessionFieldValue = {
    id: uuidv4(),
    start_tick: SessionStartTickDefault,
    stop_tick: SessionStopTickDefault,
    num_training_events: DefaultNumTrainingEvents,
    selected_training_event: DefaultSelectedTrainingEvent,
    training_events: [{
        ...DefaultTrainingEventField,
    }],
}

export {TimescaleAdjustmentFactorDelta as TimescaleAdjustmentFactorDelta};
export {TimescaleAdjustmentFactorMax as TimescaleAdjustmentFactorMax};
export {TimescaleAdjustmentFactorMin as TimescaleAdjustmentFactorMin};
export {TimeAdjustmentFactorDefault as TimeAdjustmentFactorDefault};

export {WorkloadSeedDelta as WorkloadSeedDelta};
export {WorkloadSeedMax as WorkloadSeedMax};
export {WorkloadSeedMin as WorkloadSeedMin};
export {WorkloadSeedDefault as WorkloadSeedDefault};

export {SessionStartTickDefault as SessionStartTickDefault};
export {SessionStopTickDefault as SessionStopTickDefault};
export {TrainingStartTickDefault as TrainingStartTickDefault};
export {TrainingDurationInTicksDefault as TrainingDurationInTicksDefault};
export {TrainingCpuPercentUtilDefault as TrainingCpuPercentUtilDefault};
export {TrainingGpuPercentUtilDefault as TrainingGpuPercentUtilDefault};
export {TrainingMemUsageGbDefault as TrainingMemUsageGbDefault};
export {NumberOfGpusDefault as NumberOfGpusDefault};
export {DefaultNumTrainingEvents as DefaultNumTrainingEvents};
export {DefaultSelectedTrainingEvent as DefaultSelectedTrainingEvent};

export {DefaultTrainingEventField as DefaultTrainingEventField};
export {DefaultSessionFieldValue as DefaultSessionFieldValue};