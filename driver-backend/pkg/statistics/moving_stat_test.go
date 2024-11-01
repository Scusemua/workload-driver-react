package statistics

import (
	"math"
	"testing"

	"github.com/shopspring/decimal"
)

func NewMovingStatN(n int, window int) *MovingStat {
	MovingStat := NewMovingStat(int64(window))
	for i := 1; i <= n; i++ {
		MovingStat.Add(decimal.NewFromFloat(float64(i)))
	}
	return MovingStat
}

func TestPopulationVariance(t *testing.T) {
	MovingStat := NewMovingStat(5)

	MovingStat.Add(decimal.NewFromFloat(float64(2)))
	if !MovingStat.Sum().Equal(decimal.NewFromFloat(2.0)) {
		t.Logf("wrong sum of 2 with window 5, want: %v, got: %v", 2, MovingStat.Sum().StringFixed(4))
		t.Fail()
	}

	MovingStat.Add(decimal.NewFromFloat(float64(3)))
	if !MovingStat.Sum().Equal(decimal.NewFromFloat(float64(5))) {
		t.Logf("wrong sum of 2,3 with window 5, want: %v, got: %v", 5, MovingStat.Sum().StringFixed(4))
		t.Fail()
	}

	MovingStat.Add(decimal.NewFromFloat(float64(4)))
	if !MovingStat.Sum().Equal(decimal.NewFromFloat(float64(9))) {
		t.Logf("wrong sum of 2,3,4 with window 5, want: %v, got: %v", 9, MovingStat.Sum().StringFixed(4))
		t.Fail()
	}

	MovingStat.Add(decimal.NewFromFloat(float64(5)))
	if !MovingStat.Sum().Equal(decimal.NewFromFloat(float64(14))) {
		t.Logf("wrong sum of 2,3,4,5 with window 5, want: %v, got: %v", 14, MovingStat.Sum().StringFixed(4))
		t.Fail()
	}

	MovingStat.Add(decimal.NewFromFloat(float64(6)))
	if !MovingStat.Sum().Equal(decimal.NewFromFloat(float64(20))) {
		t.Logf("wrong sum of 2,3,4,5,6 with window 5, want: %v, got: %v", 20, MovingStat.Sum().StringFixed(4))
		t.Fail()
	}

	if !MovingStat.PopulationVariance().Equal(decimal.NewFromFloat(2)) {
		t.Logf("wrong population variance of 2,3,4,5,6 with window 5, want: %v, got: %v", 2, MovingStat.PopulationVariance().StringFixed(4))
		t.Fail()
	}
}

func TestSampleVariance(t *testing.T) {
	MovingStat := NewMovingStat(5)

	MovingStat.Add(decimal.NewFromFloat(float64(2)))
	MovingStat.Add(decimal.NewFromFloat(float64(3)))
	MovingStat.Add(decimal.NewFromFloat(float64(4)))
	MovingStat.Add(decimal.NewFromFloat(float64(5)))
	MovingStat.Add(decimal.NewFromFloat(float64(6)))

	if !MovingStat.SampleVariance().Equal(decimal.NewFromFloat(2.5)) {
		t.Logf("wrong sample variance of 2,3,4,5,6 with window 5, want: %v, got: %v", 2.5, MovingStat.SampleVariance().StringFixed(4))
		t.Fail()
	}
}

func TestSampleStandardDeviation(t *testing.T) {
	MovingStat := NewMovingStat(5)

	MovingStat.Add(decimal.NewFromFloat(float64(2)))
	MovingStat.Add(decimal.NewFromFloat(float64(3)))
	MovingStat.Add(decimal.NewFromFloat(float64(4)))
	MovingStat.Add(decimal.NewFromFloat(float64(5)))
	MovingStat.Add(decimal.NewFromFloat(float64(6)))

	epsilon := decimal.NewFromFloat(1.0e-6)
	diff := MovingStat.SampleStandardDeviation().Sub(decimal.NewFromFloat(1.5811388))

	if !diff.LessThanOrEqual(epsilon) {
		t.Logf("wrong sample variance of 2,3,4,5,6 with window 5, want: %v, got: %v", math.Sqrt(2.5), MovingStat.SampleStandardDeviation().StringFixed(8))
		t.Fail()
	}
}

func TestPopulationStandardDeviation(t *testing.T) {
	MovingStat := NewMovingStat(5)

	MovingStat.Add(decimal.NewFromFloat(float64(2)))
	MovingStat.Add(decimal.NewFromFloat(float64(3)))
	MovingStat.Add(decimal.NewFromFloat(float64(4)))
	MovingStat.Add(decimal.NewFromFloat(float64(5)))
	MovingStat.Add(decimal.NewFromFloat(float64(6)))

	epsilon := decimal.NewFromFloat(1.0e-6)
	diff := MovingStat.PopulationStandardDeviation().Sub(decimal.NewFromFloat(1.4142136))

	if !diff.LessThanOrEqual(epsilon) {
		t.Logf("wrong sample variance of 2,3,4,5,6 with window 5, want: %v, got: %v", math.Sqrt(2.5), MovingStat.PopulationStandardDeviation().StringFixed(8))
		t.Fail()
	}
}

func TestSum(t *testing.T) {
	MovingStat := NewMovingStatN(1, 5)
	if !MovingStat.Sum().Equal(decimal.NewFromFloat(1.0)) {
		t.Logf("wrong sum of 1 with window 5, want: %v, got: %v", 1, MovingStat.Sum())
		t.Fail()
	}

	MovingStat.Add(decimal.NewFromFloat(float64(2)))
	if !MovingStat.Sum().Equal(decimal.NewFromFloat(float64(3))) {
		t.Logf("wrong sum of 1-2 with window 5, want: %v, got: %v", 3, MovingStat.Sum())
		t.Fail()
	}

	MovingStat = NewMovingStatN(4, 5)
	if !MovingStat.Sum().Equal(decimal.NewFromFloat(float64(10))) {
		t.Logf("wrong sum of 1-4 with window 5, want: %v, got: %v", 10, MovingStat.Sum())
		t.Fail()
	}

	MovingStat.Add(decimal.NewFromFloat(float64(5)))
	if !MovingStat.Sum().Equal(decimal.NewFromFloat(float64(15))) {
		t.Logf("wrong sum of 1-5 with window 5, want: %v, got: %v", 15, MovingStat.Sum())
		t.Fail()
	}

	MovingStat.Add(decimal.NewFromFloat(float64(6)))
	if !MovingStat.Sum().Equal(decimal.NewFromFloat(float64(20))) {
		t.Logf("wrong sum of 1-6 with window 5, want: %v, got: %v", 20, MovingStat.Sum())
		t.Fail()
	}

	MovingStat.Add(decimal.NewFromFloat(float64(7)))
	if !MovingStat.Sum().Equal(decimal.NewFromFloat(float64(25))) {
		t.Logf("wrong sum of 1-7 with window 5, want: %v, got: %v", 25, MovingStat.Sum())
		t.Fail()
	}

	MovingStat.Add(decimal.NewFromFloat(float64(8)))
	if !MovingStat.Sum().Equal(decimal.NewFromFloat(float64(30))) {
		t.Logf("wrong sum of 1-8 with window 5, want: %v, got: %v", 30, MovingStat.Sum())
		t.Fail()
	}

	MovingStat.Add(decimal.NewFromFloat(float64(9)))
	if !MovingStat.Sum().Equal(decimal.NewFromFloat(float64(35))) {
		t.Logf("wrong sum of 1-9 with window 5, want: %v, got: %v", 35, MovingStat.Sum())
		t.Fail()
	}

	MovingStat.Add(decimal.NewFromFloat(float64(10)))
	if !MovingStat.Sum().Equal(decimal.NewFromFloat(float64(40))) {
		t.Logf("wrong sum of 1-10 with window 5, want: %v, got: %v", 40, MovingStat.Sum())
		t.Fail()
	}

	MovingStat.Add(decimal.NewFromFloat(float64(11)))
	if !MovingStat.Sum().Equal(decimal.NewFromFloat(float64(45))) {
		t.Logf("wrong sum of 1-11 with window 5, want: %v, got: %v", 45, MovingStat.Sum())
		t.Fail()
	}
}

func TestLast(t *testing.T) {
	MovingStat := NewMovingStatN(1, 5)
	if !MovingStat.Last().Equal(decimal.NewFromFloat(1.0)) {
		t.Logf("wrong last of 1 with window 5, want: %v, got: %v", 1, MovingStat.Last())
		t.Fail()
	}

	MovingStat.Add(decimal.NewFromFloat(float64(2)))
	if !MovingStat.Last().Equal(decimal.NewFromFloat(float64(2))) {
		t.Logf("wrong last of 1-2 with window 5, want: %v, got: %v", 2, MovingStat.Last())
		t.Fail()
	}

	MovingStat = NewMovingStatN(4, 5)
	if !MovingStat.Last().Equal(decimal.NewFromFloat(float64(4))) {
		t.Logf("wrong last of 1-4 with window 5, want: %v, got: %v", 4, MovingStat.Last())
		t.Fail()
	}

	MovingStat.Add(decimal.NewFromFloat(float64(5)))
	if !MovingStat.Last().Equal(decimal.NewFromFloat(float64(5))) {
		t.Logf("wrong last of 1-5 with window 5, want: %v, got: %v", 5, MovingStat.Last())
		t.Fail()
	}

	MovingStat.Add(decimal.NewFromFloat(float64(6)))
	if !MovingStat.Last().Equal(decimal.NewFromFloat(float64(6))) {
		t.Logf("wrong last of 1-6 with window 5, want: %v, got: %v", 6, MovingStat.Last())
		t.Fail()
	}
}
