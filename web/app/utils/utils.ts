export function GetRowspan(val: number) {
    if (val % 2 == 0) {
        return val;
    } else {
        return val + 1;
    }
}

export function numberArrayFromRange(start: number, end: number) {
    const nums: number[] = [];
    for (let i: number = start; i < end; i++) nums.push(i);
    return nums;
}

export function FormatSecondsLong(sec_num: number): string {
  const hours: string | number = Math.floor(sec_num / 3600);
  const minutes: string | number = Math.floor((sec_num - hours * 3600) / 60);
  const seconds: string | number = Math.floor(sec_num - hours * 3600 - minutes * 60);

  return hours + ' hours, ' + minutes + ' minutes, and ' + seconds + ' seconds';
}

export function FormatSecondsShort(sec_num: number): string {
  let hours: string | number = Math.floor(sec_num / 3600);
  let minutes: string | number = Math.floor((sec_num - hours * 3600) / 60);
  let seconds: string | number = Math.floor(sec_num - hours * 3600 - minutes * 60);

  if (hours < 10) {
      hours = '0' + hours;
  }

  if (minutes < 10) {
      minutes = '0' + minutes;
  }

  if (seconds < 10) {
      seconds = '0' + seconds;
  }

  return hours + 'h' + minutes + 'm' + seconds + 's';
}
