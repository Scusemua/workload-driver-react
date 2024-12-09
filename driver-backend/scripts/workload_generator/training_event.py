import math

from typing import Any

class TrainingEvent(object):
    def __init__(
            self,
            start_time: float,
            duration: float,
            millicpus: float = 100,
            mem_mb: float = 5,
            gpu_utilizations=None,
            vram_gb: float = 1.0,
    ):
        """
        :param start_time: the time at which the event begins (in ticks).
        :param duration: the duration of the event (in ticks).
        """
        if gpu_utilizations is None:
            gpu_utilizations = [{"utilization": 50.0}]
        self.starting_tick: int = int(math.ceil(start_time))
        self.duration: int = int(math.ceil(duration))
        self.ending_tick: int = self.starting_tick + self.duration
        self.millicpus: float = millicpus
        self.mem_mb: float = mem_mb
        self.vram_gb: float = vram_gb
        self.gpu_utilizations: list[dict[str, float]] = gpu_utilizations

    def to_dict(self) -> dict[str, Any]:
        """
        Convert the TrainingEvent to a dictionary representation that can be output as JSON.
        """
        outer_dict: dict[str, Any] = {
            "start_tick": self.starting_tick,
            "duration_in_ticks": self.duration,
            "millicpus": self.millicpus,
            "memory": self.mem_mb,
            "num_gpus": len(self.gpu_utilizations),
            "vram": self.vram_gb,
            "gpu_utilizations": self.gpu_utilizations,
        }

        return outer_dict
