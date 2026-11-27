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
			s.shiftCash = append(s.shiftCash, 0)
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

// Cash leaving the drawer. A Monday shift records a petty expense when the
// drawer can cover it. Cash above the float, of at least 1,000 MXN, is
// withdrawn to the safe in multiples of 100 MXN. One shift in five counts
// over or short by at most 200 MXN; the rest count exact. The choice comes
// from the shift index, so it does not disturb the sales random stream.
const (
	pettyExpenseCents   = 15_000  // 150 MXN
	withdrawalMinCents  = 100_000 // 1,000 MXN
	withdrawalStepCents = 10_000  // 100 MXN
	overShortMaxPesos   = 200
)

// closeShifts counts every shift opened on day, once that day's sales and
// returns are known.
func (s *simulation) closeShifts(day time.Time) {
	for i, sh := range s.ds.Shifts {
		if dateOf(sh.OpenedAt).Equal(day) {
			s.closeShift(i)
		}
	}
}

func (s *simulation) closeShift(i int) {
	sh := s.ds.Shifts[i]
	branch := s.ds.Registers[sh.Register].Branch
	var cashOut int64
	for _, ret := range s.ds.Returns {
		if ret.ReturnedAt.Before(sh.OpenedAt) || !ret.ReturnedAt.Before(sh.ClosedAt) {
			continue
		}
		sale := s.ds.Sales[ret.Sale]
		if sale.Branch != branch {
			continue
		}
		cashOut += cashRefund(sale, ret)
	}
	available := sh.OpeningFloatCents + s.shiftCash[i] - cashOut
	if sh.OpenedAt.Weekday() == time.Monday && available >= pettyExpenseCents {
		s.ds.CashMovements = append(s.ds.CashMovements, CashMovement{
			Shift: i, MovedAt: sh.OpenedAt.Add(time.Hour),
			Kind: "expense", AmountCents: pettyExpenseCents, Note: "Gasto menor",
		})
		available -= pettyExpenseCents
	}
	if excess := available - sh.OpeningFloatCents; excess >= withdrawalMinCents {
		amount := excess / withdrawalStepCents * withdrawalStepCents
		s.ds.CashMovements = append(s.ds.CashMovements, CashMovement{
			Shift: i, MovedAt: sh.ClosedAt.Add(-30 * time.Minute),
			Kind: "withdrawal", AmountCents: amount, Note: "Retiro a caja fuerte",
		})
		available -= amount
	}
	s.ds.Shifts[i].ExpectedCashCents = available
	s.ds.Shifts[i].CountedCashCents = available + overShortCents(i)
}

// cashPaid is the cash tender of a sale. Cancelled sales have none.
func cashPaid(sale Sale) int64 {
	var n int64
	for _, p := range sale.Payments {
		if p.Method == "cash" {
			n += p.AmountCents
		}
	}
	return n
}

// cashRefund is the cash portion of a return, in the same proportion as
// the sale was paid in cash. A card-only sale refunds no cash.
func cashRefund(sale Sale, ret Return) int64 {
	cash := cashPaid(sale)
	if cash == 0 {
		return 0
	}
	return divRoundHalfUp(ret.TotalCents*cash, sale.TotalCents)
}

// overShortCents is 0 for most shifts. Every fifth shift is off by a
// non-zero whole-peso amount inside ±overShortMaxPesos.
func overShortCents(i int) int64 {
	if i%5 != 0 {
		return 0
	}
	n := int64(i%400) - int64(overShortMaxPesos)
	if n >= 0 {
		n++
	}
	return n * 100
}
