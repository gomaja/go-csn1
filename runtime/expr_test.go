package runtime

import "testing"

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
