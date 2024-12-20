import { v4 as uuidv4 } from 'uuid';

// How much to adjust the timescale adjustment factor when using the 'plus' and 'minus' buttons to adjust the field's value.
const TimescaleAdjustmentFactorDelta: number = 0.01;
const TimescaleAdjustmentFactorMax: number = 10;
const TimescaleAdjustmentFactorMin: number = 0;
const TimescaleAdjustmentFactorDefault: number = 0.01;

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
const TrainingVRamUsageGbDefault: number = 0.125;
const NumberOfGpusDefault: number = 1;
const DefaultNumTrainingEvents: number = 1;
const DefaultSelectedTrainingEvent: number = 0;

const WorkloadSampleSessionPercentMin: number = 0.0;
const WorkloadSampleSessionPercentMax: number = 1.0;
const WorkloadSessionSamplePercentDefault: number = 1.0;
const WorkloadSampleSessionPercentDelta: number = 0.01;

const DefaultTrainingEventField = {
    start_tick: TrainingStartTickDefault,
    duration_in_ticks: TrainingDurationInTicksDefault,
    cpus: TrainingCpuUsageDefault,
    memory: TrainingMemUsageGbDefault,
    vram: TrainingVRamUsageGbDefault,
    gpus: NumberOfGpusDefault,
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
        sessionsSamplePercentage: WorkloadSessionSamplePercentDefault,
        timescaleAdjustmentFactor: TimescaleAdjustmentFactorDefault,
        numberOfSessions: 1,
        debugLoggingEnabled: true,
        remoteStorageDefinition: DefaultRemoteStorageDefinition,
        sessions: [GetDefaultSessionFieldValue()],
    };
};

export { TimescaleAdjustmentFactorDelta as TimescaleAdjustmentFactorDelta };
export { TimescaleAdjustmentFactorMax as TimescaleAdjustmentFactorMax };
export { TimescaleAdjustmentFactorMin as TimescaleAdjustmentFactorMin };
export { TimescaleAdjustmentFactorDefault as TimescaleAdjustmentFactorDefault };

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

export { WorkloadSampleSessionPercentDelta as WorkloadSampleSessionPercentDelta };
export { WorkloadSampleSessionPercentMax as WorkloadSampleSessionPercentMax };
export { WorkloadSampleSessionPercentMin as WorkloadSampleSessionPercentMin };
export { WorkloadSessionSamplePercentDefault as WorkloadSessionSamplePercentDefault };

export { DefaultTrainingEventField as DefaultTrainingEventField };
export { GetDefaultSessionFieldValue as GetDefaultSessionFieldValue };

export { GetDefaultFormValues as GetDefaultFormValues };
