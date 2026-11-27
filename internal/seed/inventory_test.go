package seed

import (
	"reflect"
	"testing"
	"time"
)

func TestUnitsPerSaleMilli(t *testing.T) {
	// 1.38 lines per sale times 1.10 units per line.
	if got := unitsPerSaleMilli(); got != 1518 {
		t.Fatalf("unitsPerSaleMilli = %d, want 1518", got)
	}
}

func TestStockPolicies(t *testing.T) {
	tests := []struct {
		name string
		base []int
		mix  [][]int
		want [][]stockPolicy
	}{
		{"no traffic keeps one to sell and one to spare", []int{0}, [][]int{{5}},
			[][]stockPolicy{{{ReorderPoint: 1, Max: 2}}}},
		// 10 sales a day sell 15.18 units, half of them of each item.
		{"busy items scale with demand", []int{10}, [][]int{{1, 1}},
			[][]stockPolicy{{{ReorderPoint: 92, Max: 228}, {ReorderPoint: 92, Max: 228}}}},
		{"rare item gets the floor", []int{1}, [][]int{{999, 1}},
			[][]stockPolicy{{{ReorderPoint: 19, Max: 46}, {ReorderPoint: 1, Max: 2}}}},
		{"per branch", []int{2, 0}, [][]int{{1}, {1}},
			[][]stockPolicy{{{ReorderPoint: 37, Max: 92}}, {{ReorderPoint: 1, Max: 2}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := stockPolicies(tt.base, tt.mix); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("stockPolicies = %+v, want %+v", got, tt.want)
			}
		})
	}
}

// TestLedgerMatchesSalesAndReturns checks that every completed sale line
// takes stock once, every return line puts it back once, and nothing else
// does.
func TestLedgerMatchesSalesAndReturns(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
	}{
		{"small", smallConfig()},
		{"default", Default()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ds := mustGenerate(t, tt.cfg)
			sold := make([][]int, len(ds.Sales))
			for i, s := range ds.Sales {
				sold[i] = make([]int, len(s.Lines))
			}
			returned := make([][]int, len(ds.Returns))
			for i, ret := range ds.Returns {
				returned[i] = make([]int, len(ret.Lines))
			}
			var last time.Time
			for _, m := range ds.Movements {
				switch m.Kind {
				case "sale":
					s := ds.Sales[m.Doc]
					l := s.Lines[m.Line]
					if s.Status != "completed" || m.Branch != s.Branch || m.Item != l.Item || m.Quantity != -l.Quantity ||
						!m.MovedAt.Equal(s.SoldAt) || m.UnitCostCents != l.UnitCostCents || m.Reference != s.Folio {
						t.Fatalf("sale movement %+v does not match %s line %+v", m, s.Folio, l)
					}
					sold[m.Doc][m.Line]++
				case "return":
					ret := ds.Returns[m.Doc]
					rl := ret.Lines[m.Line]
					s := ds.Sales[ret.Sale]
					l := s.Lines[rl.SaleLine]
					if m.Branch != s.Branch || m.Item != l.Item || m.Quantity != rl.Quantity ||
						!m.MovedAt.Equal(ret.ReturnedAt) || m.UnitCostCents != l.UnitCostCents || m.Reference != ret.Folio {
						t.Fatalf("return movement %+v does not match %s line %+v", m, ret.Folio, rl)
					}
					returned[m.Doc][m.Line]++
				}
				if dateOf(m.MovedAt).Before(dateOf(last)) {
					t.Fatalf("movement %+v is on a day before the previous movement at %v", m, last)
				}
				last = m.MovedAt
			}
			for i, s := range ds.Sales {
				want := 0
				if s.Status == "completed" {
					want = 1
				}
				for j, n := range sold[i] {
					if n != want {
						t.Fatalf("%s sale %s line %d has %d sale movements, want %d", s.Status, s.Folio, j+1, n, want)
					}
				}
			}
			for i, ret := range ds.Returns {
				for j, n := range returned[i] {
					if n != 1 {
						t.Fatalf("return %s line %d has %d return movements, want 1", ret.Folio, j+1, n)
					}
				}
			}
		})
	}
}

func TestTransfersAndCounts(t *testing.T) {
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

			legs := map[string][]Movement{}
			var refs []string
			adjustments, stocked := 0, len(ds.Branches)*len(ds.Items)*cfg.Months
			for _, m := range ds.Movements {
				switch m.Kind {
				case "transfer_out", "transfer_in":
					if legs[m.Reference] == nil {
						refs = append(refs, m.Reference)
					}
					legs[m.Reference] = append(legs[m.Reference], m)
				case "adjustment":
					adjustments++
					b := ds.Branches[m.Branch]
					want := "CONTEO-" + b.Code + "-" + m.MovedAt.Format("2006-01")
					if m.Quantity != -1 || !isMonthEnd(m.MovedAt) || m.MovedAt.Hour() != 21 || m.Reference != want ||
						m.UnitCostCents != ds.Items[m.Item].UnitCostCents || m.MovedAt.Location().String() != cfg.Timezone {
						t.Fatalf("bad count adjustment %+v", m)
					}
				}
			}
			for _, ref := range refs {
				l := legs[ref]
				if len(l) != 2 || l[0].Kind != "transfer_out" || l[1].Kind != "transfer_in" {
					t.Fatalf("transfer %s has legs %+v", ref, l)
				}
				out, in := l[0], l[1]
				from, to := ds.Branches[out.Branch], ds.Branches[in.Branch]
				if out.Branch == in.Branch || from.City != to.City || out.Item != in.Item || out.Quantity >= 0 ||
					in.Quantity != -out.Quantity || out.UnitCostCents != ds.Items[out.Item].UnitCostCents ||
					in.UnitCostCents != out.UnitCostCents {
					t.Fatalf("transfer %s from %s to %s: %+v", ref, from.Code, to.Code, l)
				}
				if !dateOf(in.MovedAt).Equal(dateOf(out.MovedAt).AddDate(0, 0, 1)) ||
					out.MovedAt.Hour() != 21 || in.MovedAt.Hour() != 10 || !in.MovedAt.Before(end) {
					t.Fatalf("transfer %s leaves %v, arrives %v", ref, out.MovedAt, in.MovedAt)
				}
			}
			if tt.name == "default" && (len(refs) == 0 || adjustments == 0) {
				t.Fatalf("%d transfers, %d count adjustments; want some of each", len(refs), adjustments)
			}
			if adjustments*100 > stocked {
				t.Fatalf("%d count adjustments over %d branch-item months, want under 1%%", adjustments, stocked)
			}
		})
	}
}

func TestOpeningStock(t *testing.T) {
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
			first := time.Date(cfg.Start.Year(), cfg.Start.Month(), cfg.Start.Day(), 0, 0, 0, 0, loc)

			opened := make([][]int, len(ds.Branches))
			for b := range opened {
				opened[b] = make([]int, len(ds.Items))
			}
			units := map[string]int{}
			for i, m := range ds.Movements {
				if m.Kind != "opening" {
					continue
				}
				if i >= len(ds.Branches)*len(ds.Items) {
					t.Fatalf("opening movement %d comes after other movements", i)
				}
				if m.Quantity < 2 || !m.MovedAt.Equal(first) || m.MovedAt.Location().String() != cfg.Timezone ||
					m.UnitCostCents != ds.Items[m.Item].UnitCostCents || m.Reference != "INICIAL-"+ds.Branches[m.Branch].Code {
					t.Fatalf("bad opening movement %+v", m)
				}
				opened[m.Branch][m.Item]++
				units[ds.Branches[m.Branch].Tier] += m.Quantity
			}
			for b := range opened {
				for i, n := range opened[b] {
					if n != 1 {
						t.Fatalf("branch %d item %d has %d opening movements, want 1", b, i, n)
					}
				}
			}
			if units["large"] <= units["small"] {
				t.Fatalf("opening units: large %d, small %d", units["large"], units["small"])
			}
		})
	}
}
