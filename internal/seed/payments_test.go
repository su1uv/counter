package seed

import (
	"math/rand/v2"
	"regexp"
	"testing"
)

var last4Pattern = regexp.MustCompile(`^[0-9]{4}$`)

func checkPayments(t *testing.T, total int64, ps []Payment) {
	t.Helper()
	if len(ps) < 1 || len(ps) > 2 {
		t.Fatalf("total %d has %d payments", total, len(ps))
	}
	var sum int64
	for _, p := range ps {
		if p.AmountCents <= 0 {
			t.Fatalf("payment %+v has a non-positive amount", p)
		}
		sum += p.AmountCents
		switch p.Method {
		case "card":
			if p.CardBrand == "" || !last4Pattern.MatchString(p.Last4) {
				t.Fatalf("card payment %+v lacks brand or last4", p)
			}
		case "cash", "transfer":
			if p.CardBrand != "" || p.Last4 != "" {
				t.Fatalf("%s payment %+v has card fields", p.Method, p)
			}
		default:
			t.Fatalf("payment has method %q", p.Method)
		}
	}
	if len(ps) == 2 && ps[0].Method == ps[1].Method {
		t.Fatalf("split payment uses %s twice", ps[0].Method)
	}
	if sum != total {
		t.Fatalf("payments add to %d, want %d", sum, total)
	}
}

func TestGeneratePayments(t *testing.T) {
	tests := []struct {
		name      string
		total     int64
		wantSplit bool
	}{
		{"one centavo never splits", 1, false},
		{"199 centavos never split", 199, false},
		{"two pesos can split", 200, true},
		{"large total", 4_567_890, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := rand.New(rand.NewPCG(1, 2))
			split := false
			for range 2000 {
				ps := generatePayments(r, tt.total)
				checkPayments(t, tt.total, ps)
				split = split || len(ps) == 2
			}
			if split != tt.wantSplit {
				t.Fatalf("split seen = %v, want %v", split, tt.wantSplit)
			}
		})
	}
}

func TestPaymentMix(t *testing.T) {
	ds := mustGenerate(t, Default())
	methods := map[string]int{}
	splits := 0
	for _, s := range ds.Sales {
		checkPayments(t, s.TotalCents, s.Payments)
		if len(s.Payments) == 2 {
			splits++
		}
		methods[s.Payments[0].Method]++
	}
	n := len(ds.Sales)
	within := func(got, want int) bool { return got*100 >= (want-3)*n && got*100 <= (want+3)*n }
	if !within(methods["card"], 60) || !within(methods["cash"], 30) || !within(methods["transfer"], 10) {
		t.Fatalf("first payment methods %v over %d sales, want about 60/30/10%%", methods, n)
	}
	if splits == 0 || splits*100 > 8*n {
		t.Fatalf("%d split payments over %d sales, want a few", splits, n)
	}
}
