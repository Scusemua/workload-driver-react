import math
import random
import uuid
from typing import Any, Optional

import numpy as np
import pandas as pd
from scipy.interpolate import interp1d

from training_event import TrainingEvent
from util import get_truncated_normal


class Session(object):
    def __init__(
            self,
            num_events: int,
            event_times: np.ndarray[float],
            inter_arrival_times: np.ndarray[float],
            event_durations: list[float],
            max_millicpus: float = 250,
            max_mem_mb: float = 50,
            num_gpus: int = 1,
            vram_dist_df: Optional[pd.DataFrame] = None
    ):
        self.id = str(uuid.uuid4())
        self.num_training_events: int = num_events
        self.training_events: list[TrainingEvent] = []

        millicpus_rv = get_truncated_normal(mean=max_millicpus / 2, sd=max_millicpus * 0.10, low=1.0e-3,
                                            upp=max_millicpus)
        mem_mb_rv = get_truncated_normal(mean=max_mem_mb / 2, sd=max_mem_mb * 0.10, low=1.0e-3, upp=max_millicpus)
        gpu_util_rv = get_truncated_normal(mean=50, sd=10, low=1.0e-3, upp=100)

        cpu_vals = millicpus_rv.rvs(num_events)
        mem_vals = mem_mb_rv.rvs(num_events)
        gpu_util_vals = gpu_util_rv.rvs(num_events * num_gpus)

        max_vram_usage_gb: float = 0
        if vram_dist_df is not None:
            print(f"Performing inverse transform sampling to generate VRAM values for Session {self.id}.")
            # Inverse transform sampling:
            # https://en.wikipedia.org/wiki/Inverse_transform_sampling

            # Create inverse CDF (interpolation function)
            inverse_cdf = interp1d(vram_dist_df[vram_dist_df.columns[1]], vram_dist_df[vram_dist_df.columns[0]],
                                   bounds_error=False, fill_value="extrapolate")

            # Generate uniform random samples
            u_samples = np.random.uniform(0, 1, num_events)

            # Transform uniform samples to the target distribution
            samples = inverse_cdf(u_samples)

            vram_gb_vals: list[float] = []
            for idx in range(0, num_events):
                vram_util: float = samples[idx]
                vram_gb: float = num_gpus * 4 * vram_util

                # Round to nearest eighth.
                vram_gb = round(vram_gb * 8.0) / 8

                vram_gb_vals.append(vram_gb)

                if vram_gb > max_vram_usage_gb:
                    max_vram_usage_gb = vram_gb
        else:
            # Generate a uniform random utilization value from 0 to NumGpus * 4GB VRAM/ea.
            vram_gb_vals: list[float] = []
            for _ in range(0, len(gpu_util_vals)):
                vram_gb: float = num_gpus * 4 * random.random()
                vram_gb_vals.append(vram_gb)

                if vram_gb > max_vram_usage_gb:
                    max_vram_usage_gb = vram_gb

        print(f"Session {self.id} will use at-most {max_vram_usage_gb} GB of VRAM.")

        def gen_training_event(i) -> TrainingEvent:
            start_time: float = event_times[i]
            duration: float = event_durations[i]

            gpu_utilizations: list[float | dict[str, float]] = gpu_util_vals[i * num_gpus:(i + 1) * num_gpus]
            gpu_utilizations = [{"utilization": round(x, 2)} for x in gpu_utilizations]

            return TrainingEvent(
                start_time,
                duration,
                millicpus=math.floor(cpu_vals[i]),
                mem_mb=round(mem_vals[i], 4),
                gpu_utilizations=gpu_utilizations,
                vram_gb=vram_gb_vals[i])

        first_training_event: TrainingEvent = gen_training_event(0)
        self.training_events.append(first_training_event)

        for idx in range(1, num_events):
            training_event = gen_training_event(idx)
            self.training_events.append(training_event)

            # print(
            #   f"Training event #{i + 1} begins {start_time - (event_times[i - 1] + event_durations[i - 1])} tick(s) after the previous training event's conclusion.")
            assert math.fabs(
                inter_arrival_times[idx - 1] - (
                            event_times[idx] - (event_times[idx - 1] + event_durations[idx - 1]))) < 1.0e-8

        self.event_times: np.ndarray = event_times
        self.inter_arrival_times: np.ndarray = inter_arrival_times
        self.event_durations: list[float] = event_durations
        self.max_millicpus: float = max_millicpus
        self.max_mem_mb: float = max_mem_mb
        self.max_vram_gb: float = max_vram_usage_gb
        self.num_gpus: int = num_gpus

        last_training_event: TrainingEvent = self.training_events[len(self.training_events) - 1]
        self.end_tick: int = last_training_event.ending_tick + 2

        # The session will randomly start sometime before its first training event
        self.start_tick: int = random.randint(1, first_training_event.starting_tick)

    def to_dict(self) -> dict[str, Any]:
        """
        Convert the Session to a dictionary representation that can be output as JSON.
        """
        trainings: list[dict[str, Any]] = []
        for training_event in self.training_events:
            trainings.append(training_event.to_dict())

        return {
            "id": self.id,
            "start_tick": self.start_tick,
            "stop_tick": self.end_tick,
            "num_training_events": self.num_training_events,
            "trainings": trainings
        }
