package seed

// Money is int64 centavos (MXN). Rates are basis points: 1600 is 16%.

// ivaBP is the IVA rate applied at every branch.
const ivaBP = 1600

// SplitTax splits a tax-included total into subtotal and tax. The subtotal
// is rounded half up; the tax is the remainder, so the parts always add up.
// total must not be negative.
func SplitTax(total, rateBP int64) (subtotal, tax int64) {
	subtotal = divRoundHalfUp(total*10000, 10000+rateBP)
	return subtotal, total - subtotal
}

// divRoundHalfUp divides non-negative n by positive d, rounding half up.
func divRoundHalfUp(n, d int64) int64 {
	return (2*n + d) / (2 * d)
}
