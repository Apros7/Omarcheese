package main

import (
	"math"
	"sort"
)

func roundN(v float64, digits int) float64 {
	p := math.Pow(10, float64(digits))
	return math.Round(v*p) / p
}

func fptr(v float64, digits int) *float64 {
	x := roundN(v, digits)
	return &x
}

func hoursFromWh(wh, watts *float64) *float64 {
	if wh == nil || watts == nil || *watts <= 0 || *wh <= 0 {
		return nil
	}
	return fptr(*wh / *watts, 2)
}

func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 1 {
		return sorted[0]
	}
	rank := (p / 100.0) * float64(len(sorted)-1)
	lo := int(math.Floor(rank))
	hi := int(math.Ceil(rank))
	if lo == hi {
		return sorted[lo]
	}
	w := rank - float64(lo)
	return sorted[lo]*(1-w) + sorted[hi]*w
}

func summarize(values []*float64) SummaryStats {
	nums := make([]float64, 0, len(values))
	for _, v := range values {
		if v != nil {
			nums = append(nums, *v)
		}
	}
	if len(nums) == 0 {
		return SummaryStats{}
	}
	sort.Float64s(nums)
	sum := 0.0
	for _, n := range nums {
		sum += n
	}
	mean := sum / float64(len(nums))
	stdev := 0.0
	if len(nums) > 1 {
		acc := 0.0
		for _, n := range nums {
			d := n - mean
			acc += d * d
		}
		stdev = math.Sqrt(acc / float64(len(nums)))
	}
	return SummaryStats{
		N:     len(nums),
		Mean:  fptr(mean, 3),
		P50:   fptr(percentile(nums, 50), 3),
		P95:   fptr(percentile(nums, 95), 3),
		Min:   fptr(nums[0], 3),
		Max:   fptr(nums[len(nums)-1], 3),
		Stdev: roundN(stdev, 3),
	}
}

func asInt64(v any) (int64, bool) {
	switch t := v.(type) {
	case int:
		return int64(t), true
	case int64:
		return t, true
	case uint64:
		return int64(t), true
	case float64:
		return int64(t), true
	case bool:
		if t {
			return 1, true
		}
		return 0, true
	default:
		return 0, false
	}
}

func asFloat(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case int:
		return float64(t), true
	case int64:
		return float64(t), true
	case uint64:
		return float64(t), true
	default:
		return 0, false
	}
}

func asBool(v any) (bool, bool) {
	switch t := v.(type) {
	case bool:
		return t, true
	case int, int64, float64:
		n, ok := asInt64(t)
		return n != 0, ok
	default:
		return false, false
	}
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func signedCurrentMA(raw int64) int64 {
	for _, bits := range []uint{16, 32, 64} {
		sign := int64(1) << (bits - 1)
		mask := (int64(1) << bits) - 1
		v := raw
		if bits < 64 {
			v = raw & mask
		}
		if v >= sign {
			v -= int64(1) << bits
		}
		if v < 40000 && v > -40000 {
			return v
		}
	}
	return raw
}
