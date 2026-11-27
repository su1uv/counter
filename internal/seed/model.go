package seed

import "time"

// Dataset is everything Generate produces. Rows refer to each other by slice
// index; Write maps indexes to database ids. Dates (OpenedOn, HiredOn) are
// midnight UTC and carry only the calendar day.
type Dataset struct {
	Branches   []Branch
	Categories []Category
	Items      []Item
	Employees  []Employee
	Sales      []Sale
}

type Branch struct {
	Code     string
	Name     string
	City     string
	State    string
	Timezone string
	OpenedOn time.Time
	Tier     string // "large", "medium" or "small"
}

type Category struct {
	Code string
	Name string
}

type Item struct {
	SKU            string
	Category       int // index into Dataset.Categories
	Name           string
	Material       string
	UnitCostCents  int64
	ListPriceCents int64
	Active         bool
}

type Employee struct {
	Code     string
	Branch   int // index into Dataset.Branches
	FullName string
	Role     string // "seller" or "manager"
	HiredOn  time.Time
	// TerminatedOn is the last day worked; zero while still employed.
	TerminatedOn time.Time
}

// Sale times are in the branch's zone. TotalCents includes tax.
type Sale struct {
	Folio         string
	Branch        int // index into Dataset.Branches
	Employee      int // index into Dataset.Employees
	SoldAt        time.Time
	Status        string // "completed" or "cancelled"
	SubtotalCents int64
	TaxCents      int64
	TotalCents    int64
	Lines         []SaleLine
}

// SaleLine copies the item's price and cost at the moment of sale.
type SaleLine struct {
	LineNo         int
	Item           int // index into Dataset.Items
	Quantity       int
	UnitPriceCents int64
	UnitCostCents  int64
	DiscountCents  int64
	TotalCents     int64
}
