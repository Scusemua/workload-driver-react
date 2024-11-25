import { Workload } from '@src/Data';

export function GetRowspan(val: number) {
    if (val % 2 == 0) {
        return val;
    } else {
        return val + 1;
    }
}

export function numberWithCommas(x: number): string {
    return x.toString().replace(/\B(?=(\d{3})+(?!\d))/g, ',');
}

export function isNumber(value?: string | string[] | number): boolean {
    return value != null && value !== '' && !Array.isArray(value) && !isNaN(Number(value.toString()));
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

/**
 * Convert the given unix milliseconds duration time to a human-readable string.
 */
export function UnixDurationToString(ts: number): string {
    let formattedTime: string = '';
    let runningTs: number = ts;
    const hours: number = Math.floor(ts / 3.6e6);

    if (hours > 0) {
        formattedTime += hours + 'hr ';
        runningTs -= hours * 3.6e6;
    }

    const minutes: number = Math.floor(runningTs / 6e4);
    if (minutes > 0) {
        formattedTime += minutes + 'min ';
        runningTs -= minutes * 6e4;
    }

    const seconds: number = RoundToTwoDecimalPlaces(runningTs / 1e3);
    if (seconds > 0) {
        formattedTime += seconds + 'sec';
    }

    formattedTime = formattedTime.trimEnd();

    return formattedTime;
}

/**
 * Convert the given Unix Milliseconds timestamp to a human-readable string.
 */
export function UnixTimestampToDateString(unixTimestamp: number): string {
    const date: Date = new Date(unixTimestamp * 1000);
    const months: string[] = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];
    const year: number = date.getFullYear();
    const month: string = months[date.getMonth()];
    const day: number = date.getDate();
    const hour: number = date.getHours();
    const min: number = date.getMinutes();
    const sec: number = date.getSeconds();
    return day + ' ' + month + ' ' + year + ' ' + hour + ':' + min + ':' + sec;
}

/**
 * Export the workload to JSON.
 *
 * @param workload the workload to be exported.
 * @param filename the filename to use, including the file extension. if unspecified,
 *                 then filename will be set to a string of the form "workload_ID.json"
 */
export function ExportWorkloadToJson(workload: Workload, filename?: string | undefined) {
    const downloadElement: HTMLAnchorElement = document.createElement('a');
    downloadElement.setAttribute(
        'href',
        'data:text/json;charset=utf-8,' + encodeURIComponent(JSON.stringify(workload, null, 2)),
    );

    if (filename !== undefined && filename !== '') {
        downloadElement.setAttribute('download', filename);
    } else {
        downloadElement.setAttribute('download', `workload_${workload.id}.json`);
    }

    downloadElement.style.display = 'none';
    document.body.appendChild(downloadElement);

    downloadElement.click();

    document.body.removeChild(downloadElement);
}

function RoundToTwoDecimalPlaces(num: number) {
    return +(Math.round(Number.parseFloat(num.toString() + 'e+2')).toString() + 'e-2');
}

function RoundToThreeDecimalPlaces(num: number) {
    return +(Math.round(Number.parseFloat(num.toString() + 'e+3')).toString() + 'e-3');
}

function RoundToNDecimalPlaces(num: number, n: number) {
    return +(Math.round(Number.parseFloat(num.toString() + `e+${n}`)).toString() + `e-${n}`);
}

export { RoundToTwoDecimalPlaces as RoundToTwoDecimalPlaces };
export { RoundToThreeDecimalPlaces as RoundToThreeDecimalPlaces };
export { RoundToNDecimalPlaces as RoundToNDecimalPlaces };
