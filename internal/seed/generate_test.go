package seed

import (
	"reflect"
	"testing"
	"time"
)

func smallConfig() Config {
	cfg := Default()
	cfg.Months = 1
	cfg.Branches = 3
	cfg.Items = 40
	cfg.EmployeesPerBranch = 3
	cfg.MinSalesPerDay = 1
	cfg.MaxSalesPerDay = 4
	return cfg
}

func mustGenerate(t *testing.T, cfg Config) Dataset {
	t.Helper()
	ds, err := Generate(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return ds
}

func TestWordLists(t *testing.T) {
	if len(branchSites) != maxBranches {
		t.Fatalf("branchSites has %d entries, want %d", len(branchSites), maxBranches)
	}
	if branchSites[0].Name != "Centro" {
		t.Fatalf("first branch is %q, want Centro", branchSites[0].Name)
	}
	tiers := map[string]int{}
	for _, s := range branchSites {
		tiers[s.Tier]++
	}
	if want := map[string]int{"large": 3, "medium": 7, "small": 5}; !reflect.DeepEqual(tiers, want) {
		t.Fatalf("tiers = %v, want %v", tiers, want)
	}
	if first := []string{branchSites[0].Tier, branchSites[1].Tier, branchSites[2].Tier}; !reflect.DeepEqual(first, []string{"large", "medium", "small"}) {
		t.Fatalf("first three tiers = %v, want large, medium, small", first)
	}
	for _, c := range categoryDefs {
		if len(c.Styles) == 0 {
			t.Fatalf("category %s has no styles", c.Code)
		}
	}
}

func TestGenerateInvalidConfig(t *testing.T) {
	cfg := Default()
	cfg.Months = 0
	if _, err := Generate(cfg); err == nil {
		t.Fatal("expected error")
	}
}

func TestGenerateCatalog(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
	}{
		{"small", smallConfig()},
		{"default", Default()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.cfg
			ds := mustGenerate(t, cfg)
			start := cfg.Start

			if len(ds.Branches) != cfg.Branches {
				t.Fatalf("branches = %d, want %d", len(ds.Branches), cfg.Branches)
			}
			if len(ds.Categories) != len(categoryDefs) {
				t.Fatalf("categories = %d, want %d", len(ds.Categories), len(categoryDefs))
			}
			if len(ds.Items) != cfg.Items {
				t.Fatalf("items = %d, want %d", len(ds.Items), cfg.Items)
			}
			if want := cfg.Branches * cfg.EmployeesPerBranch; len(ds.Employees) != want {
				t.Fatalf("employees = %d, want %d", len(ds.Employees), want)
			}

			codes := map[string]bool{}
			unique := func(kind, code string) {
				t.Helper()
				if code == "" || codes[kind+code] {
					t.Fatalf("%s code %q empty or duplicated", kind, code)
				}
				codes[kind+code] = true
			}

			for _, b := range ds.Branches {
				unique("branch", b.Code)
				if b.Timezone != cfg.Timezone || b.Name == "" || b.City == "" || b.State == "" || b.Tier == "" {
					t.Fatalf("incomplete branch %+v", b)
				}
				if !b.OpenedOn.Before(start) {
					t.Fatalf("branch %s opened %v, not before %v", b.Code, b.OpenedOn, start)
				}
			}
			if ds.Branches[0].Name != "Centro" {
				t.Fatalf("first branch = %q, want Centro", ds.Branches[0].Name)
			}
			for _, c := range ds.Categories {
				unique("category", c.Code)
			}
			for _, it := range ds.Items {
				unique("sku", it.SKU)
				if it.Category < 0 || it.Category >= len(ds.Categories) {
					t.Fatalf("item %s has category index %d", it.SKU, it.Category)
				}
				if it.UnitCostCents <= 0 || it.ListPriceCents <= it.UnitCostCents {
					t.Fatalf("item %s cost %d, price %d", it.SKU, it.UnitCostCents, it.ListPriceCents)
				}
				if it.UnitCostCents*100 < it.ListPriceCents*35 || it.UnitCostCents*100 > it.ListPriceCents*50 {
					t.Fatalf("item %s cost %d outside 35-50%% of price %d", it.SKU, it.UnitCostCents, it.ListPriceCents)
				}
				if it.Name == "" || it.Material == "" || !it.Active {
					t.Fatalf("incomplete item %+v", it)
				}
			}
			managers := make([]int, len(ds.Branches))
			for _, e := range ds.Employees {
				unique("employee", e.Code)
				if e.Branch < 0 || e.Branch >= len(ds.Branches) {
					t.Fatalf("employee %s has branch index %d", e.Code, e.Branch)
				}
				switch e.Role {
				case "manager":
					managers[e.Branch]++
				case "seller":
				default:
					t.Fatalf("employee %s has role %q", e.Code, e.Role)
				}
				opened := ds.Branches[e.Branch].OpenedOn
				if e.HiredOn.Before(opened) || !e.HiredOn.Before(start) {
					t.Fatalf("employee %s hired %v, outside [%v, %v)", e.Code, e.HiredOn, opened, start)
				}
				if e.FullName == "" {
					t.Fatalf("employee %s has no name", e.Code)
				}
			}
			for b, n := range managers {
				if n != 1 {
					t.Fatalf("branch %s has %d managers, want 1", ds.Branches[b].Code, n)
				}
			}
		})
	}
}

func TestGenerateSales(t *testing.T) {
	oneItem := smallConfig()
	oneItem.Items = 1
	fixedVolume := smallConfig()
	fixedVolume.MinSalesPerDay, fixedVolume.MaxSalesPerDay = 2, 2
	tests := []struct {
		name string
		cfg  Config
	}{
		{"small", smallConfig()},
		{"one item", oneItem},
		{"fixed volume", fixedVolume},
		{"default", Default()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.cfg
			ds := mustGenerate(t, cfg)
			loc, _ := time.LoadLocation(cfg.Timezone)
			first := time.Date(cfg.Start.Year(), cfg.Start.Month(), cfg.Start.Day(), 0, 0, 0, 0, loc)
			end := first.AddDate(0, cfg.Months, 0)
			days := 0
			for d := first; d.Before(end); d = d.AddDate(0, 0, 1) {
				days++
			}

			perBranchDay := map[string]int{}
			folios := map[string]bool{}
			lastAt := make([]time.Time, len(ds.Branches))
			lastFolio := make([]string, len(ds.Branches))
			for _, s := range ds.Sales {
				if folios[s.Folio] {
					t.Fatalf("duplicate folio %s", s.Folio)
				}
				folios[s.Folio] = true
				if s.Branch < 0 || s.Branch >= len(ds.Branches) {
					t.Fatalf("sale %s has branch index %d", s.Folio, s.Branch)
				}
				if s.Employee < 0 || s.Employee >= len(ds.Employees) || ds.Employees[s.Employee].Branch != s.Branch {
					t.Fatalf("sale %s has employee %d from another branch", s.Folio, s.Employee)
				}
				if s.Status != "completed" {
					t.Fatalf("sale %s has status %q", s.Folio, s.Status)
				}
				if s.SoldAt.Location().String() != cfg.Timezone {
					t.Fatalf("sale %s in zone %v", s.Folio, s.SoldAt.Location())
				}
				if s.SoldAt.Before(first) || !s.SoldAt.Before(end) {
					t.Fatalf("sale %s at %v outside [%v, %v)", s.Folio, s.SoldAt, first, end)
				}
				if h := s.SoldAt.Hour(); h < 11 || h >= 21 {
					t.Fatalf("sale %s at hour %d, outside 11-21", s.Folio, h)
				}
				if s.SoldAt.Before(lastAt[s.Branch]) || s.Folio <= lastFolio[s.Branch] {
					t.Fatalf("sale %s at %v out of order", s.Folio, s.SoldAt)
				}
				lastAt[s.Branch], lastFolio[s.Branch] = s.SoldAt, s.Folio
				perBranchDay[ds.Branches[s.Branch].Code+s.SoldAt.Format(time.DateOnly)]++

				if n := len(s.Lines); n < 1 || n > 3 || n > len(ds.Items) {
					t.Fatalf("sale %s has %d lines", s.Folio, n)
				}
				var sum int64
				seen := map[int]bool{}
				for i, l := range s.Lines {
					if l.LineNo != i+1 {
						t.Fatalf("sale %s line %d has line_no %d", s.Folio, i, l.LineNo)
					}
					if l.Item < 0 || l.Item >= len(ds.Items) || seen[l.Item] {
						t.Fatalf("sale %s line %d has item %d (invalid or repeated)", s.Folio, l.LineNo, l.Item)
					}
					seen[l.Item] = true
					it := ds.Items[l.Item]
					if l.Quantity < 1 || l.UnitPriceCents != it.ListPriceCents || l.UnitCostCents != it.UnitCostCents || l.DiscountCents != 0 {
						t.Fatalf("sale %s line %+v does not match item %+v", s.Folio, l, it)
					}
					if l.TotalCents != int64(l.Quantity)*l.UnitPriceCents-l.DiscountCents {
						t.Fatalf("sale %s line %d total %d", s.Folio, l.LineNo, l.TotalCents)
					}
					sum += l.TotalCents
				}
				if s.TotalCents != sum {
					t.Fatalf("sale %s total %d, lines add to %d", s.Folio, s.TotalCents, sum)
				}
				if sub, tax := SplitTax(sum, ivaBP); s.SubtotalCents != sub || s.TaxCents != tax {
					t.Fatalf("sale %s subtotal %d tax %d, want %d %d", s.Folio, s.SubtotalCents, s.TaxCents, sub, tax)
				}
			}
			if days == 0 || len(ds.Sales) == 0 {
				t.Fatalf("%d days, %d sales", days, len(ds.Sales))
			}
			for d := first; d.Before(end); d = d.AddDate(0, 0, 1) {
				pct := dayPercent(d)
				lo := (cfg.MinSalesPerDay*pct*85 + 5000) / 10000
				hi := (cfg.MaxSalesPerDay*pct*115 + 5000) / 10000
				for _, b := range ds.Branches {
					key := b.Code + d.Format(time.DateOnly)
					if n := perBranchDay[key]; n < lo || n > hi {
						t.Fatalf("%s has %d sales, want %d-%d", key, n, lo, hi)
					}
				}
			}
		})
	}
}

func TestSalesCalendarShape(t *testing.T) {
	ds := mustGenerate(t, Default())
	byWeekday := map[time.Weekday]int{}
	byDate := map[string]int{}
	byHour := map[int]int{}
	for _, s := range ds.Sales {
		byWeekday[s.SoldAt.Weekday()]++
		byDate[s.SoldAt.Format(time.DateOnly)]++
		byHour[s.SoldAt.Hour()]++
	}
	for wd := time.Sunday; wd <= time.Saturday; wd++ {
		if wd != time.Saturday && byWeekday[wd] >= byWeekday[time.Saturday] {
			t.Fatalf("%v has %d sales, Saturday %d", wd, byWeekday[wd], byWeekday[time.Saturday])
		}
		if wd != time.Monday && byWeekday[wd] <= byWeekday[time.Monday] {
			t.Fatalf("%v has %d sales, Monday %d", wd, byWeekday[wd], byWeekday[time.Monday])
		}
	}
	busiest := ""
	for date, n := range byDate {
		if busiest == "" || n > byDate[busiest] {
			busiest = date
		}
	}
	if busiest != "2026-05-10" {
		t.Fatalf("busiest day is %s, want Mother's Day 2026-05-10", busiest)
	}
	// Both Tuesdays: a payday and the plain Tuesday a week earlier.
	if byDate["2026-06-30"] <= byDate["2026-06-23"] {
		t.Fatalf("payday has %d sales, plain Tuesday %d", byDate["2026-06-30"], byDate["2026-06-23"])
	}
	if byHour[18] <= byHour[11] {
		t.Fatalf("18:00 has %d sales, 11:00 %d", byHour[18], byHour[11])
	}
}

func TestItemPriceRanges(t *testing.T) {
	ds := mustGenerate(t, Default())
	for _, it := range ds.Items {
		lo, hi := int64(-1), int64(-1)
		for m := range materials {
			if materials[m].Label == it.Material {
				lo, hi = priceRange(it.Category, m)
			}
		}
		if it.ListPriceCents < lo*100 || it.ListPriceCents > hi*100 {
			t.Fatalf("item %s price %d outside [%d, %d] pesos", it.SKU, it.ListPriceCents, lo, hi)
		}
	}
}

func TestGenerateDeterministic(t *testing.T) {
	cfg := smallConfig()
	a, b := mustGenerate(t, cfg), mustGenerate(t, cfg)
	if !reflect.DeepEqual(a, b) {
		t.Fatal("same seed produced different datasets")
	}
	cfg.Seed = 2
	if c := mustGenerate(t, cfg); reflect.DeepEqual(a, c) {
		t.Fatal("different seeds produced identical datasets")
	}
}

func TestDateOf(t *testing.T) {
	loc, _ := time.LoadLocation("America/Mexico_City")
	got := dateOf(time.Date(2026, 5, 1, 23, 30, 0, 0, loc))
	if want := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Fatalf("dateOf = %v, want %v", got, want)
	}
}
