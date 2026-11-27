package seed

import "time"

// Stock policy, in days of a branch's expected demand for an item: reorder
// when stock falls to reorderDays, and order up to coverDays.
const (
	reorderDays = 12
	coverDays   = 30
)

// stockPolicy says when a branch reorders an item and up to what level.
// Every item keeps at least one unit to sell and one to spare.
type stockPolicy struct {
	ReorderPoint int
	Max          int
}

// unitsPerSaleMilli is the expected units per sale, in thousandths, from
// the basket and quantity weights.
func unitsPerSaleMilli() int64 {
	return meanPerMille(basketWeights) * meanPerMille(quantityWeights) / 1000
}

// meanPerMille is the mean of 1..len(weights) under weights, in
// thousandths.
func meanPerMille(weights []int) int64 {
	var sum, total int64
	for i, w := range weights {
		sum += int64(i+1) * int64(w)
		total += int64(w)
	}
	return 1000 * sum / total
}

// stockPolicies sizes each branch's stock of each item from its expected
// daily units: the branch's base daily sales, times units per sale, times
// the item's share of the branch's mix.
func stockPolicies(base []int, mix [][]int) [][]stockPolicy {
	perSale := unitsPerSaleMilli()
	out := make([][]stockPolicy, len(base))
	for b := range base {
		var total int64
		for _, w := range mix[b] {
			total += int64(w)
		}
		out[b] = make([]stockPolicy, len(mix[b]))
		for i, w := range mix[b] {
			dailyMilli := int64(base[b]) * perSale * int64(w) / total
			rop := max(1, ceilDiv(dailyMilli*reorderDays, 1000))
			out[b][i] = stockPolicy{ReorderPoint: rop, Max: max(rop+1, ceilDiv(dailyMilli*coverDays, 1000))}
		}
	}
	return out
}

func ceilDiv(n, d int64) int {
	return int((n + d - 1) / d)
}

// openingStock fills every branch with every item up to its policy's Max at
// midnight on the first day.
func openingStock(ds Dataset, policies [][]stockPolicy, first time.Time) []Movement {
	var out []Movement
	for b, branch := range ds.Branches {
		for i, it := range ds.Items {
			out = append(out, Movement{
				Branch: b, Item: i, MovedAt: first, Kind: "opening", Quantity: policies[b][i].Max,
				UnitCostCents: it.UnitCostCents, Reference: "INICIAL-" + branch.Code,
			})
		}
	}
	return out
}
