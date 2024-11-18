import argparse
from typing import Any, Tuple, List
import os
import matplotlib.pyplot as plt
import numpy as np
import pandas as pd
from numpy import ndarray


#
# References:
# - https://medium.com/@abhash-rai/poisson-process-simulation-and-analysis-in-python-e62f69d1fdd0
#

def generate_poisson_events(rate: float, time_duration: float, shape: float, scale: float) -> tuple[
  int, list[ndarray[float]], ndarray[float], ndarray[float]]:
  """
  Simulate a Poisson process by generating events with a given average rate (`rate`)
  over a specified time duration (`time_duration`).

  :param rate: the average rate of event arrival in events/second
  :param time_duration: the interval of time over which to simulate a Poisson process
  :param shape: shape parameter of Gamma distribution for training task duration
  :param scale: scale parameter of Gamma distribution for training task duration

  :return: a tuple[int, np.ndarray, np.ndarray, np.ndarray] where the first element is the number of events, the second
           element is the time of the events, and the third element is the inter-arrival times (IAT) of the events,
           and the fourth element is the durations of each event.
  """
  print(f"Simulating Poisson process with event arrival rate of {rate} events/sec for {time_duration} seconds.")
  print(f"Event durations generated using Geometric distribution with shape={shape} and scale={scale}.")
  num_events: int = np.random.poisson(rate * time_duration)
  print(f"Poisson process will have {num_events} event(s).")

  if num_events == 0:
    print("Poisson process will have no events.")
    print("Try adjusting your input parameters (such as the rate or duration).")
    exit(1)

  init_event_times: np.ndarray = np.sort(np.random.uniform(1, time_duration, num_events))
  inter_arrival_times: np.ndarray = np.diff(init_event_times)
  event_durations: np.ndarray = np.random.gamma(shape, scale=scale, size=num_events)

  event_times = [init_event_times[0]]
  duration_sum: float = event_durations[0]
  for i in range(1, num_events):
    event_time = init_event_times[i]
    event_time += duration_sum
    duration_sum += event_durations[i]
    event_times.append(event_time)

  return num_events, event_times, inter_arrival_times, event_durations


# def generate_poisson_events(rate: float, time_duration: float) -> tuple[int, np.ndarray, np.ndarray]:
#   """
#   Simulate a Poisson process by generating events with a given average rate (`rate`)
#   over a specified time duration (`time_duration`).
#
#   :param rate: the average rate of event arrival in events/second
#   :param time_duration: the interval of time over which to simulate a Poisson process
#
#   :return: a tuple[int, float, float] where the first element is the number of events, the second element is the
#            time of the events, and the third element is the inter-arrival times (IAT) of th
#   """
#   num_events: int = np.random.poisson(rate * time_duration)
#   event_times: np.ndarray = np.sort(np.random.uniform(0, time_duration, num_events))
#   inter_arrival_times: np.ndarray = np.diff(event_times)
#   return num_events, event_times, inter_arrival_times

def generate_poisson_iats(rate: float, num_events: int):
  """
  Generates inter-arrival times (IATs) for a Poisson process.
  """
  return -np.log(1 - np.random.rand(num_events)) / rate


def get_args():
  parser = argparse.ArgumentParser()

  parser.add_argument("-i", "--iat", nargs='+', default=[], type=float,
                      help="Inter-arrival time or times (in seconds). Rates are computed from this value. If both rate and IAT are specified, then rate is used.")
  parser.add_argument("-r", "--rate", nargs='+', default=[], type=float,
                      help="Average rate or rates of event arrival(s) in events/second.")
  parser.add_argument("-d", "--time-duration", default=1.0, type=float, help="Time duration in seconds")
  parser.add_argument("-v", "--show-visualization", action='store_true')

  parser.add_argument("--shape", type=float, default=2,
                      help="Shape parameter of Gamma distribution for training task duration.")
  parser.add_argument("--scale", type=float, default=10,
                      help="Scale parameter of Gamma distribution for training task duration.")

  return parser.parse_args()


def plot_non_sequential_poisson(
  num_events: int,
  event_times: np.ndarray,
  inter_arrival_times: np.ndarray,
  event_durations: np.ndarray,
  rate: float,
  time_duration: float
):
  """
  Plot a non-sequential poisson process.

  :param num_events: the number of events
  :param event_times: the times at which each event occurred
  :param inter_arrival_times: the inter-arrival times (IATs) of the events
  :param event_durations: durations of each event in seconds
  :param rate: the average rate of event arrival in events/second
  :param time_duration: the interval of time over which we simulated the Poisson process
  """
  fig, axs = plt.subplots(1, 3, figsize=(15, 6))
  fig.suptitle(f'Poisson Process Simulation (λ = {rate}, Duration = {time_duration} seconds)\n', fontsize=16)

  axs[0].step(event_times, np.arange(1, num_events + 1), where='post', color='blue')
  axs[0].set_xlabel('Time')
  axs[0].set_ylabel('Event Number')
  axs[0].set_title(f'Poisson Process Event Times\nTotal: {num_events} events\n')
  axs[0].grid(True)

  axs[1].hist(inter_arrival_times, bins=20, color='green', alpha=0.5)
  axs[1].set_xlabel('Inter-Arrival Time')
  axs[1].set_ylabel('Frequency')
  axs[1].set_title(
    f'Histogram of Inter-Arrival Times\nMEAN: {np.mean(inter_arrival_times):.2f} | STD: {np.std(inter_arrival_times):.2f}\n')
  axs[1].grid(True, alpha=0.5)

  axs[2].hist(event_durations, bins=20, color='red', alpha=0.75)
  axs[2].set_xlabel('Duration (seconds)')
  axs[2].set_ylabel('Frequency')
  axs[2].set_title(
    f'Histogram of Event Durations\nMEAN: {np.mean(event_durations):.2f} | STD: {np.std(event_durations):.2f}\n')
  axs[2].grid(True, alpha=0.5)

  plt.tight_layout()
  plt.show()


def plot_sequential_poisson(
  num_events_list: list[int],
  event_times_list: list[np.ndarray],
  inter_arrival_times_list: list[np.ndarray],
  event_durations_list: list[np.ndarray],
  rate: list[float],
  time_duration: float,
  show_visualization: bool = False,
  output_directory: str = "",
  session_index: int = -1,
):
  """
  Plot a sequence of poisson processes.

  :param show_visualization: if true, also display the output plots
  :param output_directory: directory in which to write the output plots
  :param num_events_list: the number of events of each of the poisson processes
  :param event_times_list: the times at which each event occurred within each poisson process
  :param inter_arrival_times_list: the inter-arrival times (IATs) of each poisson process
  :param event_durations_list: durations of each event in seconds
  :param rate: the average arrival rate of events in events/second for each poisson process
  :param time_duration: the duration, in seconds, that each poisson process was simulated for
  """
  fig, axs = plt.subplots(nrows=1, ncols=3, figsize=(15, 6))
  fig.suptitle(f'Poisson Process Simulation (Duration = {time_duration} seconds)\n', fontsize=16)

  # Events, step
  axs[0].set_xlabel('Time')
  axs[0].set_ylabel('Event Number')
  axs[0].set_title('Poisson Process Event Times')
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

  color_palette = plt.get_cmap('tab20')
  colors = [color_palette(i) for i in range(len(rate))]

  for n, individual_rate in enumerate(rate):
    num_events = num_events_list[n]
    event_times = event_times_list[n]
    inter_arrival_times = inter_arrival_times_list[n]
    event_durations = event_durations_list[n]

    axs[0].step(event_times, np.arange(1, num_events + 1), where='post', color=colors[n],
                label=f'λ = {individual_rate}, Total Events: {num_events}')
    axs[1].hist(inter_arrival_times, bins=20, color=colors[n], alpha=0.5,
                label=f'λ = {individual_rate}, MEAN: {np.mean(inter_arrival_times):.2f} sec, STD: {np.std(inter_arrival_times):.2f} sec')
    axs[2].hist(event_durations, bins=20, color='red', alpha=0.65,
                label=f'Mean: {np.mean(event_durations):.2f} sec | STD: {np.std(event_durations):.2f} sec')

  axs[0].legend()
  axs[1].legend()
  axs[2].legend()

  plt.tight_layout()

  if output_directory is not None and len(output_directory) > 0:
    filename: str = "poisson"
    if session_index >= 0:
      filename = f"session_{session_index}_poisson"

    directory = os.path.join(output_directory, "poisson_plots")
    os.makedirs(directory, exist_ok=True)
    plt.savefig(os.path.join(directory, f"{filename}.png"), bbox_inches = 'tight')
    plt.savefig(os.path.join(directory, f"{filename}.pdf"), bbox_inches = 'tight')

  if show_visualization:
    plt.show()


def poisson_simulation(
  rate: List[float] | float,
  iat: List[float] | float,
  time_duration: float,
  shape: float,
  scale: float,
  show_visualization: bool = True,
  output_directory: str = "",
  session_index: int = -1,
) -> Tuple[List[int], List[List[ndarray[Any, Any]]], List[ndarray], List[ndarray]]:
  if not isinstance(rate, list):
    rate = [rate]

  if not isinstance(iat, list):
    iat = [iat]

  if len(rate) == 0:
    assert len(iat) > 0

    rate = [1 / t for t in iat]
  elif len(iat) == 0:
    assert len(rate) > 0

  print(f"Simulating Poisson Process with rate={rate} and time duration={time_duration}")

  if isinstance(rate, list):
    num_events_list = []
    event_times_list = []
    inter_arrival_times_list = []
    event_durations_list = []

    for individual_rate in rate:
      num_events, event_times, inter_arrival_times, event_durations = generate_poisson_events(individual_rate,
                                                                                              time_duration, shape,
                                                                                              scale)
      num_events_list.append(num_events)
      event_times_list.append(event_times)
      inter_arrival_times_list.append(inter_arrival_times)
      event_durations_list.append(event_durations)

    plot_sequential_poisson(num_events_list, event_times_list, inter_arrival_times_list, event_durations_list, rate,
                            time_duration, output_directory = output_directory, show_visualization = show_visualization,
                            session_index = session_index)
    return num_events_list, event_times_list, inter_arrival_times_list, event_durations_list


def main():
  args = get_args()

  if len(args.rate) == 0 and len(args.iat) == 0:
    print("[ERROR] Must specify at least one rate or at least one IAT.")
    exit(1)

  num_events, event_times, iats, durations = poisson_simulation(rate=args.rate, iat=args.iat, scale=args.scale,
                                                                shape=args.shape, time_duration=args.time_duration,
                                                                show_visualization=args.show_visualization)

  # data = {
  #   "timestamp": event_times,
  #   "inter_arrival_time": iats,
  #   "duration": durations
  # }

  _iats = [0] + list(iats[0])
  print(f"event_times ({len(event_times[0])}): {event_times[0]}")
  print(f"iats ({len(_iats)}): {_iats}")
  print(f"durations ({len(durations[0])}): {durations[0]}")

  df = pd.DataFrame(np.column_stack([event_times[0], _iats, durations[0]]), columns=["ts", "iat", "dur"])
  print(df)


if __name__ == "__main__":
  main()
