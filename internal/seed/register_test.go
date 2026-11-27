package seed

import (
	"slices"
	"testing"
	"time"
)

func TestGenerateShifts(t *testing.T) {
	cfg := smallConfig()
	cfg.Months = 2
	ds := mustGenerate(t, cfg)
	if len(ds.Registers) != len(ds.Branches) {
		t.Fatalf("registers = %d, want %d", len(ds.Registers), len(ds.Branches))
	}
	for i, b := range ds.Branches {
		reg := ds.Registers[i]
		if reg.Code != b.Code+"-C1" || reg.Name != "Caja 1" || reg.Branch != i {
			t.Fatalf("register %+v, want %s Caja 1 at branch %d", reg, b.Code+"-C1", i)
		}
	}
	if len(ds.Sales) == 0 || len(ds.Shifts) == 0 {
		t.Fatal("expected sales and shifts")
	}

	loc, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		t.Fatal(err)
	}
	staff := staffByBranch(ds.Branches, ds.Employees)
	type dayKey struct {
		reg int
		day string
	}
	byDay := map[dayKey][]Shift{}
	for _, sh := range ds.Shifts {
		if sh.Register < 0 || sh.Register >= len(ds.Registers) {
			t.Fatalf("shift register %d out of range", sh.Register)
		}
		if sh.OpeningFloatCents != openingFloatCents {
			t.Fatalf("opening float %d, want %d", sh.OpeningFloatCents, openingFloatCents)
		}
		if !sh.ClosedAt.After(sh.OpenedAt) {
			t.Fatalf("shift %v does not close after it opens", sh.OpenedAt)
		}
		branch := ds.Registers[sh.Register].Branch
		day := dateOf(sh.OpenedAt)
		if !slices.Contains(onDuty(ds.Employees, staff[branch], day), sh.OpenedBy) {
			t.Fatalf("opener %d is not on duty at branch %d on %s", sh.OpenedBy, branch, day.Format(time.DateOnly))
		}
		byDay[dayKey{sh.Register, sh.OpenedAt.In(loc).Format(time.DateOnly)}] = append(
			byDay[dayKey{sh.Register, sh.OpenedAt.In(loc).Format(time.DateOnly)}], sh)
	}

	end := cfg.Start.AddDate(0, cfg.Months, 0)
	for d := cfg.Start; d.Before(end); d = d.AddDate(0, 0, 1) {
		want := 1
		if dayPercent(d) >= weekdayPercent[time.Friday] {
			want = 2
		}
		for r := range ds.Registers {
			shifts := byDay[dayKey{r, d.Format(time.DateOnly)}]
			if len(shifts) != want {
				t.Fatalf("%s register %d has %d shifts, want %d", d.Format(time.DateOnly), r, len(shifts), want)
			}
			wantOpen := []int{opensAt}
			wantClose := []int{closesAt}
			if want == 2 {
				wantOpen = []int{opensAt, shiftSplit}
				wantClose = []int{shiftSplit, closesAt}
			}
			for i, sh := range shifts {
				if clockSeconds(sh.OpenedAt) != wantOpen[i] || clockSeconds(sh.ClosedAt) != wantClose[i] {
					t.Fatalf("%s shift %d is %s–%s, want the planned hours", d.Format(time.DateOnly), i, sh.OpenedAt, sh.ClosedAt)
				}
				if i > 0 && !sh.OpenedAt.Equal(shifts[i-1].ClosedAt) {
					t.Fatalf("%s shifts overlap or leave a gap", d.Format(time.DateOnly))
				}
			}
		}
	}

	for _, sale := range ds.Sales {
		if sale.Shift < 0 || sale.Shift >= len(ds.Shifts) {
			t.Fatalf("sale %s shift %d out of range", sale.Folio, sale.Shift)
		}
		sh := ds.Shifts[sale.Shift]
		if sale.SoldAt.Before(sh.OpenedAt) || !sale.SoldAt.Before(sh.ClosedAt) {
			t.Fatalf("sale %s at %v is outside its shift %v–%v", sale.Folio, sale.SoldAt, sh.OpenedAt, sh.ClosedAt)
		}
		if ds.Registers[sh.Register].Branch != sale.Branch {
			t.Fatalf("sale %s is at branch %d, its shift is at branch %d", sale.Folio, sale.Branch, ds.Registers[sh.Register].Branch)
		}
		if dayPercent(dateOf(sale.SoldAt)) < weekdayPercent[time.Friday] {
			continue
		}
		wantOpen := opensAt
		if sale.SoldAt.Hour() >= 16 {
			wantOpen = shiftSplit
		}
		if clockSeconds(sh.OpenedAt) != wantOpen {
			t.Fatalf("sale %s at %v was rung on the shift opening at %s", sale.Folio, sale.SoldAt, sh.OpenedAt)
		}
	}
}

func TestOpener(t *testing.T) {
	day := time.Date(2026, 5, 4, 0, 0, 0, 0, time.UTC) // Monday
	hired := day.AddDate(-1, 0, 0)
	employees := []Employee{
		{Branch: 0, Role: "manager", HiredOn: hired, DayOff: time.Tuesday},
		{Branch: 0, Role: "seller", HiredOn: hired, DayOff: time.Wednesday},
	}
	s := &simulation{
		ds:    &Dataset{Employees: employees},
		staff: staffByBranch([]Branch{{}}, employees),
	}
	if got := s.opener(0, day, 0); got != 0 {
		t.Fatalf("manager on duty, opener = %d, want 0", got)
	}
	if got := s.opener(0, day, 1); got != 1 {
		t.Fatalf("second shift opener = %d, want the seller", got)
	}

	employees[0].DayOff = time.Monday
	if got := s.opener(0, day, 0); got != 1 {
		t.Fatalf("manager off, opener = %d, want the seller", got)
	}
	employees[1].DayOff = time.Monday
	employees[0].DayOff = time.Tuesday
	if got := s.opener(0, day, 1); got != 0 {
		t.Fatalf("no seller on duty, opener = %d, want the manager", got)
	}
}

func TestOverShort(t *testing.T) {
	tests := []struct {
		shift int
		want  int64
	}{
		{0, -20000},
		{1, 0},
		{4, 0},
		{5, -19500},
		{200, 100},
		{395, 19600},
	}
	for _, tt := range tests {
		if got := overShortCents(tt.shift); got != tt.want {
			t.Fatalf("overShortCents(%d) = %d, want %d", tt.shift, got, tt.want)
		}
	}
}

func TestCashRefund(t *testing.T) {
	tests := []struct {
		name string
		sale Sale
		ret  Return
		want int64
	}{
		{"card only", Sale{TotalCents: 1000, Payments: []Payment{{Method: "card", AmountCents: 1000}}}, Return{TotalCents: 1000}, 0},
		{"all cash", Sale{TotalCents: 1000, Payments: []Payment{{Method: "cash", AmountCents: 1000}}}, Return{TotalCents: 400}, 400},
		{"split rounds half up", Sale{TotalCents: 1000, Payments: []Payment{{Method: "cash", AmountCents: 500}, {Method: "card", AmountCents: 500}}}, Return{TotalCents: 1}, 1},
	}
	for _, tt := range tests {
		if got := cashRefund(tt.sale, tt.ret); got != tt.want {
			t.Fatalf("%s: cash refund %d, want %d", tt.name, got, tt.want)
		}
	}
}

func TestCloseShift(t *testing.T) {
	monday := time.Date(2026, 5, 4, 11, 0, 0, 0, time.UTC)
	tuesday := monday.AddDate(0, 0, 1)
	cashSale := Sale{Branch: 0, TotalCents: 500000, Payments: []Payment{{Method: "cash", AmountCents: 500000}}}
	tests := []struct {
		name         string
		opened       time.Time
		float, cash  int64
		refund       bool
		wantExpected int64
		wantKinds    []string
	}{
		{"Monday expense", monday, 300000, 0, false, 285000, []string{"expense"}},
		{"Monday drawer too small for an expense", monday, 0, 0, false, 0, nil},
		{"withdrawal down to the float", tuesday, 300000, 250000, false, 300000, []string{"withdrawal"}},
		{"withdrawal rounded down to 100 pesos", tuesday, 300000, 149999, false, 309999, []string{"withdrawal"}},
		{"no withdrawal below 1000 pesos", tuesday, 300000, 99999, false, 399999, nil},
		{"cash refund larger than the drawer", tuesday, 1000, 0, true, -499000, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			closed := tt.opened.Add(10 * time.Hour)
			s := &simulation{ds: &Dataset{
				Registers: []Register{{Branch: 0}},
				Shifts: []Shift{{
					Register: 0, OpenedAt: tt.opened, ClosedAt: closed, OpeningFloatCents: tt.float,
				}},
			}, shiftCash: []int64{tt.cash}}
			if tt.refund {
				s.ds.Sales = []Sale{cashSale}
				s.ds.Returns = []Return{{Sale: 0, ReturnedAt: tt.opened.Add(2 * time.Hour), TotalCents: cashSale.TotalCents}}
			}
			s.closeShift(0)
			sh := s.ds.Shifts[0]
			if sh.ExpectedCashCents != tt.wantExpected {
				t.Fatalf("expected %d, want %d", sh.ExpectedCashCents, tt.wantExpected)
			}
			if sh.CountedCashCents != tt.wantExpected+overShortCents(0) {
				t.Fatalf("counted %d, want %d", sh.CountedCashCents, tt.wantExpected+overShortCents(0))
			}
			if len(s.ds.CashMovements) != len(tt.wantKinds) {
				t.Fatalf("%d movements, want %d", len(s.ds.CashMovements), len(tt.wantKinds))
			}
			for i, kind := range tt.wantKinds {
				m := s.ds.CashMovements[i]
				if m.Kind != kind || m.Shift != 0 || m.AmountCents <= 0 {
					t.Fatalf("movement %+v, want kind %s", m, kind)
				}
				if m.MovedAt.Before(sh.OpenedAt) || !m.MovedAt.Before(sh.ClosedAt) {
					t.Fatalf("%s at %v is outside %v–%v", kind, m.MovedAt, sh.OpenedAt, sh.ClosedAt)
				}
			}
		})
	}
}

func TestCashCounts(t *testing.T) {
	ds := mustGenerate(t, Default())
	byShift := make([][]CashMovement, len(ds.Shifts))
	expenses, withdrawals := 0, 0
	for _, m := range ds.CashMovements {
		if m.Shift < 0 || m.Shift >= len(ds.Shifts) {
			t.Fatalf("movement shift %d out of range", m.Shift)
		}
		sh := ds.Shifts[m.Shift]
		if m.MovedAt.Before(sh.OpenedAt) || !m.MovedAt.Before(sh.ClosedAt) || m.AmountCents <= 0 || m.Note == "" {
			t.Fatalf("movement %+v is outside shift %v–%v", m, sh.OpenedAt, sh.ClosedAt)
		}
		switch m.Kind {
		case "expense":
			expenses++
			if m.AmountCents != pettyExpenseCents || m.Note != "Gasto menor" || sh.OpenedAt.Weekday() != time.Monday {
				t.Fatalf("expense %+v on %s", m, sh.OpenedAt.Weekday())
			}
			if !m.MovedAt.Equal(sh.OpenedAt.Add(time.Hour)) {
				t.Fatalf("expense at %v, want an hour after open", m.MovedAt)
			}
		case "withdrawal":
			withdrawals++
			if m.Note != "Retiro a caja fuerte" || m.AmountCents < withdrawalMinCents || m.AmountCents%withdrawalStepCents != 0 {
				t.Fatalf("withdrawal %+v", m)
			}
			if !m.MovedAt.Equal(sh.ClosedAt.Add(-30 * time.Minute)) {
				t.Fatalf("withdrawal at %v, want 30 minutes before close", m.MovedAt)
			}
		default:
			t.Fatalf("kind %q", m.Kind)
		}
		byShift[m.Shift] = append(byShift[m.Shift], m)
	}
	if expenses == 0 || withdrawals == 0 {
		t.Fatalf("%d expenses, %d withdrawals; want some of each", expenses, withdrawals)
	}

	var cashIn, cashOut []int64
	cashIn = make([]int64, len(ds.Shifts))
	cashOut = make([]int64, len(ds.Shifts))
	var paid, refunded int64
	for _, sale := range ds.Sales {
		if sale.Status != "completed" {
			continue
		}
		got := cashPaid(sale)
		cashIn[sale.Shift] += got
		paid += got
	}
	for _, ret := range ds.Returns {
		sale := ds.Sales[ret.Sale]
		got := cashRefund(sale, ret)
		refunded += got
		for i, sh := range ds.Shifts {
			if ds.Registers[sh.Register].Branch == sale.Branch && !ret.ReturnedAt.Before(sh.OpenedAt) && ret.ReturnedAt.Before(sh.ClosedAt) {
				cashOut[i] += got
			}
		}
	}
	var attributed int64
	for _, n := range cashOut {
		attributed += n
	}
	if attributed != refunded {
		t.Fatalf("cash refunds attributed %d, returns sum to %d", attributed, refunded)
	}

	exact := 0
	for i, sh := range ds.Shifts {
		var out int64
		for _, m := range byShift[i] {
			out += m.AmountCents
		}
		want := sh.OpeningFloatCents + cashIn[i] - cashOut[i] - out
		if sh.ExpectedCashCents != want {
			t.Fatalf("shift %d expected %d, cash flows sum to %d", i, sh.ExpectedCashCents, want)
		}
		if sh.CountedCashCents != want+overShortCents(i) {
			t.Fatalf("shift %d counted %d, want %d", i, sh.CountedCashCents, want+overShortCents(i))
		}
		if abs64(sh.CountedCashCents-sh.ExpectedCashCents) > int64(overShortMaxPesos)*100 {
			t.Fatalf("shift %d is off by %d centavos", i, sh.CountedCashCents-sh.ExpectedCashCents)
		}
		if sh.CountedCashCents == sh.ExpectedCashCents {
			exact++
		}
	}
	if exact*2 <= len(ds.Shifts) {
		t.Fatalf("%d of %d shifts count exact, want most", exact, len(ds.Shifts))
	}
	if paid == 0 {
		t.Fatal("expected some cash sales")
	}
}

func abs64(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}

func clockSeconds(t time.Time) int {
	h, m, s := t.Clock()
	return h*3600 + m*60 + s
}
