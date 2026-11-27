package seed

import (
	"math/rand/v2"
	"time"
)

// Rates in per mille: cancelled sales, and lines of completed sales that
// come back within maxReturnDays.
const (
	cancelPerMille     = 10
	returnLinePerMille = 20
	maxReturnDays      = 30
)

// generateReturns walks completed sales in order. Each line comes back with
// a small chance, so a sale has at most one return holding all of its
// returned lines. The return happens 1 to 30 days later at the same branch,
// handled by an employee on duty, and is dropped if it would fall after the
// window. Returned quantity is 1 to the quantity sold, refunded pro rata.
func generateReturns(r *rand.Rand, cfg Config, loc *time.Location, start time.Time, ds Dataset) []Return {
	staff := staffByBranch(ds.Branches, ds.Employees)
	end := start.AddDate(0, cfg.Months, 0)
	var out []Return
	for si, s := range ds.Sales {
		if s.Status != "completed" {
			continue
		}
		var lines []ReturnLine
		for li, l := range s.Lines {
			if r.IntN(1000) >= returnLinePerMille {
				continue
			}
			qty := 1 + r.IntN(l.Quantity)
			lines = append(lines, ReturnLine{
				LineNo:      len(lines) + 1,
				SaleLine:    li,
				Quantity:    qty,
				RefundCents: divRoundHalfUp(l.TotalCents*int64(qty), int64(l.Quantity)),
			})
		}
		if lines == nil {
			continue
		}
		day := dateOf(s.SoldAt).AddDate(0, 0, 1+r.IntN(maxReturnDays))
		if !day.Before(end) {
			continue
		}
		y, m, d := day.Date()
		ret := Return{
			Folio:      s.Folio + "-D",
			Sale:       si,
			Employee:   pickOnDuty(r, ds.Employees, staff[s.Branch], day),
			ReturnedAt: time.Date(y, m, d, 0, 0, saleSecond(r), 0, loc),
			Lines:      lines,
		}
		for _, l := range lines {
			ret.TotalCents += l.RefundCents
		}
		ret.SubtotalCents, ret.TaxCents = SplitTax(ret.TotalCents, ivaBP)
		out = append(out, ret)
	}
	return out
}
