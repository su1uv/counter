package seed

import (
	"fmt"
	"math/rand/v2"
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
	return ds, nil
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

// generateEmployees gives every branch one manager (the first employee)
// and sellers, all hired between the branch opening and start.
func generateEmployees(r *rand.Rand, cfg Config, branches []Branch, start time.Time) []Employee {
	out := make([]Employee, 0, len(branches)*cfg.EmployeesPerBranch)
	for b, branch := range branches {
		days := int(start.Sub(branch.OpenedOn).Hours() / 24)
		for e := range cfg.EmployeesPerBranch {
			role := "seller"
			if e == 0 {
				role = "manager"
			}
			out = append(out, Employee{
				Code:     fmt.Sprintf("%s-E%02d", branch.Code, e+1),
				Branch:   b,
				FullName: fullName(r),
				Role:     role,
				HiredOn:  branch.OpenedOn.AddDate(0, 0, r.IntN(days)),
			})
		}
	}
	return out
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
