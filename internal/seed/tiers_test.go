package seed

import (
	"math/rand/v2"
	"reflect"
	"testing"
	"time"
)

func TestTierDefs(t *testing.T) {
	for _, s := range branchSites {
		td, ok := tiers[s.Tier]
		if !ok {
			t.Fatalf("site %s has unknown tier %q", s.Name, s.Tier)
		}
		if td.MinStaff < 3 || td.MaxStaff > 6 || td.MinStaff > td.MaxStaff {
			t.Fatalf("tier %s staff %d-%d, want within 3-6", s.Tier, td.MinStaff, td.MaxStaff)
		}
		if td.TrafficLo < 0 || td.TrafficHi > 100 || td.TrafficLo > td.TrafficHi {
			t.Fatalf("tier %s traffic %d-%d", s.Tier, td.TrafficLo, td.TrafficHi)
		}
	}
	if tiers["large"].TrafficLo <= tiers["medium"].TrafficHi || tiers["medium"].TrafficLo <= tiers["small"].TrafficHi {
		t.Fatal("tier traffic ranges overlap")
	}
}

func TestTierBase(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	cfg := Config{MinSalesPerDay: 10, MaxSalesPerDay: 110}
	for name, td := range tiers {
		for range 100 {
			got := tierBase(r, cfg, td)
			if lo, hi := 10+td.TrafficLo, 10+td.TrafficHi; got < lo || got > hi {
				t.Fatalf("tier %s base %d outside %d-%d", name, got, lo, hi)
			}
		}
	}
}

func TestOnDuty(t *testing.T) {
	monday := day(2026, 5, 4)
	employees := []Employee{
		{Code: "manager", HiredOn: day(2020, 1, 1), DayOff: time.Sunday},
		{Code: "off monday", HiredOn: day(2020, 1, 1), DayOff: time.Monday},
		{Code: "hired today", HiredOn: monday, DayOff: time.Tuesday},
		{Code: "hired tomorrow", HiredOn: monday.AddDate(0, 0, 1), DayOff: time.Tuesday},
		{Code: "last day today", HiredOn: day(2020, 1, 1), TerminatedOn: monday, DayOff: time.Tuesday},
		{Code: "left yesterday", HiredOn: day(2020, 1, 1), TerminatedOn: monday.AddDate(0, 0, -1), DayOff: time.Tuesday},
	}
	tests := []struct {
		name  string
		staff []int
		want  []int
	}{
		{"skips day off and the unemployed", []int{0, 1, 2, 3, 4, 5}, []int{0, 2, 4}},
		{"everyone employed is off", []int{1, 3, 5}, []int{1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := onDuty(employees, tt.staff, monday); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("onDuty = %v, want %v", got, tt.want)
			}
		})
	}
}
