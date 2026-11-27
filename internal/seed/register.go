package seed

import "time"

// openingFloatCents is the cash in the drawer when a shift opens: 3,000 MXN.
const openingFloatCents = 300_000

// shiftSplit is 16:00, the middle of the 11:00–21:00 day. A busy day is
// two shifts meeting here; a quiet day is one shift across the whole day.
const shiftSplit = 16 * 3600

// generateRegisters gives each branch one register. The slice index matches
// the branch index.
func generateRegisters(branches []Branch) []Register {
	out := make([]Register, len(branches))
	for i, b := range branches {
		out[i] = Register{Code: b.Code + "-C1", Branch: i, Name: "Caja 1"}
	}
	return out
}

// openShifts opens the branch registers for day, before any sale. A day at
// least as busy as a Friday is split into a morning and an evening shift.
// The manager opens; on a split day a seller takes the evening when one is
// on duty.
func (s *simulation) openShifts(day time.Time) {
	y, m, d := day.Date()
	at := func(sec int) time.Time {
		return time.Date(y, m, d, 0, 0, sec, 0, s.loc)
	}
	bounds := [][2]int{{opensAt, closesAt}}
	if dayPercent(day) >= weekdayPercent[time.Friday] {
		bounds = [][2]int{{opensAt, shiftSplit}, {shiftSplit, closesAt}}
	}
	s.openToday = make([][]int, len(s.ds.Branches))
	for b := range s.ds.Branches {
		for n, span := range bounds {
			s.openToday[b] = append(s.openToday[b], len(s.ds.Shifts))
			s.ds.Shifts = append(s.ds.Shifts, Shift{
				Register:          b,
				OpenedBy:          s.opener(b, day, n),
				OpenedAt:          at(span[0]),
				ClosedAt:          at(span[1]),
				OpeningFloatCents: openingFloatCents,
			})
		}
	}
}

// opener picks who opens shift which of branch on day. which 0 wants the
// manager and any later shift wants a seller; either falls back to whoever
// is on duty.
func (s *simulation) opener(branch int, day time.Time, which int) int {
	duty := onDuty(s.ds.Employees, s.staff[branch], day)
	want := "manager"
	if which > 0 {
		want = "seller"
	}
	for _, i := range duty {
		if s.ds.Employees[i].Role == want {
			return i
		}
	}
	return duty[0]
}

// shiftAt returns the shift of branch that is open at. The branch's
// register has the same index as the branch.
func (s *simulation) shiftAt(branch int, at time.Time) int {
	found := s.openToday[branch][0]
	for _, i := range s.openToday[branch] {
		sh := s.ds.Shifts[i]
		if !at.Before(sh.OpenedAt) && at.Before(sh.ClosedAt) {
			found = i
		}
	}
	return found
}
