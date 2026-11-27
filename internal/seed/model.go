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
	Returns    []Return
	Movements  []Movement
}

// Movement is one row of the inventory ledger, in the branch's zone.
// Quantity is signed: sales and transfers out are negative. Reference is
// the business code of the source document.
type Movement struct {
	Branch        int // index into Dataset.Branches
	Item          int // index into Dataset.Items
	MovedAt       time.Time
	Kind          string // "opening", "receipt", "sale", "return", "transfer_in", "transfer_out" or "adjustment"
	Quantity      int
	UnitCostCents int64
	Reference     string
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
	// Weight and DayOff are not stored. They decide who rings up each sale.
	Weight int
	DayOff time.Weekday
}

// Sale times are in the branch's zone. TotalCents includes tax. Cancelled
// sales have no payments.
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
	Payments      []Payment
}

// Return refunds lines of one completed sale, at the sale's branch, in the
// branch's zone. TotalCents includes tax.
type Return struct {
	Folio         string
	Sale          int // index into Dataset.Sales
	Employee      int // index into Dataset.Employees
	ReturnedAt    time.Time
	SubtotalCents int64
	TaxCents      int64
	TotalCents    int64
	Lines         []ReturnLine
}

type ReturnLine struct {
	LineNo      int
	SaleLine    int // index into the sale's Lines
	Quantity    int
	RefundCents int64
}

// Payment amounts add up to the sale total. Card payments carry a brand and
// the card's last four digits; other methods leave them empty.
type Payment struct {
	Method      string // "card", "cash" or "transfer"
	AmountCents int64
	CardBrand   string
	Last4       string
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
