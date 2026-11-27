package seed

import (
	"math/rand/v2"
	"time"
)

// Opening hours, in seconds after local midnight.
const (
	opensAt  = 11 * 3600
	closesAt = 21 * 3600
)

// weekdayPercent scales daily traffic, indexed by time.Weekday (Sunday
// first): Saturday is the busiest day and Monday the quietest.
var weekdayPercent = [7]int{100, 70, 80, 85, 90, 115, 140}

// paydayPercent applies on the 15th and the last day of the month.
const paydayPercent = 130

// hourWeights covers each opening hour from 11:00, peaking at 18:00.
var hourWeights = []int{4, 6, 8, 8, 7, 8, 10, 12, 11, 6}

type calendarEvent struct {
	Month   time.Month
	Day     int
	Percent int
}

var calendarEvents = []calendarEvent{
	{time.February, 14, 180}, // San Valentín
	{time.May, 9, 160},       // eve of Día de las Madres
	{time.May, 10, 250},      // Día de las Madres
	{time.December, 24, 200}, // Nochebuena
}

// dayPercent is the traffic multiplier for a calendar day, in percent of a
// plain Sunday. The weekday, payday and event factors multiply.
func dayPercent(day time.Time) int {
	p := weekdayPercent[day.Weekday()]
	if isPayday(day) {
		p = p * paydayPercent / 100
	}
	for _, e := range calendarEvents {
		if day.Month() == e.Month && day.Day() == e.Day {
			p = p * e.Percent / 100
		}
	}
	return p
}

func isPayday(day time.Time) bool {
	return day.Day() == 15 || day.AddDate(0, 0, 1).Day() == 1
}

// dailySales scales a branch's base daily rate by the calendar and by a
// random 85-115% jitter, rounding to the nearest sale.
func dailySales(r *rand.Rand, base int, day time.Time) int {
	return (base*dayPercent(day)*(85+r.IntN(31)) + 5000) / 10000
}

// saleSecond picks a time of day within opening hours, following
// hourWeights.
func saleSecond(r *rand.Rand) int {
	return opensAt + weightedIndex(r, hourWeights)*3600 + r.IntN(3600)
}

// weightedIndex picks an index with probability proportional to its weight.
// At least one weight must be positive.
func weightedIndex(r *rand.Rand, weights []int) int {
	total := 0
	for _, w := range weights {
		total += w
	}
	n := r.IntN(total)
	last := len(weights) - 1
	for i, w := range weights[:last] {
		if n < w {
			return i
		}
		n -= w
	}
	return last
}
