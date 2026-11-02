package arithaudit

import "testing"

func TestFillReadNarrowingCallShape(t *testing.T) {
	for _, call := range []string{"r.ReadUintWithFill()", "r.ReadUintWithFill(width)", "r.ReadUintWithFill(width,source,field)", "r.ReadUintWithFill(width,source,fields...)", "x.ReadUintWithFill(width,source)", "r.ReadUintWithFill(other,source)", "r.ReadUint(width,source)"} {
		source := []byte("package sample;func decode(r reader)uint8{width:=8;v,_:=" + call + ";return uint8(v)}")
		if _, err := CheckGeneratedNarrowing("sample.go", source); err == nil {
			t.Fatalf("unreviewed read accepted: %s", call)
		}
	}
}
