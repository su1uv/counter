package seed

import (
	"fmt"
	"slices"
	"time"
)

// Purchase orders go out after closing and arrive before opening, up to
// maxLeadDelayDays after the supplier's lead time.
const (
	orderSecond      = 21*3600 + 30*60
	receiveSecond    = 10 * 3600
	maxLeadDelayDays = 2
)

func generateSuppliers() []Supplier {
	out := make([]Supplier, len(supplierDefs))
	for i, s := range supplierDefs {
		out[i] = Supplier{Code: s.Code, Name: s.Name, City: s.City, LeadTimeDays: s.LeadTimeDays}
	}
	return out
}

// supplierFor returns the index of the supplier of it: watches come from
// the watch distributor, everything else from the maker of its material.
func supplierFor(it Item) int {
	if categoryDefs[it.Category].Code == "RE" {
		return watchSupplier
	}
	return materialSuppliers[slices.IndexFunc(materials, func(m materialDef) bool { return m.Label == it.Material })]
}

// receive puts the deliveries due on day in stock before opening.
func (s *simulation) receive(day time.Time) {
	later := s.arriving[:0]
	for _, m := range s.arriving {
		if dateOf(m.MovedAt).Equal(day) {
			s.apply(m)
			s.onOrder[m.Branch][m.Item] -= m.Quantity
		} else {
			later = append(later, m)
		}
	}
	s.arriving = later
}

// reorder runs after closing. Every item whose stock on hand plus on order
// is at or below its reorder point is ordered up to its maximum, in one
// purchase order per branch and supplier. An order that would arrive after
// the window stays open.
func (s *simulation) reorder(day time.Time) {
	y, m, d := day.Date()
	for b, branch := range s.ds.Branches {
		bySupplier := make([][]PurchaseOrderLine, len(s.ds.Suppliers))
		for i, it := range s.ds.Items {
			p, position := s.policies[b][i], s.onHand[b][i]+s.onOrder[b][i]
			if position > p.ReorderPoint {
				continue
			}
			sup := s.itemSupplier[i]
			bySupplier[sup] = append(bySupplier[sup], PurchaseOrderLine{
				LineNo: len(bySupplier[sup]) + 1, Item: i, Quantity: p.Max - position, UnitCostCents: it.UnitCostCents,
			})
			s.onOrder[b][i] += p.Max - position
		}
		for sup, lines := range bySupplier {
			if lines == nil {
				continue
			}
			s.poFolios[b]++
			po := PurchaseOrder{
				Folio:     fmt.Sprintf("OC-%s-%06d", branch.Code, s.poFolios[b]),
				Supplier:  sup,
				Branch:    b,
				OrderedAt: time.Date(y, m, d, 0, 0, orderSecond, 0, s.loc),
				Status:    "ordered",
				Lines:     lines,
			}
			arrives := day.AddDate(0, 0, s.ds.Suppliers[sup].LeadTimeDays+s.r.IntN(maxLeadDelayDays+1))
			if arrives.Before(s.end) {
				ay, am, ad := arrives.Date()
				po.Status, po.ReceivedAt = "received", time.Date(ay, am, ad, 0, 0, receiveSecond, 0, s.loc)
				for li, l := range lines {
					s.arriving = append(s.arriving, Movement{
						Branch: b, Item: l.Item, MovedAt: po.ReceivedAt, Kind: "receipt", Quantity: l.Quantity,
						UnitCostCents: l.UnitCostCents, Reference: po.Folio, Doc: len(s.ds.PurchaseOrders), Line: li,
					})
				}
			}
			s.ds.PurchaseOrders = append(s.ds.PurchaseOrders, po)
		}
	}
}
