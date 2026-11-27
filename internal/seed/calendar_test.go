package seed

import (
	"math/rand/v2"
	"testing"
	"time"
)

func day(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func TestDayPercent(t *testing.T) {
	tests := []struct {
		name string
		day  time.Time
		want int
	}{
		{"plain monday", day(2026, 5, 4), 70},
		{"plain saturday", day(2026, 5, 2), 140},
		{"plain sunday", day(2026, 5, 3), 100},
		{"payday 15th on friday", day(2026, 5, 15), 115 * 130 / 100},
		{"payday last day of 31-day month", day(2026, 5, 31), 130},
		{"payday last day of 30-day month", day(2026, 6, 30), 80 * 130 / 100},
		{"payday last day of february", day(2027, 2, 28), 100 * 130 / 100},
		{"leap day is payday", day(2028, 2, 29), 80 * 130 / 100},
		{"february 28 of a leap year is not", day(2028, 2, 28), 70},
		{"mother's day eve", day(2026, 5, 9), 140 * 160 / 100},
		{"mother's day", day(2026, 5, 10), 250},
		{"valentine's day on sunday", day(2027, 2, 14), 100 * 180 / 100},
		{"christmas eve on thursday", day(2026, 12, 24), 90 * 200 / 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := dayPercent(tt.day); got != tt.want {
				t.Fatalf("dayPercent(%s) = %d, want %d", tt.day.Format(time.DateOnly), got, tt.want)
			}
		})
	}
}

func TestIsPayday(t *testing.T) {
	tests := []struct {
		day  time.Time
		want bool
	}{
		{day(2026, 5, 15), true},
		{day(2026, 5, 31), true},
		{day(2026, 5, 30), false},
		{day(2026, 5, 1), false},
		{day(2026, 6, 30), true},
	}
	for _, tt := range tests {
		if got := isPayday(tt.day); got != tt.want {
			t.Fatalf("isPayday(%s) = %v, want %v", tt.day.Format(time.DateOnly), got, tt.want)
		}
	}
}

func TestWeightedIndex(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	weights := []int{0, 1, 0, 3}
	counts := make([]int, len(weights))
	for range 4000 {
		counts[weightedIndex(r, weights)]++
	}
	if counts[0] != 0 || counts[2] != 0 {
		t.Fatalf("zero weights picked: %v", counts)
	}
	if counts[3] < 2*counts[1] {
		t.Fatalf("weight 3 picked %d times, weight 1 %d times", counts[3], counts[1])
	}
}

func TestSaleSecond(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	hours := map[int]int{}
	for range 20000 {
		sec := saleSecond(r)
		if sec < opensAt || sec >= closesAt {
			t.Fatalf("second %d outside opening hours", sec)
		}
		hours[sec/3600]++
	}
	if len(hourWeights) != (closesAt-opensAt)/3600 {
		t.Fatalf("hourWeights has %d entries for %d opening hours", len(hourWeights), (closesAt-opensAt)/3600)
	}
	if hours[18] <= hours[11] || hours[18] <= hours[20] {
		t.Fatalf("no afternoon peak: %v", hours)
	}
}

func TestIsMonthEnd(t *testing.T) {
	tests := []struct {
		day  time.Time
		want bool
	}{
		{time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC), true},
		{time.Date(2026, 6, 30, 21, 5, 0, 0, time.UTC), true},
		{time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC), true},
		{time.Date(2028, 2, 28, 0, 0, 0, 0, time.UTC), false},
		{time.Date(2026, 5, 30, 0, 0, 0, 0, time.UTC), false},
		{time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC), false},
	}
	for _, tt := range tests {
		if got := isMonthEnd(tt.day); got != tt.want {
			t.Errorf("isMonthEnd(%v) = %v, want %v", tt.day, got, tt.want)
		}
	}
}
