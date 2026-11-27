package seed

import (
	"slices"
	"testing"
	"time"
)

func TestGenerateReturns(t *testing.T) {
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
			loc, _ := time.LoadLocation(cfg.Timezone)
			end := time.Date(cfg.Start.Year(), cfg.Start.Month(), cfg.Start.Day(), 0, 0, 0, 0, loc).AddDate(0, cfg.Months, 0)
			staff := staffByBranch(ds.Branches, ds.Employees)
			folios := map[string]bool{}
			lastSale := -1
			for _, ret := range ds.Returns {
				if folios[ret.Folio] {
					t.Fatalf("duplicate return folio %s", ret.Folio)
				}
				folios[ret.Folio] = true
				if ret.Sale <= lastSale || ret.Sale >= len(ds.Sales) {
					t.Fatalf("return %s has sale index %d after %d", ret.Folio, ret.Sale, lastSale)
				}
				lastSale = ret.Sale
				s := ds.Sales[ret.Sale]
				if s.Status != "completed" || ret.Folio != s.Folio+"-D" {
					t.Fatalf("return %s of %s sale %s", ret.Folio, s.Status, s.Folio)
				}
				days := dateOf(ret.ReturnedAt).Sub(dateOf(s.SoldAt)).Hours() / 24
				if days < 1 || days > maxReturnDays || !ret.ReturnedAt.Before(end) {
					t.Fatalf("return %s at %v, %v days after sale at %v", ret.Folio, ret.ReturnedAt, days, s.SoldAt)
				}
				if h := ret.ReturnedAt.Hour(); h < 11 || h >= 21 || ret.ReturnedAt.Location().String() != cfg.Timezone {
					t.Fatalf("return %s at %v, outside opening hours", ret.Folio, ret.ReturnedAt)
				}
				if duty := onDuty(ds.Employees, staff[s.Branch], dateOf(ret.ReturnedAt)); !slices.Contains(duty, ret.Employee) {
					t.Fatalf("return %s by employee %d, who is not on duty at the sale's branch", ret.Folio, ret.Employee)
				}
				if len(ret.Lines) == 0 {
					t.Fatalf("return %s has no lines", ret.Folio)
				}
				var sum int64
				seen := map[int]bool{}
				for i, l := range ret.Lines {
					if l.LineNo != i+1 || l.SaleLine < 0 || l.SaleLine >= len(s.Lines) || seen[l.SaleLine] {
						t.Fatalf("return %s line %+v invalid or repeated", ret.Folio, l)
					}
					seen[l.SaleLine] = true
					sold := s.Lines[l.SaleLine]
					if l.Quantity < 1 || l.Quantity > sold.Quantity {
						t.Fatalf("return %s returns %d of %d units", ret.Folio, l.Quantity, sold.Quantity)
					}
					if l.RefundCents <= 0 || l.RefundCents > sold.TotalCents || (l.Quantity == sold.Quantity && l.RefundCents != sold.TotalCents) {
						t.Fatalf("return %s refunds %d for line total %d", ret.Folio, l.RefundCents, sold.TotalCents)
					}
					sum += l.RefundCents
				}
				if ret.TotalCents != sum {
					t.Fatalf("return %s total %d, lines add to %d", ret.Folio, ret.TotalCents, sum)
				}
				if sub, tax := SplitTax(sum, ivaBP); ret.SubtotalCents != sub || ret.TaxCents != tax {
					t.Fatalf("return %s subtotal %d tax %d, want %d %d", ret.Folio, ret.SubtotalCents, ret.TaxCents, sub, tax)
				}
			}
		})
	}
}

func TestCancellationAndReturnRates(t *testing.T) {
	ds := mustGenerate(t, Default())
	cancelled, lines := 0, 0
	for _, s := range ds.Sales {
		if s.Status == "cancelled" {
			cancelled++
		}
		lines += len(s.Lines)
	}
	returned := 0
	for _, ret := range ds.Returns {
		returned += len(ret.Lines)
	}
	if pm := cancelled * 1000 / len(ds.Sales); pm < 7 || pm > 13 {
		t.Fatalf("%d of %d sales cancelled (%d per mille), want about 1%%", cancelled, len(ds.Sales), pm)
	}
	if pm := returned * 1000 / lines; pm < 15 || pm > 25 {
		t.Fatalf("%d of %d lines returned (%d per mille), want about 2%%", returned, lines, pm)
	}
}
