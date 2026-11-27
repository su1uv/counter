package seed

import "testing"

func TestSplitTax(t *testing.T) {
	tests := []struct {
		name         string
		total        int64
		rateBP       int64
		wantSubtotal int64
		wantTax      int64
	}{
		{"zero", 0, ivaBP, 0, 0},
		{"one centavo stays subtotal", 1, ivaBP, 1, 0},
		{"exact 16 percent", 116, ivaBP, 100, 16},
		{"exact half of 116", 58, ivaBP, 50, 8},
		{"rounds down below half", 100, ivaBP, 86, 14},  // 86.21
		{"rounds up above half", 1000, ivaBP, 862, 138}, // 862.07 -> 862
		{"rounds up at .9", 10, ivaBP, 9, 1},            // 8.62 -> 9
		{"half rounds up", 3, 10000, 2, 1},              // 1.5 at 100 percent
		{"zero rate", 999, 0, 999, 0},
		{"large amount", 4_000_000_00, ivaBP, 344_827_586, 55_172_414},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sub, tax := SplitTax(tt.total, tt.rateBP)
			if sub != tt.wantSubtotal || tax != tt.wantTax {
				t.Fatalf("SplitTax(%d, %d) = %d, %d; want %d, %d", tt.total, tt.rateBP, sub, tax, tt.wantSubtotal, tt.wantTax)
			}
			if sub+tax != tt.total {
				t.Fatalf("subtotal + tax = %d, want %d", sub+tax, tt.total)
			}
		})
	}
}

func TestDivRoundHalfUp(t *testing.T) {
	tests := []struct {
		n, d, want int64
	}{
		{0, 3, 0},
		{1, 3, 0},
		{1, 2, 1},
		{2, 3, 1},
		{5, 2, 3},
		{6, 4, 2},
		{7, 4, 2},
	}
	for _, tt := range tests {
		if got := divRoundHalfUp(tt.n, tt.d); got != tt.want {
			t.Errorf("divRoundHalfUp(%d, %d) = %d, want %d", tt.n, tt.d, got, tt.want)
		}
	}
}
