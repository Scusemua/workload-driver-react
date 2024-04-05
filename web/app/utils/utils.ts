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
