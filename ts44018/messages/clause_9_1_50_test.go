package messages

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44018/restoctets"
)

// TS 44.018 V19.0.0 §§9.1.50–9.1.50b, DOCX paragraphs 10632–10665;
// assigned message types are in table 10.4.2, paragraphs 13159–13224.
func checkSI10FixedCarrier(t *testing.T, clause, name string, value any, wire []byte) {
	t.Helper()
	d, err := LookupClause(clause, name)
	if err != nil {
		t.Fatal(err)
	}
	canonical, err := d.Canonical(value)
	if err != nil || !bytes.Equal(canonical, wire) {
		t.Fatalf("fresh fixed carrier %x want %x: %v", canonical, wire, err)
	}
	decoded, err := d.Decode(wire)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := d.Encode(decoded)
	if err != nil || !bytes.Equal(replay, wire) {
		t.Fatalf("replay %x: %v", replay, err)
	}
	got := reflect.ValueOf(decoded).FieldByName("Value").Interface()
	if !runtime.SemanticallyEqual(got, value) {
		t.Fatalf("typed value %#v want %#v", got, value)
	}
	stable, err := d.CanonicalAtLength(decoded, 21)
	if err != nil || !bytes.Equal(stable, canonical) {
		t.Fatalf("unstable canonical %x: %v", stable, err)
	}
	for _, n := range []int{0, 1, 20, 22} {
		if _, err := d.CanonicalAtLength(value, n); err == nil {
			t.Errorf("accepted canonical extent %d", n)
		}
	}
	for cut := 0; cut < 168; cut++ {
		r := runtime.NewReader(wire)
		if _, err := r.PushLimit(cut); err != nil {
			t.Fatal(err)
		}
		if _, err := d.DecodeFrom(r); err == nil {
			t.Errorf("accepted %d-bit truncated carrier", cut)
		}
		if r.Position() > cut {
			t.Errorf("crossed %d-bit reader limit", cut)
		}
	}
	r := runtime.NewReader(append(append([]byte{}, wire...), 0xa5))
	old, err := r.PushLimit(168)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.DecodeFrom(r); err != nil {
		t.Fatal(err)
	}
	if r.Position() != 168 {
		t.Fatalf("carrier consumed %d bits", r.Position())
	}
	if err := r.PopLimit(old); err != nil {
		t.Fatal(err)
	}
	sentinel, err := r.ReadUint(8)
	if err != nil || sentinel != 0xa5 {
		t.Fatalf("caller suffix %x: %v", sentinel, err)
	}
	for typ := 0; typ < 32; typ++ {
		if byte(typ) == wire[0]>>2 {
			continue
		}
		bad := append([]byte{}, wire...)
		bad[0] = (bad[0] & 0x83) | byte(typ<<2)
		if _, err := d.Decode(bad); err == nil {
			t.Errorf("accepted message type %d", typ)
		}
	}
	for _, field := range []struct {
		name string
		bad  uint64
	}{{"RRShortPD", 2}, {"ShortLayer2Header", 4}, {"MessageType", 32}} {
		bad := reflect.New(reflect.TypeOf(value)).Elem()
		bad.Set(reflect.ValueOf(value))
		bad.FieldByName(field.name).SetUint(field.bad)
		if _, err := d.Canonical(bad.Interface()); err == nil {
			t.Errorf("accepted out-of-range %s", field.name)
		}
	}
}

func TestSI10FixedCanonicalCarrier(t *testing.T) {
	value := SYSTEMINFORMATIONTYPE10{SI10RestOctets: restoctets.SI10RestOctets{NeighbourInformationChoice: restoctets.SI10RestOctetsNeighbourInformationChoice{AltL: &restoctets.SI10RestOctetsNeighbourInformationChoiceAltL{}}}}
	wire := append([]byte{0}, bytes.Repeat([]byte{0x2b}, 20)...)
	checkSI10FixedCarrier(t, "9.1.50", "SYSTEM INFORMATION TYPE 10", value, wire)
	value.SI10RestOctets.BAInd = 1
	wire[1] |= 0x80
	checkSI10FixedCarrier(t, "9.1.50", "SYSTEM INFORMATION TYPE 10", value, wire)
	value.SI10RestOctets.NeighbourInformationChoice.Alternative = 255
	if _, err := EncodeSYSTEMINFORMATIONTYPE10Canonical(value); err == nil {
		t.Fatal("invalid rest-octet selector accepted")
	}
}

func FuzzSI10FixedCarrierReplay(f *testing.F) {
	for _, typ := range []byte{0, 10, 11} {
		f.Add(typ, append([]byte{typ << 2}, bytes.Repeat([]byte{0x2b}, 20)...))
	}
	roots := map[byte][2]string{0: {"9.1.50", "SYSTEM INFORMATION TYPE 10"}, 10: {"9.1.50a", "SYSTEM INFORMATION TYPE 10bis"}, 11: {"9.1.50b", "SYSTEM INFORMATION TYPE 10ter"}}
	f.Fuzz(func(t *testing.T, typ byte, wire []byte) {
		if len(wire) > 64 {
			return
		}
		root, ok := roots[typ]
		if !ok {
			return
		}
		d, err := LookupClause(root[0], root[1])
		if err != nil {
			t.Fatal(err)
		}
		decoded, err := d.Decode(wire)
		if err != nil {
			return
		}
		replay, err := d.Encode(decoded)
		if err != nil || !bytes.Equal(replay, wire) {
			t.Fatalf("replay %x: %v", replay, err)
		}
	})
}
