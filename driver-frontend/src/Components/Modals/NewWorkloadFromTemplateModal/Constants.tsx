import { v4 as uuidv4 } from 'uuid';

// How much to adjust the timescale adjustment factor when using the 'plus' and 'minus' buttons to adjust the field's value.
const TimescaleAdjustmentFactorDelta: number = 0.1;
const TimescaleAdjustmentFactorMax: number = 10;
const TimescaleAdjustmentFactorMin: number = 1.0e-3;
const TimeAdjustmentFactorDefault: number = 0.1;

// The number of Sessions in the workload.
const NumberOfSessionsDefault: number = 1;
const NumberOfSessionsMin: number = 0;
const NumberOfSessionsMax: number = Number.MAX_SAFE_INTEGER;
const NumberOfSessionsDelta: number = 1;

// How much to adjust the workload seed when using the 'plus' and 'minus' buttons to adjust the field's value.
const WorkloadSeedDelta: number = 1.0;
const WorkloadSeedMax: number = 2147483647.0;
const WorkloadSeedMin: number = 0.0;
const WorkloadSeedDefault: number = 0.0;

const SessionStartTickDefault: number = 1;
const SessionStopTickDefault: number = 6;
const TrainingStartTickDefault: number = 2;
const TrainingDurationInTicksDefault: number = 2;
const TrainingCpuUsageDefault: number = 100; // in millicpus
const TrainingGpuPercentUtilDefault: number = 50;
const TrainingMemUsageGbDefault: number = 0.25;
const TrainingVRamUsageGbDefault: number = 0.128;
const NumberOfGpusDefault: number = 1;
const DefaultNumTrainingEvents: number = 1;
const DefaultSelectedTrainingEvent: number = 0;

const DefaultTrainingEventField = {
    start_tick: TrainingStartTickDefault,
    duration_in_ticks: TrainingDurationInTicksDefault,
    millicpus: TrainingCpuUsageDefault,
    mem_usage_mb: TrainingMemUsageGbDefault,
    vram_usage_gb: TrainingVRamUsageGbDefault,
    num_gpus: NumberOfGpusDefault,
    gpu_utilizations: [
        {
            utilization: TrainingGpuPercentUtilDefault,
        },
    ],
};

const GetDefaultSessionFieldValue = () => {
    return {
        id: uuidv4(),
        start_tick: SessionStartTickDefault,
        stop_tick: SessionStopTickDefault,
        num_training_events: DefaultNumTrainingEvents,
        selected_training_event: DefaultSelectedTrainingEvent,
        trainings: [DefaultTrainingEventField],
    };
};

const DefaultRemoteStorageDefinition = {
    name: 'AWS S3',
    downloadRate: 200e6,
    uploadRate: 125e6,
    downloadRateVariancePercentage: 5,
    uploadRateVariancePercentage: 5,
    readFailureChancePercentage: 0.0,
    writeFailureChancePercentage: 0.0,
};

const GetDefaultFormValues = () => {
    const title: string = uuidv4();

    return {
        workloadTitle: title,
        workloadSeed: WorkloadSeedDefault,
        timescaleAdjustmentFactor: TimeAdjustmentFactorDefault,
        numberOfSessions: 1,
        debugLoggingEnabled: true,
        remoteStorageDefinition: DefaultRemoteStorageDefinition,
        sessions: [GetDefaultSessionFieldValue()],
    };
};

function RoundToTwoDecimalPlaces(num: number) {
    return +(Math.round(Number.parseFloat(num.toString() + 'e+2')).toString() + 'e-2');
}

function RoundToThreeDecimalPlaces(num: number) {
    return +(Math.round(Number.parseFloat(num.toString() + 'e+3')).toString() + 'e-3');
}

function RoundToNDecimalPlaces(num: number, n: number) {
    return +(Math.round(Number.parseFloat(num.toString() + `e+${n}`)).toString() + `e-${n}`);
}

export { TimescaleAdjustmentFactorDelta as TimescaleAdjustmentFactorDelta };
export { TimescaleAdjustmentFactorMax as TimescaleAdjustmentFactorMax };
export { TimescaleAdjustmentFactorMin as TimescaleAdjustmentFactorMin };
export { TimeAdjustmentFactorDefault as TimeAdjustmentFactorDefault };

export { WorkloadSeedDelta as WorkloadSeedDelta };
export { WorkloadSeedMax as WorkloadSeedMax };
export { WorkloadSeedMin as WorkloadSeedMin };
export { WorkloadSeedDefault as WorkloadSeedDefault };

export { NumberOfSessionsDefault as NumberOfSessionsDefault };
export { NumberOfSessionsMin as NumberOfSessionsMin };
export { NumberOfSessionsMax as NumberOfSessionsMax };
export { NumberOfSessionsDelta as NumberOfSessionsDelta };

export { SessionStartTickDefault as SessionStartTickDefault };
export { SessionStopTickDefault as SessionStopTickDefault };
export { TrainingStartTickDefault as TrainingStartTickDefault };
export { TrainingDurationInTicksDefault as TrainingDurationInTicksDefault };
export { TrainingCpuUsageDefault as TrainingCpuUsageDefault };
export { TrainingGpuPercentUtilDefault as TrainingGpuPercentUtilDefault };
export { TrainingVRamUsageGbDefault as TrainingVRamUsageGbDefault };
export { TrainingMemUsageGbDefault as TrainingMemUsageGbDefault };
export { NumberOfGpusDefault as NumberOfGpusDefault };
export { DefaultNumTrainingEvents as DefaultNumTrainingEvents };
export { DefaultSelectedTrainingEvent as DefaultSelectedTrainingEvent };

export { DefaultTrainingEventField as DefaultTrainingEventField };
export { GetDefaultSessionFieldValue as GetDefaultSessionFieldValue };

export { GetDefaultFormValues as GetDefaultFormValues };

export { RoundToTwoDecimalPlaces as RoundToTwoDecimalPlaces };
export { RoundToThreeDecimalPlaces as RoundToThreeDecimalPlaces };
export { RoundToNDecimalPlaces as RoundToNDecimalPlaces };
