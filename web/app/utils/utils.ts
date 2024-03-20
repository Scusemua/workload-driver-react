export function GetRowspan(val: number) {
    if (val % 2 == 0) {
        return val;
    } else {
        return val + 1;
    }
}
