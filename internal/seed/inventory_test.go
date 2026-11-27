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
