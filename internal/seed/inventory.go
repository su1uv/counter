package seed

import (
	"fmt"
	"time"
)

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

// After closing, monthly counts come first, then transfers leave, then
// purchase orders go out (see orderSecond). Shrinkage is the chance, per
// item a branch has, that a count comes up one unit short.
const (
	countSecond    = 21*3600 + 5*60
	transferSecond = 21*3600 + 15*60
	shrinkPerMille = 5
)

// count runs after closing on the last day of each month. Each branch
// counts every item it has, and a few come up one unit short.
func (s *simulation) count(day time.Time) {
	if !isMonthEnd(day) {
		return
	}
	y, m, d := day.Date()
	at := time.Date(y, m, d, 0, 0, countSecond, 0, s.loc)
	for b, branch := range s.ds.Branches {
		ref := fmt.Sprintf("CONTEO-%s-%04d-%02d", branch.Code, y, m)
		for i, it := range s.ds.Items {
			if s.onHand[b][i] > 0 && s.r.IntN(1000) < shrinkPerMille {
				s.apply(Movement{
					Branch: b, Item: i, MovedAt: at, Kind: "adjustment", Quantity: -1,
					UnitCostCents: it.UnitCostCents, Reference: ref,
				})
			}
		}
	}
}

// transfer runs after closing. A branch that sold out of an item borrows
// it from the branch in the same city with the most units above its own
// reorder point, up to the borrower's reorder point. The units leave that
// night and arrive before the next opening, so nothing moves on the
// window's last day.
func (s *simulation) transfer(day time.Time) {
	next := day.AddDate(0, 0, 1)
	if !next.Before(s.end) {
		return
	}
	y, m, d := day.Date()
	ny, nm, nd := next.Date()
	leaves := time.Date(y, m, d, 0, 0, transferSecond, 0, s.loc)
	arrives := time.Date(ny, nm, nd, 0, 0, receiveSecond, 0, s.loc)
	for b, branch := range s.ds.Branches {
		for i, it := range s.ds.Items {
			if s.onHand[b][i] > 0 {
				continue
			}
			donor, spare := -1, 0
			for o, other := range s.ds.Branches {
				if left := s.onHand[o][i] - s.policies[o][i].ReorderPoint - 1; o != b && other.City == branch.City && left > spare {
					donor, spare = o, left
				}
			}
			if donor < 0 {
				continue
			}
			qty := min(spare, s.policies[b][i].ReorderPoint)
			s.transfers++
			ref := fmt.Sprintf("TR-%06d", s.transfers)
			s.apply(Movement{
				Branch: donor, Item: i, MovedAt: leaves, Kind: "transfer_out", Quantity: -qty,
				UnitCostCents: it.UnitCostCents, Reference: ref,
			})
			s.arriving = append(s.arriving, Movement{
				Branch: b, Item: i, MovedAt: arrives, Kind: "transfer_in", Quantity: qty,
				UnitCostCents: it.UnitCostCents, Reference: ref,
			})
			s.onOrder[b][i] += qty
		}
	}
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
