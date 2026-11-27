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

func clockSeconds(t time.Time) int {
	h, m, s := t.Clock()
	return h*3600 + m*60 + s
}
