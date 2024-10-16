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

def plot_session_gantt(ax: plt.Axes, event_durations, event_times, inter_arrival_times, y:int = 1, line_width:int = 15):
  label:str = f"Session {y}\n{len(event_times)} Events\nAvg IAT: {round(sum(inter_arrival_times)/len(inter_arrival_times), 2)}\nAvg Duration: {round(sum(event_durations)/len(event_durations), 2)}"

  first: bool = True

  for i in range(0, len(event_times)):
    st_tr = event_times[i]
    et_tr = st_tr + event_durations[i]

    st_idle = et_tr

    try:
      et_idle = et_tr + inter_arrival_times[i]
    except:
      et_idle = et_tr

    if first and y == 1:
      ax.hlines(y, st_tr, et_tr, color = 'green', linewidth = line_width, label = "Busy")
      ax.hlines(y, st_idle, et_idle, color = 'grey', linewidth = line_width, label = "Idle")

      first = False
    else:
      ax.hlines(y, st_tr, et_tr, color = 'green', linewidth = line_width)
      ax.hlines(y, st_idle, et_idle, color = 'grey', linewidth = line_width)

  return y, label

def plot_aggregate_session_histograms(
  sessions: list[Session],
  output_directory:str = "",
  show_visualization: bool = False,
  rate: float = -1
):
  height = int(len(sessions)  * 1.25)
  fig, axs = plt.subplots(nrows=1, ncols=3, figsize=(20, height), dpi = 256)

  if isinstance(rate, list):
    if len(rate) == 0:
      return
    rate = rate[0]

  raw_data_dir:str = os.path.join(output_directory, "raw_data")
  os.makedirs(raw_data_dir, exist_ok=True)

  # Events, step
  axs[0].set_xlabel('Time')
  axs[0].set_ylabel('Event Number')
  axs[0].set_title(f'Poisson Process Event Times (λ = {rate})')
  axs[0].grid(True)

  # IATs, histogram
  axs[1].set_xlabel('Inter-Arrival Time')
  axs[1].set_ylabel('Frequency')
  axs[1].set_title('Histogram of Inter-Arrival Times')
  axs[1].grid(True, alpha=0.5)

  # Durations, histogram
  axs[2].set_ylabel('Frequency')
  axs[2].set_xlabel('Duration (seconds)')
  axs[2].set_title(
    f'Histogram of Event Durations')
  axs[2].grid(True, alpha=0.5)

  num_events: int = 0
  inter_arrival_times: list[int] = []
  event_durations: list[float] = []
  all_event_times: list[float] = []

  ticks: list[int] = []
  labels: list[str] = []

  for i, session in enumerate(sessions):
    session_raw_data_dir:str = os.path.join(raw_data_dir, f"session_{i}")
    os.makedirs(session_raw_data_dir, exist_ok=True)

    with open(os.path.join(session_raw_data_dir, "inter_arrival_times.txt"), "w") as fp:
      fp.writelines(str(iat) + "\n" for iat in session.inter_arrival_times)

    with open(os.path.join(session_raw_data_dir, "event_times.txt"), "w") as fp:
      fp.writelines(str(event_time) + "\n" for event_time in session.event_times)

    with open(os.path.join(session_raw_data_dir, "event_durations.txt"), "w") as fp:
      fp.writelines(str(event_duration) + "\n" for event_duration in session.event_durations)

    num_events += len(session.training_events)
    inter_arrival_times.extend(session.inter_arrival_times)
    event_durations.extend(session.event_durations)
    all_event_times.extend(session.event_times)

    tick, label = plot_session_gantt(
      axs[0],
      y = 1 + i,
      event_durations = session.event_durations,
      inter_arrival_times = session.inter_arrival_times,
      event_times = session.event_times)
    ticks.append(tick)
    labels.append(label)

    #axs[0].step(session.event_times, [0] + session.inter_arrival_times, where='post', color=colors[i], alpha=0.75,
    #            label=f'Session #{i}, Total Events: {len(session.training_events)}')

  axs[0].set_yticks(ticks = ticks, labels = labels)
  axs[0].set_ylim(0, len(sessions) + 1)

  axs[1].hist(inter_arrival_times, bins=20, color="tab:green", alpha=0.5,
              label=f'λ = {rate}, MEAN: {np.mean(inter_arrival_times):.2f} sec, STD: {np.std(inter_arrival_times):.2f} sec')
  axs[2].hist(event_durations, bins=20, color='tab:red', alpha=0.65,
              label=f'Mean: {np.mean(event_durations):.2f} sec | STD: {np.std(event_durations):.2f} sec')

  axs[0].legend(prop={'size': 16})
  axs[1].legend()
  axs[2].legend()

  plt.tight_layout()

  if output_directory is not None and len(output_directory) > 0:
    filename = f"all_sessions_poisson"

    os.makedirs(output_directory, exist_ok=True)
    plt.savefig(os.path.join(output_directory, f"{filename}.png"), bbox_inches = 'tight')
    plt.savefig(os.path.join(output_directory, f"{filename}.pdf"), bbox_inches = 'tight')

  with open(os.path.join(raw_data_dir, "inter_arrival_times.txt"), "w") as fp:
    fp.writelines(str(iat) + "\n" for iat in inter_arrival_times)

  with open(os.path.join(raw_data_dir, "event_times.txt"), "w") as fp:
    fp.writelines([str(event_time) + "\n" for event_time in all_event_times])

  with open(os.path.join(raw_data_dir, "event_durations.txt"), "w") as fp:
    fp.writelines([str(event_duration) + "\n" for event_duration in event_durations])

  if show_visualization:
    plt.show()

def plot_resource_histograms(cpu, mem, gpu, output_dir: str, show_visualization: bool):
  num_sessions: int = len(cpu)

  fig, axs = plt.subplots(1, 3, figsize=(15, 5), dpi = 256)
  fig.suptitle(f'Workload Resource Distribution (NumSessions = {num_sessions})\n', fontsize=16)

  cpu_bins = np.histogram_bin_edges(cpu, bins = "fd", range=(0,np.max(cpu)))
  print("cpu_bins:", cpu_bins)
  axs[0].hist(cpu, bins=cpu_bins, color='red', alpha=0.75)
  axs[0].set_xlabel("Millicpus (1/1000th core)")
  axs[0].set_ylabel("Frequency")
  axs[0].set_title(
    f'Histogram of Max Millicpus\nMEAN: {np.mean(cpu):.2f} | STD: {np.std(cpu):.2f}\n')
  axs[0].grid(True, alpha=0.5)

  memory_bins = np.histogram_bin_edges(mem, bins = "fd", range=(0,np.max(mem)))
  print("memory_bins:", memory_bins)
  axs[1].hist(mem, bins=memory_bins, color='red', alpha=0.75)
  axs[1].set_xlabel("Memory (MB)")
  axs[1].set_ylabel("Frequency")
  axs[1].set_title(
    f'Histogram of Max Memory Usage (MB)\nMEAN: {np.mean(mem):.2f} | STD: {np.std(mem):.2f}\n')
  axs[1].grid(True, alpha=0.5)

  gpu_bins: list = [0,1,2,3,4,5,6,7,8,9]
  axs[2].hist(gpu, bins=gpu_bins, color='red', alpha=0.75)
  axs[2].set_xticks(gpu_bins)
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

  mean_num_gpus: float = 1.5
  sd_num_gpus: float = 1.5

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
        f"Submitting sessions {split[0]} - {split[len(split) - 1]} ({split[len(split) - 1] - split[0] + 1} in total) for processing.")
      res = pool.apply_async(generate_session,
                             (args.rate, args.iat, args.scale, args.shape, args.time_duration, output_directory,
                              args.show_visualization, split[0], split[len(split) - 1]+1, max_cpus_vals,
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

  print("Creating workload object now")
  workload: Workload = Workload(sessions, workload_name=args.workload_name)

  workload_dict: dict[str, Any] = workload.to_dict()

  plt.style.use('ggplot')

  plot_resource_histograms(max_cpus_vals, max_mem_vals, num_gpus_vals,
                           output_dir=output_directory, show_visualization=args.show_visualization)

  plot_aggregate_session_histograms(sessions, output_directory=output_directory, show_visualization=args.show_visualization, rate=args.rate)

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