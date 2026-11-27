package seed

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"time"
)

// namedCustomersPerBranch is the pool of repeat buyers for each branch.
// Customer 0, shared by the whole chain, is Público en general.
const namedCustomersPerBranch = 8

// postalCodes are public codes from the cities where the chain has branches.
var postalCodes = []string{
	"06000", "11560", "04000", "03100",
	"44100", "44630",
	"66220",
	"72000",
	"76000",
}

// inconvenient is the SAT list of 4-letter keys a persona física RFC must
// not start with. A hit replaces the fourth letter with X.
var inconvenient = map[string]bool{
	"BUEI": true, "BUEY": true, "CACA": true, "CACO": true, "CAGA": true, "CAGO": true,
	"CAKA": true, "CAKO": true, "COGE": true, "COJA": true, "COJE": true, "COJI": true,
	"COJO": true, "CULO": true, "FETO": true, "GUEY": true, "JOTO": true, "KACA": true,
	"KACO": true, "KAGA": true, "KAGO": true, "KAKA": true, "KOGE": true, "KOJO": true,
	"KULO": true, "MAME": true, "MAMO": true, "MEAR": true, "MEAS": true, "MEON": true,
	"MION": true, "MOCO": true, "MULA": true, "PEDA": true, "PEDO": true, "PENE": true,
	"PUTA": true, "PUTO": true, "QULO": true, "RATA": true, "RUIN": true,
}

// generateCustomers builds the receptor list. Names and birth dates come
// from r, after the sales simulation, so the sales themselves stay put.
func generateCustomers(r *rand.Rand, branches int) []Customer {
	out := make([]Customer, 1+namedCustomersPerBranch*branches)
	out[0] = Customer{
		Code: "C0001", RFC: "XAXX010101000", LegalName: "Público en general",
		TaxRegime: "616", PostalCode: "06000",
	}
	for i := 1; i < len(out); i++ {
		if i%10 == 0 {
			out[i] = newCompany(r, i)
		} else {
			out[i] = newPerson(r, i)
		}
	}
	return out
}

func newPerson(r *rand.Rand, i int) Customer {
	first := firstNames[r.IntN(len(firstNames))]
	paternal := lastNames[r.IntN(len(lastNames))]
	maternal := lastNames[r.IntN(len(lastNames))]
	return Customer{
		Code:       fmt.Sprintf("C%04d", i+1),
		RFC:        personKey(first, paternal, maternal) + rfcDate(r) + homoclave(i),
		LegalName:  first + " " + paternal + " " + maternal,
		TaxRegime:  "612",
		PostalCode: postalCodes[r.IntN(len(postalCodes))],
	}
}

func newCompany(r *rand.Rand, i int) Customer {
	last := lastNames[r.IntN(len(lastNames))]
	return Customer{
		Code:       fmt.Sprintf("C%04d", i+1),
		RFC:        companyKey(last) + rfcDate(r) + homoclave(i),
		LegalName:  "Joyería " + last + " SA de CV",
		TaxRegime:  "601",
		PostalCode: postalCodes[r.IntN(len(postalCodes))],
	}
}

// personKey is the 4-letter start of a persona física RFC. Accents are
// dropped and Ñ is kept. A missing letter or vowel becomes X.
func personKey(first, paternal, maternal string) string {
	p := foldLetters(paternal)
	key := string([]rune{
		firstLetter(p),
		firstVowel(p),
		firstLetter(foldLetters(maternal)),
		firstLetter(foldLetters(first)),
	})
	if inconvenient[key] {
		letters := []rune(key)
		letters[3] = 'X'
		key = string(letters)
	}
	return key
}

// companyKey is the 3-letter start of a persona moral RFC for a Joyería
// named after last.
func companyKey(last string) string {
	letters := []rune(foldLetters(last))
	for len(letters) < 2 {
		letters = append(letters, 'X')
	}
	return "J" + string(letters[:2])
}

func foldLetters(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case 'Á', 'á':
			b.WriteByte('A')
		case 'É', 'é':
			b.WriteByte('E')
		case 'Í', 'í':
			b.WriteByte('I')
		case 'Ó', 'ó':
			b.WriteByte('O')
		case 'Ú', 'ú', 'Ü', 'ü':
			b.WriteByte('U')
		case 'Ñ', 'ñ':
			b.WriteRune('Ñ')
		default:
			if r >= 'a' && r <= 'z' {
				b.WriteByte(byte(r - 32))
			} else if r >= 'A' && r <= 'Z' || r == '&' {
				b.WriteRune(r)
			}
		}
	}
	return b.String()
}

func firstLetter(s string) rune {
	for _, r := range s {
		return r
	}
	return 'X'
}

// firstVowel is the first AEIOU after the initial letter.
func firstVowel(s string) rune {
	i := 0
	for _, r := range s {
		if i > 0 && strings.ContainsRune("AEIOU", r) {
			return r
		}
		i++
	}
	return 'X'
}

// homoclave encodes the customer index so two receptors cannot share an RFC.
func homoclave(n int) string {
	const digits = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	return string([]byte{digits[(n/36/36)%36], digits[(n/36)%36], digits[n%36]})
}

// rfcDate is a YYMMDD from 1955 through 2000.
func rfcDate(r *rand.Rand) string {
	year := 1955 + r.IntN(46)
	month := time.Month(1 + r.IntN(12))
	day := 1 + r.IntN(daysIn(year, month))
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC).Format("060102")
}

func daysIn(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}
