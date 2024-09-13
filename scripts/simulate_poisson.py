import argparse

import matplotlib.pyplot as plt
import numpy as np


#
# References:
# - https://medium.com/@abhash-rai/poisson-process-simulation-and-analysis-in-python-e62f69d1fdd0
#

def generate_poisson_events(rate: float, time_duration: float) -> tuple[int, np.ndarray, np.ndarray]:
  """
  Simulate a Poisson process by generating events with a given average rate (`rate`)
  over a specified time duration (`time_duration`).

  :param rate: the average rate of event arrival in events/second
  :param time_duration: the interval of time over which to simulate a Poisson process

  :return: a tuple[int, float, float] where the first element is the number of events, the second element is the
           time of the events, and the third element is the inter-arrival times (IAT) of th
  """
  num_events: int = np.random.poisson(rate * time_duration)
  event_times: np.ndarray = np.sort(np.random.uniform(0, time_duration, num_events))
  inter_arrival_times: np.ndarray = np.diff(event_times)
  return num_events, event_times, inter_arrival_times


def get_args():
  parser = argparse.ArgumentParser()

  parser.add_argument("-r", "--rate", nargs='+', default=[4], type=float,
                      help="Average rate or rates of event arrival(s) in events/second.")
  parser.add_argument("-t", "--time-duration", default=1.0, type=float, help="Time duration in seconds")
  parser.add_argument("-v", "--show-visualization", action='store_true')

  return parser.parse_args()


def plot_non_sequential_poisson(
  num_events: int,
  event_times: np.ndarray,
  inter_arrival_times: np.ndarray,
  rate: float,
  time_duration: float
):
  """
  Plot a non-sequential poisson process.

  :param num_events: the number of events
  :param event_times: the times at which each event occurred
  :param inter_arrival_times: the inter-arrival times (IATs) of the events
  :param rate: the average rate of event arrival in events/second
  :param time_duration: the interval of time over which we simulated the Poisson process
  """
  fig, axs = plt.subplots(1, 2, figsize=(15, 6))
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

  plt.tight_layout()
  plt.show()


def plot_sequential_poisson(
  num_events_list: list[int],
  event_times_list: list[np.ndarray],
  inter_arrival_times_list: list[np.ndarray],
  rate: list[float],
  time_duration: list[float]
):
  """
  Plot a sequence of poisson processes.

  :param num_events_list: the number of events of each of the poisson processes
  :param event_times_list: the times at which each event occurred within each poisson process
  :param inter_arrival_times_list: the inter-arrival times (IATs) of each poisson process
  :param rate: the average arrival rate of events in events/second for each poisson process
  :param time_duration: the duration, in seconds, that each poisson process was simulated for
  """
  fig, axs = plt.subplots(nrows=1, ncols=2, figsize=(15, 6))
  fig.suptitle(f'Poisson Process Simulation (Duration = {time_duration} seconds)\n', fontsize=16)

  axs[0].set_xlabel('Time')
  axs[0].set_ylabel('Event Number')
  axs[0].set_title('Poisson Process Event Times')
  axs[0].grid(True)

  axs[1].set_xlabel('Inter-Arrival Time')
  axs[1].set_ylabel('Frequency')
  axs[1].set_title('Histogram of Inter-Arrival Times')
  axs[1].grid(True, alpha=0.5)

  color_palette = plt.get_cmap('tab20')
  colors = [color_palette(i) for i in range(len(rate))]

  for n, individual_rate in enumerate(rate):
    num_events = num_events_list[n]
    event_times = event_times_list[n]
    inter_arrival_times = inter_arrival_times_list[n]

    axs[0].step(event_times, np.arange(1, num_events + 1), where='post', color=colors[n],
                label=f'λ = {individual_rate}, Total Events: {num_events}')
    axs[1].hist(inter_arrival_times, bins=20, color=colors[n], alpha=0.5,
                label=f'λ = {individual_rate}, MEAN: {np.mean(inter_arrival_times):.2f}, STD: {np.std(inter_arrival_times):.2f}')

  axs[0].legend()
  axs[1].legend()

  plt.tight_layout()
  plt.show()


def poisson_simulation(rate, time_duration, show_visualization=True):
  print(f"Simulating Poisson Process with rate={rate} and time duration={time_duration}")
  if isinstance(rate, int):
    num_events, event_times, inter_arrival_times = generate_poisson_events(rate, time_duration)

    if show_visualization:
      plot_non_sequential_poisson(num_events, event_times, inter_arrival_times, rate, time_duration)
    else:
      return num_events, event_times, inter_arrival_times

  elif isinstance(rate, list):
    num_events_list = []
    event_times_list = []
    inter_arrival_times_list = []

    for individual_rate in rate:
      num_events, event_times, inter_arrival_times = generate_poisson_events(individual_rate, time_duration)
      num_events_list.append(num_events)
      event_times_list.append(event_times)
      inter_arrival_times_list.append(inter_arrival_times)

    if show_visualization:
      plot_sequential_poisson(num_events_list, event_times_list, inter_arrival_times_list, rate, time_duration)
    else:
      return num_events_list, event_times_list, inter_arrival_times_list


def main():
  args = get_args()

  poisson_simulation(rate=args.rate, time_duration=args.time_duration, show_visualization=args.show_visualization)


if __name__ == "__main__":
  main()
