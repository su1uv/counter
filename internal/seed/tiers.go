package seed

import (
	"math/rand/v2"
	"time"
)

// tierDef shapes branches of one size. Traffic is where the branch's base
// daily rate falls between Config.MinSalesPerDay (0) and MaxSalesPerDay
// (100), in percent.
type tierDef struct {
	TrafficLo, TrafficHi int
	MinStaff, MaxStaff   int
}

// tiers is only looked up by name, never iterated by the generator.
var tiers = map[string]tierDef{
	"large":  {TrafficLo: 70, TrafficHi: 100, MinStaff: 5, MaxStaff: 6},
	"medium": {TrafficLo: 30, TrafficHi: 55, MinStaff: 4, MaxStaff: 5},
	"small":  {TrafficLo: 0, TrafficHi: 20, MinStaff: 3, MaxStaff: 4},
}

// Staffing within the window, in percent.
const (
	departurePercent = 15 // chance that a seller leaves during the window
	minWeight        = 60 // productivity weight range of an employee
	maxWeight        = 140
)

// tierBase picks a branch's base daily sales within its tier's traffic range.
func tierBase(r *rand.Rand, cfg Config, t tierDef) int {
	pct := t.TrafficLo + r.IntN(t.TrafficHi-t.TrafficLo+1)
	return cfg.MinSalesPerDay + (cfg.MaxSalesPerDay-cfg.MinSalesPerDay)*pct/100
}

// onDuty returns the staff (indexes into employees) who work on day: those
// employed that day and not on their weekly day off. If everyone employed
// is off, they all work.
func onDuty(employees []Employee, staff []int, day time.Time) []int {
	var employed, working []int
	for _, i := range staff {
		e := employees[i]
		if e.HiredOn.After(day) || (!e.TerminatedOn.IsZero() && day.After(e.TerminatedOn)) {
			continue
		}
		employed = append(employed, i)
		if e.DayOff != day.Weekday() {
			working = append(working, i)
		}
	}
	if len(working) == 0 {
		return employed
	}
	return working
}
