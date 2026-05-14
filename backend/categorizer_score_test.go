package main

import (
	"math"
	"testing"
)

const scoreEpsilon = 1e-6

func approxEqual(a, b, eps float32) bool {
	d := a - b
	if d < 0 {
		d = -d
	}
	return d <= eps
}

func TestFlooredSim(t *testing.T) {
	cases := []struct {
		name  string
		sim   float32
		floor float32
		want  float32
	}{
		{"below_floor", 0.50, 0.55, 0},
		{"exactly_floor", 0.55, 0.55, 0},
		{"just_above", 0.70, 0.55, 0.15},
		{"max_above", 1.00, 0.55, 0.45},
		{"negative_sim_zeroed", -0.20, 0.55, 0},
		{"zero_floor_passes_through", 0.42, 0.0, 0.42},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := flooredSim(c.sim, c.floor)
			if !approxEqual(got, c.want, scoreEpsilon) {
				t.Fatalf("flooredSim(%v, %v) = %v, want %v", c.sim, c.floor, got, c.want)
			}
		})
	}

	t.Run("monotone_non_decreasing_in_sim", func(t *testing.T) {
		var floor float32 = 0.55
		var prev float32 = -1
		for s := -0.2; s <= 1.0; s += 0.05 {
			got := flooredSim(float32(s), floor)
			if got < prev {
				t.Fatalf("monotonicity violated at sim=%v: prev=%v got=%v", s, prev, got)
			}
			prev = got
		}
	})
}

func TestSumFloored(t *testing.T) {
	const floor float32 = 0.55

	t.Run("empty", func(t *testing.T) {
		if got := sumFloored(nil, floor); got != 0 {
			t.Fatalf("sumFloored(nil) = %v, want 0", got)
		}
	})

	t.Run("mixed_values", func(t *testing.T) {
		sims := []float32{0.40, 0.60, 0.80, 0.90}
		want := flooredSim(0.40, floor) + flooredSim(0.60, floor) +
			flooredSim(0.80, floor) + flooredSim(0.90, floor)
		got := sumFloored(sims, floor)
		if !approxEqual(got, want, scoreEpsilon) {
			t.Fatalf("sumFloored mixed = %v, want %v", got, want)
		}
	})

	t.Run("all_below_floor", func(t *testing.T) {
		sims := []float32{0.10, 0.20, 0.40, 0.54}
		if got := sumFloored(sims, floor); got != 0 {
			t.Fatalf("sumFloored all-below = %v, want 0", got)
		}
	})
}

func TestBlend(t *testing.T) {
	cases := []struct {
		name                 string
		recent, all, w, want float32
	}{
		{"default_blend", 0.8, 0.2, 0.7, 0.7*0.8 + 0.3*0.2},
		{"w_zero_uses_all", 0.8, 0.2, 0.0, 0.2},
		{"w_one_uses_recent", 0.8, 0.2, 1.0, 0.8},
		{"w_clamped_below_zero", 0.8, 0.2, -0.5, 0.2},
		{"w_clamped_above_one", 0.8, 0.2, 1.5, 0.8},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := blend(c.recent, c.all, c.w)
			if !approxEqual(got, c.want, scoreEpsilon) {
				t.Fatalf("blend(%v, %v, %v) = %v, want %v",
					c.recent, c.all, c.w, got, c.want)
			}
		})
	}
}

func TestPopularity(t *testing.T) {
	t.Run("zero_n_no_boost", func(t *testing.T) {
		if got := popularity(0, 0.3); got != 1 {
			t.Fatalf("popularity(0, 0.3) = %v, want 1", got)
		}
	})

	t.Run("negative_n_no_boost", func(t *testing.T) {
		if got := popularity(-5, 0.3); got != 1 {
			t.Fatalf("popularity(-5, 0.3) = %v, want 1", got)
		}
	})

	t.Run("zero_weight_no_boost", func(t *testing.T) {
		if got := popularity(10, 0); got != 1 {
			t.Fatalf("popularity(10, 0) = %v, want 1", got)
		}
	})

	t.Run("matches_formula", func(t *testing.T) {
		want := float32(1 + 0.3*math.Log(1+10))
		got := popularity(10, 0.3)
		if !approxEqual(got, want, 1e-5) {
			t.Fatalf("popularity(10, 0.3) = %v, want %v", got, want)
		}
	})

	t.Run("monotone_strictly_increasing", func(t *testing.T) {
		var prev float32 = 1
		for n := 1; n <= 100; n++ {
			got := popularity(n, 0.3)
			if got <= prev {
				t.Fatalf("expected strictly increasing at n=%d: prev=%v got=%v",
					n, prev, got)
			}
			prev = got
		}
	})

	t.Run("sublinear_growth", func(t *testing.T) {
		// Doubling n by 2.5x should NOT multiply the factor by 2.5x.
		// popularity(10) / popularity(4) << 2.5.
		ratio := popularity(10, 0.3) / popularity(4, 0.3)
		if ratio >= 2.5 {
			t.Fatalf("expected sublinear growth, got ratio=%v >= 2.5", ratio)
		}
		// Also confirm it's strictly > 1 (otherwise popularity does nothing).
		if ratio <= 1 {
			t.Fatalf("expected ratio > 1 (popularity should grow with n), got %v", ratio)
		}
	})
}
