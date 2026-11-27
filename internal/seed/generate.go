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
	base, mix := branchDemand(r, cfg, ds.Branches, ds.Items)
	policies := stockPolicies(base, mix)
	ds.Movements = openingStock(ds, policies, time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, loc))
	ds.Suppliers = generateSuppliers()
	newSimulation(r, cfg, loc, start, &ds, base, mix, policies).run(start)
	return ds, nil
}

// branchDemand picks each branch's base daily sales by tier, and weighs
// every item for the branch's tier.
func branchDemand(r *rand.Rand, cfg Config, branches []Branch, items []Item) (base []int, mix [][]int) {
	base = make([]int, len(branches))
	mix = make([][]int, len(branches))
	for b, branch := range branches {
		base[b] = tierBase(r, cfg, tiers[branch.Tier])
		mix[b] = itemWeights(tiers[branch.Tier], items)
	}
	return base, mix
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
// price, with an occasional percentage discount off the line. An item with
// fewer than the wanted units on hand is swapped for a substitute; with
// none, the line is lost. It returns nil if every line is lost.
func generateLines(r *rand.Rand, items []Item, weights, onHand []int) []SaleLine {
	n := min(1+weightedIndex(r, basketWeights), len(items))
	used := make([]int, 0, 2*n) // items asked for or sold in this sale
	var lines []SaleLine
	for range n {
		if len(used) == len(items) {
			break
		}
		item := weightedIndex(r, weights)
		for slices.Contains(used, item) {
			item = weightedIndex(r, weights)
		}
		used = append(used, item)
		qty := 1 + weightedIndex(r, quantityWeights)
		if onHand[item] < qty {
			if item = substitute(r, items, weights, onHand, used, item, qty); item < 0 {
				continue
			}
			used = append(used, item)
		}
		it := items[item]
		gross := int64(qty) * it.ListPriceCents
		var discount int64
		if r.IntN(100) < discountLinePercent {
			pct := discountPercents[r.IntN(len(discountPercents))]
			discount = divRoundHalfUp(gross*int64(pct), 100)
		}
		lines = append(lines, SaleLine{
			LineNo:         len(lines) + 1,
			Item:           item,
			Quantity:       qty,
			UnitPriceCents: it.ListPriceCents,
			UnitCostCents:  it.UnitCostCents,
			DiscountCents:  discount,
			TotalCents:     gross - discount,
		})
	}
	return lines
}

// substitute picks, by weight, an item of the same category as item that
// is not used in the sale yet and has qty units on hand. It returns -1 if
// there is none.
func substitute(r *rand.Rand, items []Item, weights, onHand, used []int, item, qty int) int {
	var candidates, candidateWeights []int
	for i, it := range items {
		if it.Category == items[item].Category && onHand[i] >= qty && !slices.Contains(used, i) {
			candidates = append(candidates, i)
			candidateWeights = append(candidateWeights, weights[i])
		}
	}
	if candidates == nil {
		return -1
	}
	return candidates[weightedIndex(r, candidateWeights)]
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
