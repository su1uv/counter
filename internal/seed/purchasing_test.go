package seed

import (
	"math/rand/v2"
	"slices"
	"testing"
	"time"
)

func TestSupplierFor(t *testing.T) {
	re := slices.IndexFunc(categoryDefs, func(c categoryDef) bool { return c.Code == "RE" })
	tests := []struct {
		name string
		item Item
		want string
	}{
		{"silver ring", Item{Category: 0, Material: "plata .925"}, "PR01"},
		{"10k earrings", Item{Category: 1, Material: "oro 10k"}, "PR02"},
		{"14k chain", Item{Category: 3, Material: "oro 14k"}, "PR02"},
		{"18k necklace", Item{Category: 2, Material: "oro 18k"}, "PR03"},
		{"silver watch", Item{Category: re, Material: "plata .925"}, "PR04"},
		{"gold watch", Item{Category: re, Material: "oro 18k"}, "PR04"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := supplierDefs[supplierFor(tt.item)].Code; got != tt.want {
				t.Fatalf("supplierFor = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestSubstitute(t *testing.T) {
	items := []Item{{Category: 0}, {Category: 0}, {Category: 1}, {Category: 0}}
	weights := []int{5, 5, 5, 5}
	tests := []struct {
		name   string
		onHand []int
		used   []int
		qty    int
		want   []int // any of these; empty means none
	}{
		{"same category with stock", []int{0, 3, 3, 3}, []int{0}, 1, []int{1, 3}},
		{"skips items already in the sale", []int{0, 3, 3, 3}, []int{0, 1}, 1, []int{3}},
		{"needs the whole quantity", []int{0, 1, 3, 2}, []int{0}, 2, []int{3}},
		{"other categories do not count", []int{0, 0, 3, 0}, []int{0}, 1, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := rand.New(rand.NewPCG(1, 2))
			for range 20 {
				got := substitute(r, items, weights, tt.onHand, tt.used, 0, tt.qty)
				if (tt.want == nil && got != -1) || (tt.want != nil && !slices.Contains(tt.want, got)) {
					t.Fatalf("substitute = %d, want one of %v", got, tt.want)
				}
			}
		})
	}
}

func TestGenerateLinesStock(t *testing.T) {
	items := []Item{{Category: 0, ListPriceCents: 100}, {Category: 1, ListPriceCents: 100}, {Category: 2, ListPriceCents: 100}}
	weights := []int{1, 1, 1}
	r := rand.New(rand.NewPCG(1, 2))
	for range 50 {
		if lines := generateLines(r, items, weights, []int{0, 0, 0}); lines != nil {
			t.Fatalf("lines %+v sold with no stock", lines)
		}
		lines := generateLines(r, items, weights, []int{9, 9, 9})
		if len(lines) == 0 {
			t.Fatal("no lines with plenty of stock")
		}
		for i, l := range lines {
			if l.LineNo != i+1 {
				t.Fatalf("line %d has line_no %d", i, l.LineNo)
			}
		}
	}

	// Two items of one category, one sold out: every basket ends up with the
	// other one alone, and stops once both items are used.
	pair := []Item{{Category: 0, ListPriceCents: 100}, {Category: 0, ListPriceCents: 100}}
	for range 50 {
		lines := generateLines(r, pair, []int{1, 1}, []int{0, 9})
		if len(lines) != 1 || lines[0].Item != 1 {
			t.Fatalf("lines %+v, want only item 1", lines)
		}
	}
}

// replay applies the ledger in time order, keeping generation order for
// ties, and fails if any balance goes negative. It returns the final stock.
func replay(t *testing.T, ds Dataset) [][]int {
	t.Helper()
	order := make([]int, len(ds.Movements))
	for i := range order {
		order[i] = i
	}
	slices.SortStableFunc(order, func(a, b int) int {
		return ds.Movements[a].MovedAt.Compare(ds.Movements[b].MovedAt)
	})
	stock := make([][]int, len(ds.Branches))
	for b := range stock {
		stock[b] = make([]int, len(ds.Items))
	}
	for _, i := range order {
		m := ds.Movements[i]
		stock[m.Branch][m.Item] += m.Quantity
		if stock[m.Branch][m.Item] < 0 {
			t.Fatalf("%s at branch %d goes negative after %+v", ds.Items[m.Item].SKU, m.Branch, m)
		}
	}
	return stock
}

func TestPurchasing(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
	}{
		{"small", smallConfig()},
		{"default", Default()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.cfg
			ds := mustGenerate(t, cfg)
			replay(t, ds)
			loc, _ := time.LoadLocation(cfg.Timezone)
			first := time.Date(cfg.Start.Year(), cfg.Start.Month(), cfg.Start.Day(), 0, 0, 0, 0, loc)
			end := first.AddDate(0, cfg.Months, 0)

			if len(ds.Suppliers) != len(supplierDefs) {
				t.Fatalf("%d suppliers, want %d", len(ds.Suppliers), len(supplierDefs))
			}
			for i, s := range ds.Suppliers {
				if s.Code != supplierDefs[i].Code || s.Name == "" || s.City == "" || s.LeadTimeDays < 1 {
					t.Fatalf("bad supplier %+v", s)
				}
			}

			received := make([][]int, len(ds.PurchaseOrders))
			for i, po := range ds.PurchaseOrders {
				received[i] = make([]int, len(po.Lines))
			}
			for _, m := range ds.Movements {
				if m.Kind != "receipt" {
					continue
				}
				po := ds.PurchaseOrders[m.Doc]
				l := po.Lines[m.Line]
				if po.Status != "received" || m.Branch != po.Branch || m.Item != l.Item || m.Quantity != l.Quantity ||
					!m.MovedAt.Equal(po.ReceivedAt) || m.UnitCostCents != l.UnitCostCents || m.Reference != po.Folio {
					t.Fatalf("receipt %+v does not match %s line %+v", m, po.Folio, l)
				}
				received[m.Doc][m.Line]++
			}

			folios := map[string]bool{}
			open := 0
			for i, po := range ds.PurchaseOrders {
				if folios[po.Folio] {
					t.Fatalf("duplicate purchase order folio %s", po.Folio)
				}
				folios[po.Folio] = true
				if po.Branch < 0 || po.Branch >= len(ds.Branches) || po.Supplier < 0 || po.Supplier >= len(ds.Suppliers) {
					t.Fatalf("purchase order %s has branch %d, supplier %d", po.Folio, po.Branch, po.Supplier)
				}
				if po.OrderedAt.Before(first) || !po.OrderedAt.Before(end) || po.OrderedAt.Hour() != 21 ||
					po.OrderedAt.Location().String() != cfg.Timezone {
					t.Fatalf("purchase order %s ordered at %v", po.Folio, po.OrderedAt)
				}
				lead := ds.Suppliers[po.Supplier].LeadTimeDays
				earliest := dateOf(po.OrderedAt).AddDate(0, 0, lead)
				switch po.Status {
				case "received":
					days := dateOf(po.ReceivedAt).Sub(earliest).Hours() / 24
					if days < 0 || days > maxLeadDelayDays || !po.ReceivedAt.Before(end) || po.ReceivedAt.Hour() != 10 {
						t.Fatalf("purchase order %s ordered %v received %v, lead %d", po.Folio, po.OrderedAt, po.ReceivedAt, lead)
					}
				case "ordered":
					open++
					// Still open only if it could not arrive inside the window.
					if !po.ReceivedAt.IsZero() || earliest.AddDate(0, 0, maxLeadDelayDays).Before(dateOf(end)) {
						t.Fatalf("open purchase order %s ordered %v, received %v", po.Folio, po.OrderedAt, po.ReceivedAt)
					}
				default:
					t.Fatalf("purchase order %s has status %q", po.Folio, po.Status)
				}
				if len(po.Lines) == 0 {
					t.Fatalf("purchase order %s has no lines", po.Folio)
				}
				seen := map[int]bool{}
				for j, l := range po.Lines {
					if l.LineNo != j+1 || l.Item < 0 || l.Item >= len(ds.Items) || seen[l.Item] || l.Quantity < 1 ||
						l.UnitCostCents != ds.Items[l.Item].UnitCostCents || supplierFor(ds.Items[l.Item]) != po.Supplier {
						t.Fatalf("purchase order %s line %+v", po.Folio, l)
					}
					seen[l.Item] = true
					want := 0
					if po.Status == "received" {
						want = 1
					}
					if n := received[i][j]; n != want {
						t.Fatalf("%s purchase order %s line %d has %d receipts, want %d", po.Status, po.Folio, j+1, n, want)
					}
				}
			}
			if len(ds.PurchaseOrders) == 0 || open == len(ds.PurchaseOrders) {
				t.Fatalf("%d purchase orders, %d open; want some received", len(ds.PurchaseOrders), open)
			}
		})
	}
}
