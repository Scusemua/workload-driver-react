package generator

import (
	"strconv"
	"time"
)

const (
	MaxTimeSec = 10000000000
)

type UnixTime time.Time

// UnmarshalText supports time in unix timestamp format of both seconds and nanoseconds.
func (t *UnixTime) UnmarshalText(text []byte) error {
	if ts, err := strconv.ParseInt(string(text), 10, 64); err != nil {
		return err
	} else if ts < MaxTimeSec {
		*t = UnixTime(time.Unix(ts, 0))
	} else {
		*t = UnixTime(time.Unix(0, ts))
	}
	return nil
}

func (t UnixTime) Time() time.Time {
	return time.Time(t)
}

func (t UnixTime) String() string {
	return time.Time(t).String()
}
