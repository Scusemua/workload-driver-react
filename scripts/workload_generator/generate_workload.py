import argparse
import datetime
import json
import os
import time
from multiprocessing import Pool
from typing import Any

import matplotlib.pyplot as plt
import numpy as np

from session import Session
from simulate_poisson import poisson_simulation
from util import get_truncated_normal
from workload import Workload


def get_args():
  parser = argparse.ArgumentParser()

  parser.add_argument( "--num-procs", type = int, default = 1, help = "Number of processes to use.")

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


def plot_resource_histograms(cpu, mem, gpu, output_dir: str, show_visualization: bool):
  num_sessions: int = len(cpu)

  fig, axs = plt.subplots(1, 3, figsize=(15, 6))
  fig.suptitle(f'Workload Resource Distribution (NumSessions = {num_sessions})\n', fontsize=16)

  axs[0].hist(cpu, bins=20, color='red', alpha=0.75)
  axs[0].set_xlabel("Millicpus (1/1000th core)")
  axs[0].set_ylabel("Frequency")
  axs[0].set_title(
    f'Histogram of Max Millicpus\nMEAN: {np.mean(cpu):.2f} | STD: {np.std(cpu):.2f}\n')
  axs[0].grid(True, alpha=0.5)

  axs[1].hist(mem, bins=20, color='red', alpha=0.75)
  axs[1].set_xlabel("Memory (MB)")
  axs[1].set_ylabel("Frequency")
  axs[1].set_title(
    f'Histogram of Max Memory Usage (MB)\nMEAN: {np.mean(mem):.2f} | STD: {np.std(mem):.2f}\n')
  axs[1].grid(True, alpha=0.5)

  axs[2].hist(gpu, bins=20, color='red', alpha=0.75)
  axs[2].set_xlabel("GPUs")
  axs[2].set_ylabel("Frequency")
  axs[2].set_title(
    f'Histogram of Number of GPUs\nMEAN: {np.mean(gpu):.2f} | STD: {np.std(gpu):.2f}\n')
  axs[2].grid(True, alpha=0.5)

  plt.tight_layout()

  if output_dir is not None and len(output_dir) > 0:
    plt.savefig(os.path.join(output_dir, "workload_resource_histogram.png"), bbox_inches='tight')
    plt.savefig(os.path.join(output_dir, "workload_resource_histogram.pdf"), bbox_inches='tight')

  if show_visualization:
    plt.show()


def create_splits(a, n):
  k, m = divmod(len(a), n)
  return (a[i * k + min(i, m):(i + 1) * k + min(i + 1, m)] for i in range(n))


def main():
  args = get_args()
  start_time = time.time()

  output_directory: str = os.path.join(args.output_directory,
                                       "template-{date:%Y-%m-%d_%H-%M-%S}".format(date=datetime.datetime.now()))
  os.makedirs(output_directory, exist_ok=False)

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

  max_cpu_rv = get_truncated_normal(mean=mean_num_cpus, sd=sd_num_cpus, low=1.0e-3, upp=max_session_millicpus)
  max_mem_mb_rv = get_truncated_normal(mean=mean_mem_mb, sd=sd_mem_mb, low=1.0e-3, upp=max_session_memory_mb)
  num_gpus_rv = get_truncated_normal(mean=mean_num_gpus, sd=sd_num_gpus, low=1, upp=max_session_num_gpus)

  max_cpus_vals = max_cpu_rv.rvs(args.num_sessions)
  max_mem_vals = max_mem_mb_rv.rvs(args.num_sessions)
  num_gpus_vals = num_gpus_rv.rvs(args.num_sessions)

  if args.num_procs > 1:
    cpus_to_run_on = args.num_procs
    indices = list(range(args.num_sessions))
    splits = create_splits(indices, cpus_to_run_on)

    pool = Pool(processes=cpus_to_run_on)

    results = []
    for split in splits:
      print(
        f"Submitting sessions {split[0]} - {split[len(split) - 1] + 1} ({split[len(split) - 1] - split[0] + 1} in total) for processing.")
      res = pool.apply_async(generate_session,
                             (args.rate, args.iat, args.scale, args.shape, args.time_duration, output_directory,
                              args.show_visualization, split[0], split[len(split) - 1] + 1, max_cpus_vals,
                              max_mem_vals, num_gpus_vals))
      results.append(res)

    print("Aggregating sessions now")

    sessions: list = [None for _ in range(0, args.num_sessions)]
    for res in results:
      ret = res.get()
      ses = ret[0]
      st_idx = ret[1]
      end_idx = ret[2]

      print(f"Received sessions {st_idx} - {end_idx}.")

      ctr = 0
      for j in range(st_idx, end_idx, 1):
        sessions[j] = ses[ctr]
        ctr += 1
  else:
    ret = generate_session(args.rate, args.iat, args.scale, args.shape, args.time_duration,
                                               output_directory, args.show_visualization, 0, args.num_sessions,
                                               max_cpus_vals, max_mem_vals, num_gpus_vals)
    sessions: list[Session] = ret[0]

  # sessions: list[Session] = []
  # for i in range(args.num_sessions):
  #   num_events, event_times, iats, durations = poisson_simulation(rate=args.rate, iat=args.iat, scale=args.scale,
  #                                                                 shape=args.shape, time_duration=args.time_duration,
  #                                                                 output_directory=output_directory,
  #                                                                 show_visualization=args.show_visualization,
  #                                                                 session_index=i)
  #
  #   session: Session = Session(
  #     num_events[0],
  #     event_times[0],
  #     iats[0],
  #     durations[0],
  #     max_millicpus=max_cpus_vals[i],
  #     max_mem_mb=max_mem_vals[i],
  #     num_gpus=int(num_gpus_vals[i]))
  #   sessions.append(session)

  print("Creating workload object now")
  workload: Workload = Workload(sessions, workload_name=args.workload_name)

  workload_dict: dict[str, Any] = workload.to_dict()

  plot_resource_histograms(max_cpus_vals, max_mem_vals, num_gpus_vals,
                           output_dir=output_directory, show_visualization=args.show_visualization)

  with open(os.path.join(output_directory, "template.json"), "w") as f:
    json.dump(workload_dict, f, indent=2)

  print("Done generating workload. Time elapsed: ", time.time() - start_time)


def generate_session(rate, iat, scale, shape, time_duration, output_directory, show_visualization, start_idx, end_idx,
                     max_cpus_vals, max_mem_vals, num_gpus_vals):
  print(f"Creating sessions {start_idx}-{end_idx} now (total of {end_idx - start_idx + 1} sessions).")
  sessions: list[Session] = []
  st = time.time()
  for i in range(start_idx, end_idx, 1):
    num_events, event_times, iats, durations = poisson_simulation(rate=rate, iat=iat, scale=scale,
                                                                  shape=shape, time_duration=time_duration,
                                                                  output_directory=output_directory,
                                                                  show_visualization=show_visualization,
                                                                  session_index=i)
    session: Session = Session(
      num_events[0],
      event_times[0],
      iats[0],
      durations[0],
      max_millicpus=max_cpus_vals[i],
      max_mem_mb=max_mem_vals[i],
      num_gpus=int(num_gpus_vals[i]))
    sessions.append(session)

  print(
    f"Finished generating sessions {start_idx} -- {end_idx} in {time.time() - st} seconds (total of {end_idx - start_idx + 1} sessions).")

  return [sessions, start_idx, end_idx]


if __name__ == '__main__':
  main()
