package seed

import (
	"math/rand/v2"
	"regexp"
	"testing"
	"time"
)

func TestPaymentForm(t *testing.T) {
	tests := []struct {
		name     string
		payments []Payment
		want     string
	}{
		{"cash", []Payment{{Method: "cash", AmountCents: 100}}, "01"},
		{"transfer", []Payment{{Method: "transfer", AmountCents: 100}}, "03"},
		{"card", []Payment{{Method: "card", AmountCents: 100, CardBrand: "American Express"}}, "04"},
		{"larger card", []Payment{{Method: "cash", AmountCents: 40}, {Method: "card", AmountCents: 60}}, "04"},
		{"larger cash", []Payment{{Method: "card", AmountCents: 40}, {Method: "cash", AmountCents: 60}}, "01"},
		{"tie keeps the first", []Payment{{Method: "cash", AmountCents: 50}, {Method: "transfer", AmountCents: 50}}, "01"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := paymentForm(tt.payments); got != tt.want {
				t.Fatalf("paymentForm = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestCfdiUse(t *testing.T) {
	tests := []struct{ regime, want string }{
		{"616", "S01"},
		{"601", "G01"},
		{"612", "G03"},
		{"", "G03"},
	}
	for _, tt := range tests {
		t.Run(tt.regime, func(t *testing.T) {
			if got := cfdiUse(Customer{TaxRegime: tt.regime}); got != tt.want {
				t.Fatalf("cfdiUse(%s) = %s, want %s", tt.regime, got, tt.want)
			}
		})
	}
}

func TestInvoiceUUID(t *testing.T) {
	r := rand.New(rand.NewPCG(1, pcgStream))
	re := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	seen := map[string]bool{}
	for sale := range 32 {
		id := invoiceUUID(r, sale)
		if !re.MatchString(id) || seen[id] {
			t.Fatalf("uuid %q", id)
		}
		seen[id] = true
	}
}

func TestGenerateInvoicesOnlyGeneral(t *testing.T) {
	soldAt := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	sales := []Sale{{Status: "cancelled", SoldAt: soldAt}}
	for range 40 {
		sales = append(sales, Sale{
			Status: "completed", SoldAt: soldAt,
			Payments: []Payment{{Method: "cash", AmountCents: 1000}},
		})
	}
	invoices := generateInvoices(rand.New(rand.NewPCG(1, pcgStream)), sales, []Customer{{
		RFC: "XAXX010101000", TaxRegime: "616",
	}})
	if len(invoices) == 0 {
		t.Fatal("no invoices")
	}
	for _, inv := range invoices {
		if sales[inv.Sale].Status != "completed" || inv.Customer != 0 || inv.CfdiUse != "S01" || inv.PaymentForm != "01" {
			t.Fatalf("invoice %+v for sale %+v", inv, sales[inv.Sale])
		}
	}
}

func TestGenerateInvoices(t *testing.T) {
	uuidRe := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	check := func(t *testing.T, ds Dataset, wantRate bool) {
		t.Helper()
		if len(ds.Invoices) == 0 {
			t.Fatal("no invoices")
		}
		seen := map[string]bool{}
		invoiced := map[int]bool{}
		named, general := 0, 0
		completed := 0
		for _, s := range ds.Sales {
			if s.Status == "completed" {
				completed++
			}
		}
		for _, inv := range ds.Invoices {
			sale := ds.Sales[inv.Sale]
			if sale.Status != "completed" || invoiced[inv.Sale] || !uuidRe.MatchString(inv.UUID) || seen[inv.UUID] {
				t.Fatalf("bad invoice %+v", inv)
			}
			invoiced[inv.Sale] = true
			seen[inv.UUID] = true
			if inv.PaymentMethod != "PUE" || inv.PaymentForm != paymentForm(sale.Payments) || !inv.IssuedAt.Equal(sale.SoldAt) {
				t.Fatalf("invoice %+v does not match sale %s", inv, sale.Folio)
			}
			if inv.Customer < 0 || inv.Customer >= len(ds.Customers) || inv.CfdiUse != cfdiUse(ds.Customers[inv.Customer]) {
				t.Fatalf("invoice receptor %+v", inv)
			}
			if ds.Customers[inv.Customer].RFC == "XAXX010101000" {
				general++
			} else {
				named++
			}
		}
		if named == 0 || general == 0 || len(ds.Invoices) >= completed {
			t.Fatalf("named %d, general %d, invoices %d, completed %d", named, general, len(ds.Invoices), completed)
		}
		if wantRate {
			if pct := len(ds.Invoices) * 100 / completed; pct < 18 || pct > 22 {
				t.Fatalf("%d%% of completed sales invoiced, want about %d%%", pct, invoicePercent)
			}
			if pct := named * 100 / len(ds.Invoices); pct < 15 || pct > 25 {
				t.Fatalf("%d%% of invoices name a customer, want about %d%%", pct, namedInvoicePercent)
			}
		}
	}

	t.Run("small", func(t *testing.T) { check(t, mustGenerate(t, smallConfig()), false) })
	t.Run("default", func(t *testing.T) { check(t, mustGenerate(t, Default()), true) })
}
