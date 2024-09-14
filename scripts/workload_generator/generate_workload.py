import argparse
import os
import json
from textwrap import indent

from typing import Any

from session import Session
from simulate_poisson import poisson_simulation
from workload import Workload
from util import get_truncated_normal

def get_args():
  parser = argparse.ArgumentParser()

  # Workload-specific arguments.
  parser.add_argument("--workload-name", type=str, default='',
                      help="The name of the workload. By default, a random UUID will be generated to serve as the name.")
  parser.add_argument("-n", "--num-sessions", default=1, type=int, help="The number of sessions to generate.")
  parser.add_argument("-o", "--output-directory", default="output", type=str, help="Path to output directory.")

  parser.add_argument("--max-session-millicpus", type=int, default=8000,
                      help="The maximum number of millicpus to generate for a given Session.")
  parser.add_argument("--max-session-memory-mb", type=float, default=16000,
                      help="The maximum amount of memory (in MB) to generate for a given Session.")
  parser.add_argument("--max-session-num-gpus", type=int, default=8,
                      help="The maximum number of GPUs to generate for a given Session.")
  # parser.add_argument("--max-session-gpu-utilization", type=float, default=100,
  #                     help="The maximum GPU utilization to generate for a given Session.")

  # Poisson process arguments.
  parser.add_argument("-i", "--iat", default=-1, type=float,
                      help="Inter-arrival time or times (in seconds). Rates are computed from this value. If both rate and IAT are specified, then rate is used. Specify a value of -1 to omit. (The default is -1.)")
  parser.add_argument("-r", "--rate", default=1, type=float,
                      help="Average rate or rates of event arrival(s) in events/second.")
  parser.add_argument("-d", "--time-duration", default=30, type=float, help="Time duration in seconds")
  parser.add_argument("-v", "--show-visualization", action='store_true')
  parser.add_argument("--shape", type=float, default=2,
                      help="Shape parameter of Gamma distribution for training task duration.")
  parser.add_argument("--scale", type=float, default=10,
                      help="Scale parameter of Gamma distribution for training task duration.")

  return parser.parse_args()


def main():
  args = get_args()

  max_session_millicpus: float = args.max_session_millicpus
  max_session_memory_mb: float = args.max_session_memory_mb
  max_session_num_gpus: int = args.max_session_num_gpus
  # max_session_gpu_utilization: float = args.max_session_gpu_utilization

  mean_num_cpus: float = (0.15 * max_session_millicpus)
  sd_num_cpus: float = (0.05 * max_session_millicpus)

  mean_mem_mb: float = (0.15 * max_session_memory_mb)
  sd_mem_mb: float = (0.05 * max_session_memory_mb)

  mean_num_gpus: float = (0.5 * max_session_num_gpus)
  sd_num_gpus: float = (0.25 * max_session_num_gpus)

  print(f"Mean #CPUs: {mean_num_cpus}, Std. Dev.: {sd_num_cpus}")
  print(f"Mean MemMB: {mean_mem_mb}, Std. Dev.: {sd_mem_mb}")
  print(f"Mean #GPUs: {mean_num_gpus}, Std. Dev.: {sd_num_gpus}")

  max_cpu_rv = get_truncated_normal(mean = mean_num_cpus, sd = sd_num_cpus, low = 1.0e-3, upp = max_session_millicpus)
  max_mem_mb_rv = get_truncated_normal(mean = mean_mem_mb, sd = sd_mem_mb, low = 1.0e-3, upp = max_session_memory_mb)
  num_gpus_rv = get_truncated_normal(mean = mean_num_gpus, sd = sd_num_gpus, low = 1, upp = max_session_num_gpus)

  max_cpus_vals = max_cpu_rv.rvs(args.num_sessions)
  max_mem_vals = max_mem_mb_rv.rvs(args.num_sessions)
  num_gpus_vals = num_gpus_rv.rvs(args.num_sessions)

  sessions: list[Session] = []
  for i in range(args.num_sessions):
    num_events, event_times, iats, durations = poisson_simulation(rate=args.rate, iat=args.iat, scale=args.scale,
                                                                  shape=args.shape, time_duration=args.time_duration,
                                                                  show_visualization=args.show_visualization)

    session: Session = Session(
      num_events[0],
      event_times[0],
      iats[0],
      durations[0],
      max_millicpus = max_cpus_vals[i],
      max_mem_mb = max_mem_vals[i],
      num_gpus = int(num_gpus_vals[i]))
    sessions.append(session)

  workload: Workload = Workload(sessions, workload_name=args.workload_name)

  os.makedirs(args.output_directory, exist_ok=True)

  workload_dict: dict[str, Any] = workload.to_dict()

  with open(os.path.join(args.output_directory, "template.json"), "w") as f:
    json.dump(workload_dict, f, indent = 2)

if __name__ == '__main__':
  main()
