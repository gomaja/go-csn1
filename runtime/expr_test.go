package runtime

import "testing"

func TestSI2quaterCellInformationWidths(t *testing.T) {
	// TS 44.018 V19.0.0 §9.1.54 table 9.1.54.1a/b gives p(n), q(m).
	p := []int{0, 10, 19, 28, 36, 44, 52, 60, 67, 74, 81, 88, 95, 102, 109, 116, 122}
	q := []int{0, 9, 17, 25, 32, 39, 46, 53, 59, 65, 71, 77, 83, 89, 95, 101, 106, 111, 116, 121, 126}
	for n := 0; n <= 31; n++ {
		for _, tc := range []struct {
			name string
			want []int
		}{{"p", p}, {"q", q}} {
			want := 0
			if n < len(tc.want) {
				want = tc.want[n]
			}
			got, err := eval(tc.name+"(n)", map[string]uint64{"n": uint64(n)})
			if err != nil || got != want {
				t.Fatalf("%s(%d) = %d, %v; want %d", tc.name, n, got, err, want)
			}
		}
	}
	for _, expression := range []string{"p(32)", "q(32)", "p(-1)", "q(-1)"} {
		if _, err := eval(expression, nil); err == nil {
			t.Fatalf("accepted out-of-range %s", expression)
		}
	}
}

func TestEvalPrintedMultiwordFieldName(t *testing.T) {
	vars := map[string]uint64{key("Length of frequency parameters"): 0, key("Length Indicator of MS ID"): 2}
	for _, tc := range []struct {
		expression string
		want       int
	}{
		{"Length of frequency parameters", 0},
		{"Lengthoffrequencyparameters", 0},
		{"val (Length of frequency parameters) + 1", 1},
		{"8*(val(Lengthoffrequencyparameters)-1)+8", 0},
		{"val (Length Indicator of MS ID) + 1", 3},
	} {
		got, err := eval(tc.expression, vars)
		if err != nil || got != tc.want {
			t.Errorf("eval(%q) = %d, %v; want %d", tc.expression, got, err, tc.want)
		}
	}
	if _, err := eval("Length of frequency parameters extra", vars); err == nil {
		t.Fatal("unknown suffix accepted")
	}
}
