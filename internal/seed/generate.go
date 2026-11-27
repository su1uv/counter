package seed

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"time"
)

// pcgStream is the fixed second PCG word; only Config.Seed varies.
const pcgStream = 0xC0FFEE

// Generate builds the whole dataset in memory. It does no I/O and reads no
// clock, so the same Config always yields the same Dataset.
func Generate(cfg Config) (Dataset, error) {
	if err := cfg.Validate(); err != nil {
		return Dataset{}, err
	}
	r := rand.New(rand.NewPCG(cfg.Seed, pcgStream))
	start := dateOf(cfg.Start)

	ds := Dataset{Categories: generateCategories()}
	ds.Branches = generateBranches(r, cfg, start)
	ds.Items = generateItems(r, cfg.Items)
	ds.Employees = generateEmployees(r, cfg, ds.Branches, start)
	loc, _ := time.LoadLocation(cfg.Timezone) // checked by Validate
	ds.Sales = generateSales(r, cfg, loc, start, ds)
	return ds, nil
}

// generateSales walks the window day by day. Each branch's daily count
// follows its tier's base rate and the calendar (see dailySales), and sale
// times follow the hourly curve. Each sale is by an employee on duty,
// weighted by productivity. Folios run per branch in time order.
func generateSales(r *rand.Rand, cfg Config, loc *time.Location, start time.Time, ds Dataset) []Sale {
	staff := make([][]int, len(ds.Branches))
	for i, e := range ds.Employees {
		staff[e.Branch] = append(staff[e.Branch], i)
	}
	base := make([]int, len(ds.Branches))
	mix := make([][]int, len(ds.Branches))
	for b, branch := range ds.Branches {
		base[b] = tierBase(r, cfg, tiers[branch.Tier])
		mix[b] = itemWeights(tiers[branch.Tier], ds.Items)
	}
	folios := make([]int, len(ds.Branches))
	end := start.AddDate(0, cfg.Months, 0)
	var out []Sale
	for day := start; day.Before(end); day = day.AddDate(0, 0, 1) {
		y, m, d := day.Date()
		for b, branch := range ds.Branches {
			duty := onDuty(ds.Employees, staff[b], day)
			weights := make([]int, len(duty))
			for i, e := range duty {
				weights[i] = ds.Employees[e].Weight
			}
			times := make([]int, dailySales(r, base[b], day))
			for i := range times {
				times[i] = saleSecond(r)
			}
			slices.Sort(times)
			for _, sec := range times {
				folios[b]++
				sale := Sale{
					Folio:    fmt.Sprintf("%s-%06d", branch.Code, folios[b]),
					Branch:   b,
					Employee: duty[weightedIndex(r, weights)],
					SoldAt:   time.Date(y, m, d, 0, 0, sec, 0, loc),
					Status:   "completed",
					Lines:    generateLines(r, ds.Items, mix[b]),
				}
				for _, l := range sale.Lines {
					sale.TotalCents += l.TotalCents
				}
				sale.SubtotalCents, sale.TaxCents = SplitTax(sale.TotalCents, ivaBP)
				sale.Payments = generatePayments(r, sale.TotalCents)
				out = append(out, sale)
			}
		}
	}
	return out
}

// Basket shape. Index i of basketWeights is the weight of i+1 lines, and
// likewise for quantityWeights and units per line.
var (
	basketWeights    = []int{70, 22, 8}
	quantityWeights  = []int{92, 6, 2}
	discountPercents = []int{5, 10, 15, 20}
)

// discountLinePercent is the share of lines that get a discount.
const discountLinePercent = 10

// generateLines picks distinct items by the branch's item weights, at list
// price, with an occasional percentage discount off the line.
func generateLines(r *rand.Rand, items []Item, weights []int) []SaleLine {
	n := min(1+weightedIndex(r, basketWeights), len(items))
	picked := make([]int, 0, n)
	for len(picked) < n {
		if i := weightedIndex(r, weights); !slices.Contains(picked, i) {
			picked = append(picked, i)
		}
	}
	lines := make([]SaleLine, n)
	for i, item := range picked {
		it := items[item]
		qty := 1 + weightedIndex(r, quantityWeights)
		gross := int64(qty) * it.ListPriceCents
		var discount int64
		if r.IntN(100) < discountLinePercent {
			pct := discountPercents[r.IntN(len(discountPercents))]
			discount = divRoundHalfUp(gross*int64(pct), 100)
		}
		lines[i] = SaleLine{
			LineNo:         i + 1,
			Item:           item,
			Quantity:       qty,
			UnitPriceCents: it.ListPriceCents,
			UnitCostCents:  it.UnitCostCents,
			DiscountCents:  discount,
			TotalCents:     gross - discount,
		}
	}
	return lines
}

func generateCategories() []Category {
	out := make([]Category, len(categoryDefs))
	for i, c := range categoryDefs {
		out[i] = Category{Code: c.Code, Name: c.Name}
	}
	return out
}

// generateBranches opens each branch 1 to 15 years before start.
func generateBranches(r *rand.Rand, cfg Config, start time.Time) []Branch {
	out := make([]Branch, cfg.Branches)
	for i := range out {
		s := branchSites[i]
		out[i] = Branch{
			Code:     fmt.Sprintf("S%02d", i+1),
			Name:     s.Name,
			City:     s.City,
			State:    s.State,
			Timezone: cfg.Timezone,
			OpenedOn: start.AddDate(0, 0, -(365 + r.IntN(14*365))),
			Tier:     s.Tier,
		}
	}
	return out
}

// generateItems spreads items evenly over categories and picks a material,
// a price rounded down to 10 pesos, and a cost of 35-50% of the price.
func generateItems(r *rand.Rand, n int) []Item {
	out := make([]Item, n)
	seq := make([][]int, len(categoryDefs))
	for c := range seq {
		seq[c] = make([]int, len(materials))
	}
	for i := range out {
		c := i % len(categoryDefs)
		m := r.IntN(len(materials))
		seq[c][m]++
		cat, mat := categoryDefs[c], materials[m]
		lo, hi := priceRange(c, m)
		price := (lo + r.Int64N(hi-lo+1)) / 10 * 10 * 100
		out[i] = Item{
			SKU:            fmt.Sprintf("%s-%s-%04d", cat.Code, mat.Code, seq[c][m]),
			Category:       c,
			Name:           fmt.Sprintf("%s %s de %s", cat.Singular, cat.Styles[r.IntN(len(cat.Styles))], mat.Label),
			Material:       mat.Label,
			ListPriceCents: price,
			UnitCostCents:  price * (35 + r.Int64N(16)) / 100,
			Active:         true,
		}
	}
	return out
}

// priceRange returns the price range in whole pesos for a category and
// material. Both bounds are multiples of 10.
func priceRange(category, material int) (lo, hi int64) {
	if c := categoryDefs[category]; c.MaxPesos > 0 {
		return c.MinPesos, c.MaxPesos
	}
	m := materials[material]
	return m.MinPesos, m.MaxPesos
}

// generateEmployees staffs each branch by tier with one manager (the first
// employee) and sellers, hired between the branch opening and start. A
// seller may leave during the window; a replacement starts 3 to 20 days
// later if that is still inside the window. The manager never leaves.
func generateEmployees(r *rand.Rand, cfg Config, branches []Branch, start time.Time) []Employee {
	end := start.AddDate(0, cfg.Months, 0)
	window := int(end.Sub(start).Hours() / 24)
	var out []Employee
	for b, branch := range branches {
		t := tiers[branch.Tier]
		days := int(start.Sub(branch.OpenedOn).Hours() / 24)
		n := t.MinStaff + r.IntN(t.MaxStaff-t.MinStaff+1)
		var left []time.Time
		for e := range n {
			role := "seller"
			if e == 0 {
				role = "manager"
			}
			emp := newEmployee(r, branch, b, e+1, role, branch.OpenedOn.AddDate(0, 0, r.IntN(days)))
			if e > 0 && r.IntN(100) < departurePercent {
				emp.TerminatedOn = start.AddDate(0, 0, r.IntN(window))
				left = append(left, emp.TerminatedOn)
			}
			out = append(out, emp)
		}
		for _, on := range left {
			if hired := on.AddDate(0, 0, 3+r.IntN(18)); hired.Before(end) {
				n++
				out = append(out, newEmployee(r, branch, b, n, "seller", hired))
			}
		}
	}
	return out
}

// newEmployee is the seq-th employee of branch b. The day off is Sunday to
// Thursday, so nobody is off on the busy Friday and Saturday.
func newEmployee(r *rand.Rand, branch Branch, b, seq int, role string, hired time.Time) Employee {
	return Employee{
		Code:     fmt.Sprintf("%s-E%02d", branch.Code, seq),
		Branch:   b,
		FullName: fullName(r),
		Role:     role,
		HiredOn:  hired,
		Weight:   minWeight + r.IntN(maxWeight-minWeight+1),
		DayOff:   time.Weekday(r.IntN(5)),
	}
}

func fullName(r *rand.Rand) string {
	return firstNames[r.IntN(len(firstNames))] + " " +
		lastNames[r.IntN(len(lastNames))] + " " +
		lastNames[r.IntN(len(lastNames))]
}

// dateOf returns the calendar day of t as midnight UTC.
func dateOf(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}
