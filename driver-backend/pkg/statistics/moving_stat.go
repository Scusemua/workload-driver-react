package statistics

import (
	"github.com/shopspring/decimal"
)

type MovingStat struct {
	window    int64
	n         int64
	values    []decimal.Decimal
	last      int64
	sum       [2]decimal.Decimal // include a moving sum(0) and a reset(1) used to ensuring moving sum by adding only.
	active    int
	resetting int
}

func NewMovingStat(window int64) *MovingStat {
	return &MovingStat{
		window:    window,
		n:         0,
		values:    make([]decimal.Decimal, window),
		last:      0,
		active:    0,
		resetting: 1,
	}
}

func (s *MovingStat) Add(val decimal.Decimal) {
	// Move forward.
	s.last = (s.last + 1) % s.window

	// Add difference to sum.
	s.sum[s.active] = s.sum[s.active].Add(val.Sub(s.values[s.last]))
	s.sum[s.resetting] = s.sum[s.resetting].Add(val) // Resetting is used to sum from ground above each window interval.
	if s.last == 0 {
		s.active = s.resetting
		s.resetting = (s.resetting + 1) % len(s.sum)
		s.sum[s.resetting] = decimal.Zero.Copy()
	}

	// Record history value
	s.values[s.last] = val

	// update length
	if s.n < s.window {
		s.n += 1
	}
}

func (s *MovingStat) Sum() decimal.Decimal {
	return s.sum[s.active]
}

func (s *MovingStat) Window() int64 {
	return s.window
}

func (s *MovingStat) N() int64 {
	return s.n
}

func (s *MovingStat) Avg() decimal.Decimal {
	return s.sum[s.active].Div(decimal.NewFromFloat(float64(s.n)))
}

// PopulationVariance computes and returns the population variance of the data currently within the moving stat/window
func (s *MovingStat) PopulationVariance() decimal.Decimal {
	avg := s.Avg()
	variance := decimal.Zero

	for _, n := range s.values {
		diff := n.Sub(avg)
		variance = variance.Add(diff.Mul(diff))
	}

	return variance.Div(decimal.NewFromFloat(float64(s.N())))
}

// SampleVariance computes and returns the sample variance of the data currently within the moving stat/window
func (s *MovingStat) SampleVariance() decimal.Decimal {
	avg := s.Avg()
	variance := decimal.Zero

	for _, n := range s.values {
		diff := n.Sub(avg)
		variance = variance.Add(diff.Mul(diff))
	}

	return variance.Div(decimal.NewFromFloat(float64(s.N() - 1)))
}

func (s *MovingStat) PopulationStandardDeviation() decimal.Decimal {
	return s.PopulationVariance().Pow(decimal.NewFromFloat(0.5))
}

func (s *MovingStat) SampleStandardDeviation() decimal.Decimal {
	return s.SampleVariance().Pow(decimal.NewFromFloat(0.5))
}

func (s *MovingStat) Last() decimal.Decimal {
	return s.values[s.last]
}

func (s *MovingStat) LastN(n int64) decimal.Decimal {
	if n > s.n {
		n = s.n
	}
	return s.values[(s.last+s.window-n)%s.window]
}
