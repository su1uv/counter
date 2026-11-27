package seed

import (
	"fmt"
	"math/rand/v2"
)

// Payment mix. Weights are in percent and follow the order of the names.
var (
	paymentMethods = []string{"card", "cash", "transfer"}
	methodWeights  = []int{60, 30, 10}
	cardBrands     = []string{"Visa", "Mastercard", "American Express"}
	brandWeights   = []int{55, 35, 10}
)

// splitPercent is the share of sales paid with two methods.
const splitPercent = 5

// generatePayments pays a positive total with one method, or sometimes with
// two different methods. The first part of a split is 20-80% of the total in
// whole pesos, so splits need at least 2 pesos.
func generatePayments(r *rand.Rand, total int64) []Payment {
	first := newPayment(r, weightedIndex(r, methodWeights))
	if total < 200 || r.IntN(100) >= splitPercent {
		first.AmountCents = total
		return []Payment{first}
	}
	m := weightedIndex(r, methodWeights)
	for paymentMethods[m] == first.Method {
		m = weightedIndex(r, methodWeights)
	}
	second := newPayment(r, m)
	first.AmountCents = max(100, total/100*int64(20+r.IntN(61))/100*100)
	second.AmountCents = total - first.AmountCents
	return []Payment{first, second}
}

func newPayment(r *rand.Rand, method int) Payment {
	p := Payment{Method: paymentMethods[method]}
	if p.Method == "card" {
		p.CardBrand = cardBrands[weightedIndex(r, brandWeights)]
		p.Last4 = fmt.Sprintf("%04d", r.IntN(10000))
	}
	return p
}
