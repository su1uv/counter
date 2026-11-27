package seed

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"time"
)

// simulation walks the window day by day and fills a Dataset with sales,
// returns and their inventory movements. It carries the state between
// days: every branch's stock on hand and returns not yet back on the shelf.
type simulation struct {
	r      *rand.Rand
	loc    *time.Location
	ds     *Dataset
	end    time.Time // the day after the window, midnight UTC
	staff  [][]int   // employees of each branch
	base   []int     // base daily sales of each branch
	mix    [][]int   // item weights of each branch
	folios []int     // last sale folio of each branch
	onHand [][]int   // units of each item at each branch
	// returning holds return movements for a later day.
	returning []Movement
}

// newSimulation starts from the movements already in ds, such as opening
// stock.
func newSimulation(r *rand.Rand, cfg Config, loc *time.Location, start time.Time, ds *Dataset, base []int, mix [][]int) *simulation {
	s := &simulation{
		r: r, loc: loc, ds: ds, end: start.AddDate(0, cfg.Months, 0),
		staff: staffByBranch(ds.Branches, ds.Employees), base: base, mix: mix,
		folios: make([]int, len(ds.Branches)), onHand: make([][]int, len(ds.Branches)),
	}
	for b := range s.onHand {
		s.onHand[b] = make([]int, len(ds.Items))
	}
	for _, m := range ds.Movements {
		s.onHand[m.Branch][m.Item] += m.Quantity
	}
	return s
}

// run simulates each day: every branch sells during opening hours, then
// returns of the day go back on the shelf.
func (s *simulation) run(start time.Time) {
	for day := start; day.Before(s.end); day = day.AddDate(0, 0, 1) {
		for b := range s.ds.Branches {
			s.sell(b, day)
		}
		s.restockReturns(day)
	}
}

// apply records m in the ledger and in stock on hand.
func (s *simulation) apply(m Movement) {
	s.ds.Movements = append(s.ds.Movements, m)
	s.onHand[m.Branch][m.Item] += m.Quantity
}

// sell generates one day of sales at branch b. The daily count follows the
// branch's base rate and the calendar (see dailySales), and sale times
// follow the hourly curve. Each sale is by an employee on duty, weighted by
// productivity. Folios run per branch in time order. A few sales are
// cancelled the same day; they keep their lines but take no stock and have
// no payments. A completed sale takes its units from stock and may come
// back later as a return.
func (s *simulation) sell(b int, day time.Time) {
	r, branch := s.r, s.ds.Branches[b]
	y, m, d := day.Date()
	times := make([]int, dailySales(r, s.base[b], day))
	for i := range times {
		times[i] = saleSecond(r)
	}
	slices.Sort(times)
	for _, sec := range times {
		s.folios[b]++
		sale := Sale{
			Folio:    fmt.Sprintf("%s-%06d", branch.Code, s.folios[b]),
			Branch:   b,
			Employee: pickOnDuty(r, s.ds.Employees, s.staff[b], day),
			SoldAt:   time.Date(y, m, d, 0, 0, sec, 0, s.loc),
			Status:   "completed",
			Lines:    generateLines(r, s.ds.Items, s.mix[b]),
		}
		for _, l := range sale.Lines {
			sale.TotalCents += l.TotalCents
		}
		sale.SubtotalCents, sale.TaxCents = SplitTax(sale.TotalCents, ivaBP)
		if r.IntN(1000) < cancelPerMille {
			sale.Status = "cancelled"
		} else {
			sale.Payments = generatePayments(r, sale.TotalCents)
		}
		si := len(s.ds.Sales)
		s.ds.Sales = append(s.ds.Sales, sale)
		if sale.Status != "completed" {
			continue
		}
		for li, l := range sale.Lines {
			s.apply(Movement{
				Branch: b, Item: l.Item, MovedAt: sale.SoldAt, Kind: "sale", Quantity: -l.Quantity,
				UnitCostCents: l.UnitCostCents, Reference: sale.Folio, Doc: si, Line: li,
			})
		}
		s.scheduleReturn(si)
	}
}

// restockReturns puts the units returned on day back in stock after
// closing.
func (s *simulation) restockReturns(day time.Time) {
	later := s.returning[:0]
	for _, m := range s.returning {
		if dateOf(m.MovedAt).Equal(day) {
			s.apply(m)
		} else {
			later = append(later, m)
		}
	}
	s.returning = later
}

// staffByBranch lists each branch's employees as indexes into employees.
func staffByBranch(branches []Branch, employees []Employee) [][]int {
	staff := make([][]int, len(branches))
	for i, e := range employees {
		staff[e.Branch] = append(staff[e.Branch], i)
	}
	return staff
}

// pickOnDuty picks one of the staff on duty on day, weighted by
// productivity.
func pickOnDuty(r *rand.Rand, employees []Employee, staff []int, day time.Time) int {
	duty := onDuty(employees, staff, day)
	weights := make([]int, len(duty))
	for i, e := range duty {
		weights[i] = employees[e].Weight
	}
	return duty[weightedIndex(r, weights)]
}
