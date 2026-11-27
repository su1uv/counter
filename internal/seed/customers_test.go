package seed

import (
	"fmt"
	"regexp"
	"testing"
	"time"
)

func TestPersonKey(t *testing.T) {
	tests := []struct {
		first, paternal, maternal, want string
	}{
		{"Ana", "López", "García", "LOGA"},
		{"José", "Pérez", "Cruz", "PECJ"},
		{"Ana", "Cano", "Cano", "CACX"},
		{"Luis", "Ng", "Díaz", "NXDL"},
		{"Ana", "De León", "Cruz", "DECA"},
		{"Ana", "Müller", "Cruz", "MUCA"},
		{"", "", "", "XXXX"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := personKey(tt.first, tt.paternal, tt.maternal); got != tt.want {
				t.Fatalf("personKey(%q, %q, %q) = %q, want %q", tt.first, tt.paternal, tt.maternal, got, tt.want)
			}
		})
	}
}

func TestCompanyKey(t *testing.T) {
	tests := []struct{ last, want string }{
		{"López", "JLO"},
		{"", "JXX"},
		{"Ñ", "JÑX"},
		{"A&", "JA&"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := companyKey(tt.last); got != tt.want {
				t.Fatalf("companyKey(%q) = %q, want %q", tt.last, got, tt.want)
			}
		})
	}
}

func TestHomoclave(t *testing.T) {
	if homoclave(0) != "000" || homoclave(1) != "001" || homoclave(36) != "010" {
		t.Fatalf("homoclave = %s %s %s", homoclave(0), homoclave(1), homoclave(36))
	}
	if homoclave(1) == homoclave(2) {
		t.Fatal("homoclave collided")
	}
}

func TestDaysIn(t *testing.T) {
	if daysIn(2000, time.February) != 29 || daysIn(1955, time.February) != 28 || daysIn(2026, time.May) != 31 {
		t.Fatalf("daysIn = %d, %d, %d", daysIn(2000, time.February), daysIn(1955, time.February), daysIn(2026, time.May))
	}
}

func TestGenerateCustomers(t *testing.T) {
	rfcRe := regexp.MustCompile(`^[A-ZÑ&]{3,4}[0-9]{6}[A-Z0-9]{3}$`)
	configs := []Config{smallConfig()}
	one := smallConfig()
	one.Branches = 1
	configs = append(configs, one)

	for _, cfg := range configs {
		ds := mustGenerate(t, cfg)
		wantN := 1 + namedCustomersPerBranch*cfg.Branches
		if len(ds.Customers) != wantN {
			t.Fatalf("customers = %d, want %d", len(ds.Customers), wantN)
		}
		general := ds.Customers[0]
		if general.Code != "C0001" || general.RFC != "XAXX010101000" || general.LegalName != "Público en general" ||
			general.TaxRegime != "616" || general.PostalCode != "06000" {
			t.Fatalf("first customer = %+v", general)
		}

		seen := map[string]bool{}
		for i, c := range ds.Customers {
			if c.Code != fmt.Sprintf("C%04d", i+1) {
				t.Fatalf("customer %d code = %s", i, c.Code)
			}
			if c.LegalName == "" || seen[c.Code] || seen[c.RFC] || !rfcRe.MatchString(c.RFC) || len(c.PostalCode) != 5 {
				t.Fatalf("bad customer %+v", c)
			}
			seen[c.Code] = true
			seen[c.RFC] = true
			date := c.RFC[len(c.RFC)-9 : len(c.RFC)-3]
			parsed, err := time.Parse("060102", date)
			if err != nil || parsed.Format("060102") != date {
				t.Fatalf("customer %s has date %s", c.RFC, date)
			}
			if i == 0 {
				continue
			}
			if i%10 == 0 {
				if c.TaxRegime != "601" || len(c.RFC) != 12 || len(c.LegalName) < len("Joyería ") || c.LegalName[:len("Joyería ")] != "Joyería " {
					t.Fatalf("company %+v", c)
				}
				continue
			}
			if c.TaxRegime != "612" || len(c.RFC) != 13 || inconvenient[c.RFC[:4]] {
				t.Fatalf("person %+v", c)
			}
		}
	}
}
