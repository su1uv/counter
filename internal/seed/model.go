package seed

import "time"

// Dataset is everything Generate produces. Rows refer to each other by slice
// index; Write maps indexes to database ids. Dates (OpenedOn, HiredOn) are
// midnight UTC and carry only the calendar day.
type Dataset struct {
	Branches       []Branch
	Categories     []Category
	Items          []Item
	Employees      []Employee
	Registers      []Register
	Shifts         []Shift
	Sales          []Sale
	Returns        []Return
	Suppliers      []Supplier
	PurchaseOrders []PurchaseOrder
	Movements      []Movement
	CashMovements  []CashMovement
}

type Supplier struct {
	Code         string
	Name         string
	City         string
	LeadTimeDays int
}

// PurchaseOrder is placed by a branch after closing and delivered straight
// to it, in full, a few days later. Times are in the branch's zone.
type PurchaseOrder struct {
	Folio      string
	Supplier   int // index into Dataset.Suppliers
	Branch     int // index into Dataset.Branches
	OrderedAt  time.Time
	ReceivedAt time.Time // zero while still open
	Status     string    // "ordered" or "received"
	Lines      []PurchaseOrderLine
}

type PurchaseOrderLine struct {
	LineNo        int
	Item          int // index into Dataset.Items
	Quantity      int
	UnitCostCents int64
}

// Movement is one row of the inventory ledger, in the branch's zone.
// Quantity is signed: sales and transfers out are negative. Reference is
// the business code of the source document. Doc and Line locate the source
// line of sale movements (Dataset.Sales), return movements
// (Dataset.Returns) and receipts (Dataset.PurchaseOrders); other kinds
// leave them zero.
type Movement struct {
	Branch        int // index into Dataset.Branches
	Item          int // index into Dataset.Items
	MovedAt       time.Time
	Kind          string // "opening", "receipt", "sale", "return", "transfer_in", "transfer_out" or "adjustment"
	Quantity      int
	UnitCostCents int64
	Reference     string
	Doc, Line     int
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

// Register is one cash register at a branch.
type Register struct {
	Code   string
	Branch int // index into Dataset.Branches
	Name   string
}

// Shift is one opening of a register, in the branch's zone. Expected cash
// is the drawer balance the count should match. Counted cash is that
// balance plus a small over or short; both can be negative when cash
// refunds exceed the drawer.
type Shift struct {
	Register          int // index into Dataset.Registers
	OpenedBy          int // index into Dataset.Employees
	OpenedAt          time.Time
	ClosedAt          time.Time
	OpeningFloatCents int64
	ExpectedCashCents int64
	CountedCashCents  int64
}

// CashMovement is cash that left the drawer during a shift: a withdrawal
// to the safe, or a petty expense. AmountCents is positive.
type CashMovement struct {
	Shift       int // index into Dataset.Shifts
	MovedAt     time.Time
	Kind        string // "withdrawal" or "expense"
	AmountCents int64
	Note        string
}

// Sale times are in the branch's zone. TotalCents includes tax. Cancelled
// sales have no payments.
type Sale struct {
	Folio         string
	Branch        int // index into Dataset.Branches
	Employee      int // index into Dataset.Employees
	Shift         int // index into Dataset.Shifts
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
