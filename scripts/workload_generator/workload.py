import uuid
from typing import Any

import numpy as np
import pandas as pd

from session import Session


class Workload(object):
  def __init__(
    self,
    sessions: list[Session],
    workload_name: str = "",
    max_session_millicpus: float = 8000,
    max_session_memory_mb: float = 16000,
    max_session_num_gpus: int = 8,
    max_session_gpu_utilization: float = 100,
  ):

    self.workload_name = workload_name
    if self.workload_name is None or len(self.workload_name) == 0:
      self.workload_name = str(uuid.uuid4())

    self.sessions: list[Session] = sessions

    self.max_session_millicpus: float = max_session_millicpus
    self.max_session_memory_mb: float = max_session_memory_mb
    self.max_session_num_gpus: int = max_session_num_gpus
    self.max_session_gpu_utilization: float = max_session_gpu_utilization


  def to_dict(self) -> dict[str, Any]:
    """
    Convert the workload to a dictionary representation that can be output as JSON.
    """
    outer_dict: dict[str, Any] = {
      "workloadTitle": self.workload_name,
      "workloadSeed": 0,
      "timescaleAdjustmentFactor": 0.1,
      "numberOfSessions": len(self.sessions),
      "debugLoggingEnabled": True,
    }

    sessions: list[dict[str, Any]] = []
    for session in self.sessions:
      sessions.append(session.to_dict())

    outer_dict["sessions"] = sessions

    return outer_dict
