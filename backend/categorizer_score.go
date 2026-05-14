package main

import "math"

// flooredSim returns max(0, sim - floor). Similarity values below the floor
// are treated as noise and contribute zero. Callers use this so a "weakly
// related" item never sneaks score into an unrelated category.
func flooredSim(sim, floor float32) float32 {
	v := sim - floor
	if v < 0 {
		return 0
	}
	return v
}

// sumFloored sums flooredSim over a slice of cosine similarities. Empty
// input returns 0.
func sumFloored(sims []float32, floor float32) float32 {
	var sum float32
	for _, s := range sims {
		sum += flooredSim(s, floor)
	}
	return sum
}

// blend combines the recent-window and all-time per-category scores as
// recentWeight*recent + (1-recentWeight)*all. recentWeight is clamped to
// [0, 1] so callers don't have to validate it.
func blend(recent, all, recentWeight float32) float32 {
	w := recentWeight
	if w < 0 {
		w = 0
	}
	if w > 1 {
		w = 1
	}
	return w*recent + (1-w)*all
}

// popularity returns the sublinear size bias 1 + weight*log(1 + n). It is
// always >= 1 (no shrinkage) and grows slowly: doubling n increases the
// factor by less than +weight*log(2). Returns 1 (no boost) when weight <= 0
// or n <= 0 so the caller can disable the popularity signal by setting
// PopularityWeight=0.
func popularity(n int, weight float32) float32 {
	if weight <= 0 || n <= 0 {
		return 1
	}
	return 1 + weight*float32(math.Log(1+float64(n)))
}
