package seed

import (
	"encoding/binary"
	"fmt"
	"math/rand/v2"
)

// invoicePercent is the share of completed sales that receive a CFDI.
// namedInvoicePercent is the share of those invoices issued to a named
// receptor; the rest go to Público en general.
const (
	invoicePercent      = 20
	namedInvoicePercent = 20
)

// generateInvoices attaches CFDI metadata to some completed sales. Cancelled
// sales are not invoiced. A later return does not remove the invoice.
func generateInvoices(r *rand.Rand, sales []Sale, customers []Customer) []Invoice {
	var out []Invoice
	for i, s := range sales {
		if s.Status != "completed" || r.IntN(100) >= invoicePercent {
			continue
		}
		customer := 0
		if len(customers) > 1 && r.IntN(100) < namedInvoicePercent {
			customer = 1 + r.IntN(len(customers)-1)
		}
		out = append(out, Invoice{
			Sale:          i,
			Customer:      customer,
			UUID:          invoiceUUID(r, i),
			CfdiUse:       cfdiUse(customers[customer]),
			PaymentForm:   paymentForm(s.Payments),
			PaymentMethod: "PUE",
			IssuedAt:      s.SoldAt,
		})
	}
	return out
}

// cfdiUse is the receptor's uso CFDI: S01 for público en general, G01 for a
// persona moral, G03 for a persona física.
func cfdiUse(c Customer) string {
	switch c.TaxRegime {
	case "616":
		return "S01"
	case "601":
		return "G01"
	default:
		return "G03"
	}
}

// paymentForm is the c_FormaPago of the largest tender. Equal amounts keep
// the earlier payment. Card sales use 04 (credit); the seeded brands are
// credit brands.
func paymentForm(payments []Payment) string {
	best := payments[0]
	for _, p := range payments[1:] {
		if p.AmountCents > best.AmountCents {
			best = p
		}
	}
	switch best.Method {
	case "cash":
		return "01"
	case "transfer":
		return "03"
	default:
		return "04"
	}
}

// invoiceUUID is a deterministic UUID version 4. The sale index is mixed
// into the last four bytes so two sales cannot share one.
func invoiceUUID(r *rand.Rand, sale int) string {
	var b [16]byte
	binary.LittleEndian.PutUint64(b[0:8], r.Uint64())
	binary.LittleEndian.PutUint64(b[8:16], r.Uint64())
	b[12] ^= byte(sale)
	b[13] ^= byte(sale >> 8)
	b[14] ^= byte(sale >> 16)
	b[15] ^= byte(sale >> 24)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
