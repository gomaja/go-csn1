package messages

import (
	"bytes"
	"encoding/hex"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

type ecSourceCase struct {
	name, wire  string
	contentBits int
}

func checkECSourceCases(t *testing.T, clause, name string, messageType uint8, cases []ecSourceCase) {
	t.Helper()
	d, err := LookupClause(clause, name)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			wire, err := hex.DecodeString(c.wire)
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := d.Decode(wire)
			if err != nil {
				t.Fatal(err)
			}
			checkECTypedFields(t, d, decoded, wire)
			replay, err := d.Encode(decoded)
			if err != nil || !bytes.Equal(replay, wire) {
				t.Fatalf("replay %x: %v", replay, err)
			}
			canonical, err := d.CanonicalAtLength(decoded, len(wire))
			if err != nil || !bytes.Equal(canonical, wire) {
				t.Fatalf("canonical %x want %x: %v", canonical, wire, err)
			}
			fresh, err := d.Decode(canonical)
			if err != nil {
				t.Fatal(err)
			}
			stable, err := d.CanonicalAtLength(fresh, len(wire))
			if err != nil || !bytes.Equal(stable, canonical) {
				t.Fatalf("unstable canonical %x: %v", stable, err)
			}
			if _, err := d.Canonical(decoded); err == nil {
				t.Fatal("canonical padding encoded without an extent")
			}
			if _, err := d.CanonicalAtLength(decoded, (c.contentBits-1)/8); err == nil {
				t.Fatal("content encoded into an insufficient extent")
			}
			// TS 24.007 V20.0.0 Annex B.1.2.1/B.1.2.2: sender L padding,
			// receiver spare bits. Changing only the received padding must preserve replay.
			received := append([]byte(nil), wire...)
			for bit := c.contentBits; bit < len(received)*8; bit++ {
				received[bit/8] |= 1 << uint(7-bit%8)
			}
			other, err := d.Decode(received)
			if err != nil {
				t.Fatal(err)
			}
			replay, err = d.Encode(other)
			if err != nil || !bytes.Equal(replay, received) {
				t.Fatalf("received padding replay %x: %v", replay, err)
			}
			canonical, err = d.CanonicalAtLength(other, len(wire))
			if err != nil || !bytes.Equal(canonical, wire) {
				t.Fatalf("received padding canonical %x: %v", canonical, err)
			}
			for bits := 0; bits < c.contentBits; bits++ {
				reader := runtime.NewReader(wire)
				if _, err := reader.PushLimit(bits); err != nil {
					t.Fatal(err)
				}
				if _, err := d.DecodeFrom(reader); err == nil {
					t.Fatalf("accepted truncated content at bit %d of %d", bits, c.contentBits)
				}
				if reader.Position() > bits {
					t.Fatalf("truncated reader crossed its bound at %d: %d", bits, reader.Position())
				}
			}
			reader := runtime.NewReader(append(append([]byte(nil), wire...), 0xd7))
			old, err := reader.PushLimit(len(wire) * 8)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := d.DecodeFrom(reader); err != nil {
				t.Fatal(err)
			}
			if reader.Position() != len(wire)*8 {
				t.Fatalf("bounded reader position %d", reader.Position())
			}
			if err := reader.PopLimit(old); err != nil {
				t.Fatal(err)
			}
			tail, err := reader.ReadUint(8)
			if err != nil || tail != 0xd7 {
				t.Fatalf("following octet %x: %v", tail, err)
			}
			for candidate := uint8(0); candidate < 8; candidate++ {
				if candidate == messageType {
					continue
				}
				wrong := append([]byte(nil), wire...)
				wrong[0] = (wrong[0] & 0x1f) | (candidate << 5)
				if _, err := d.Decode(wrong); err == nil {
					t.Fatalf("accepted message type %d", candidate)
				}
				if _, err := d.CanonicalAtLength(ecEditedValue(t, decoded, "MessageType", uint64(candidate)), len(wire)); err == nil {
					t.Fatalf("encoded message type %d", candidate)
				}
			}
			for field, invalid := range map[string]uint64{"ECSI1INDEX": 4, "ECSI1COUNT": 4, "ECSI3INDEX": 4, "ECSI3COUNT": 4, "ECSI4INDEX": 4, "ECSI4COUNT": 4, "ECSICHANGEMARK": 32} {
				value := reflect.ValueOf(decoded)
				if !value.FieldByName("Value").FieldByName(field).IsValid() {
					continue
				}
				if _, err := d.CanonicalAtLength(ecEditedValue(t, decoded, field, invalid), len(wire)); err == nil {
					t.Fatalf("encoded invalid %s=%d", field, invalid)
				}
			}
		})
	}
}

func ecEditedValue(t *testing.T, decoded any, name string, number uint64) any {
	t.Helper()
	canonical := reflect.ValueOf(runtime.Canonical(decoded))
	copyValue := reflect.New(canonical.Type()).Elem()
	copyValue.Set(canonical)
	field := copyValue.FieldByName("Value").FieldByName(name)
	if !field.IsValid() || !field.CanSet() {
		t.Fatalf("missing editable field %s", name)
	}
	field.SetUint(number)
	return copyValue.Interface()
}

func fuzzECSource(f *testing.F, clause, name string, cases []ecSourceCase) {
	for _, c := range cases {
		wire, err := hex.DecodeString(c.wire)
		if err != nil {
			f.Fatal(err)
		}
		f.Add(wire)
	}
	d, err := LookupClause(clause, name)
	if err != nil {
		f.Fatal(err)
	}
	f.Fuzz(func(t *testing.T, wire []byte) {
		if len(wire) > 512 {
			return
		}
		value, err := d.Decode(wire)
		if err != nil {
			return
		}
		replay, err := d.Encode(value)
		if err != nil || !bytes.Equal(replay, wire) {
			t.Fatalf("accepted bytes did not replay: %x: %v", replay, err)
		}
		canonical, err := d.CanonicalAtLength(value, len(wire))
		if err != nil {
			t.Fatal(err)
		}
		fresh, err := d.Decode(canonical)
		if err != nil {
			t.Fatal(err)
		}
		stable, err := d.CanonicalAtLength(fresh, len(wire))
		if err != nil || !bytes.Equal(stable, canonical) {
			t.Fatalf("canonical stability %x: %v", stable, err)
		}
	})
}

// TS 44.018 V19.0.0 §9.1.43p; all values are synthetic.
var ecSystemInformationType1Cases = []ecSourceCase{
	{"empty", "37132b2b", 15},
	{"frequency-0", "37182a9b2b2b", 28},
	{"frequency-31", "371feaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa9b2b2b", 276},
	{"allocation-0-single-0", "3717e7feab2b2b", 33},
	{"allocation-0-previous-0", "3717e92b2b", 24},
	{"allocation-0-bitmap-0", "3717ec052b2b", 32},
	{"allocation-0-bitmap-63", "3717edfffffffffffffffffb2b2b", 95},
	{"allocation-0-hopping-0", "3717eff8172b2b", 38},
	{"allocation-0-hopping-63", "3717efffffffffffffffffffeb2b2b", 101},
	{"allocation-1-single-0", "3717ffdffb2b2b", 39},
	{"allocation-1-previous-0", "3717ffe72b2b", 30},
	{"allocation-1-bitmap-0", "3717fff0172b2b", 38},
	{"allocation-1-bitmap-63", "3717fff7ffffffffffffffffeb2b2b", 101},
	{"allocation-1-hopping-0", "3717ffffe05b2b2b", 44},
	{"allocation-1-hopping-63", "3717ffffffffffffffffffffffab2b2b", 107},
}

func TestECSystemInformationType1Source(t *testing.T) {
	checkECSourceCases(t, "9.1.43p", "EC System Information Type 1", 1, ecSystemInformationType1Cases)
}
func FuzzECSystemInformationType1(f *testing.F) {
	fuzzECSource(f, "9.1.43p", "EC System Information Type 1", ecSystemInformationType1Cases)
}

// checkECTypedFields checks source values independently of replay and checks every
// representable out-of-range scalar in each populated branch.
func checkECTypedFields(t *testing.T, d runtime.Descriptor, decoded any, wire []byte) {
	t.Helper()
	root := reflect.ValueOf(decoded).FieldByName("Value")
	for name, want := range map[string]uint64{"MessageType": uint64(wire[0] >> 5), "ECSI1INDEX": 2, "ECSI1COUNT": 3, "ECSI3INDEX": 2, "ECSI3COUNT": 3, "ECSI4INDEX": 2, "ECSI4COUNT": 3, "ECSICHANGEMARK": 17} {
		field := root.FieldByName(name)
		if field.IsValid() && field.Uint() != want {
			t.Fatalf("%s=%d want %d", name, field.Uint(), want)
		}
	}
	widths := map[string]uint{"ECSI1INDEX": 2, "ECSI1COUNT": 2, "ECSI3INDEX": 2, "ECSI3COUNT": 2, "ECSI4INDEX": 2, "ECSI4COUNT": 2, "ECSICHANGEMARK": 5, "NumberOfOctets": 5, "ECMANUMBER": 5, "MAIO": 6, "ARFCN": 10, "HSN": 6, "MALENGTH": 6, "Alternative": 1, "BandIndicator": 1, "CELLRESELECTHYSTERESIS": 3, "CELLRESELECTOFFSET": 6, "C1DELTAMIN": 2, "C1DELTAMAX": 3, "NbNCELL": 5, "CELLTYPE": 1, "CELLBARACCESS": 1, "SAMERAASSERVINGCELL": 1, "ECRXLEVACCESSMIN": 6, "MSTXPWRMAXCCH": 5, "CEAUTHOFFSET": 2, "ECSI4Indicator": 1, "CommonPLMNAllowed": 1, "NbAdditionalPLMNs": 2, "MCC": 12, "MNC": 12, "ECAccessControlClass": 7, "ExceptionReportStatus": 1}
	type step struct {
		field string
		index int
	}
	var visit func(reflect.Value, []step, string)
	visit = func(value reflect.Value, steps []step, name string) {
		for value.Kind() == reflect.Pointer {
			if value.IsNil() {
				return
			}
			value = value.Elem()
		}
		switch value.Kind() {
		case reflect.Struct:
			if value.Type() == reflect.TypeFor[runtime.BitString]() {
				return
			}
			for i := 0; i < value.NumField(); i++ {
				field := value.Type().Field(i)
				if field.Name != "Wire" {
					visit(value.Field(i), append(append([]step(nil), steps...), step{field: field.Name}), field.Name)
				}
			}
		case reflect.Slice:
			for i := 0; i < value.Len(); i++ {
				visit(value.Index(i), append(append([]step(nil), steps...), step{index: i}), name)
			}
		case reflect.Uint8, reflect.Uint16:
			width, ok := widths[name]
			if name == "BSIC" {
				width = 6
				if value.Kind() == reflect.Uint16 {
					width = 9
				}
				ok = true
			}
			if !ok {
				return
			}
			fresh, err := d.Decode(wire)
			if err != nil {
				t.Fatal(err)
			}
			canonical := reflect.ValueOf(runtime.Canonical(fresh))
			mutable := reflect.New(canonical.Type()).Elem()
			mutable.Set(canonical)
			field := mutable.FieldByName("Value")
			for _, s := range steps {
				for field.Kind() == reflect.Pointer {
					field = field.Elem()
				}
				if s.field != "" {
					field = field.FieldByName(s.field)
				} else {
					field = field.Index(s.index)
				}
			}
			for field.Kind() == reflect.Pointer {
				field = field.Elem()
			}
			field.SetUint(1 << width)
			if _, err := d.CanonicalAtLength(mutable.Interface(), len(wire)); err == nil {
				t.Fatalf("encoded out-of-range %s at %v", name, steps)
			}
		}
	}
	visit(root, nil, "")
}

func TestECSystemInformationType1TypedAllocation(t *testing.T) {
	// TS 44.018 V19.0.0 §9.1.43p paragraphs 9749–9770.
	for _, c := range ecSystemInformationType1Cases {
		t.Run(c.name, func(t *testing.T) {
			wire, err := hex.DecodeString(c.wire)
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := DecodeECSystemInformationType1(wire)
			if err != nil {
				t.Fatal(err)
			}
			v := decoded.Value
			if v.BandIndicator != 1 {
				t.Fatal("band indicator lost")
			}
			cell := v.ECCellChannelDescription
			if strings.HasPrefix(c.name, "frequency-") {
				n, err := strconv.Atoi(strings.TrimPrefix(c.name, "frequency-"))
				if err != nil {
					t.Fatal(err)
				}
				if cell == nil || int(cell.NumberOfOctets) != n || cell.FrequencyListInformation.BitLength != 8*(n+1) || !bytes.Equal(cell.FrequencyListInformation.Bytes, bytes.Repeat([]byte{0xaa}, n+1)) {
					t.Fatalf("cell channel description: %+v", cell)
				}
				cell.FrequencyListInformation.BitLength--
				if _, err := EncodeECSystemInformationType1CanonicalAtLength(v, len(wire)); err == nil {
					t.Fatal("invalid frequency-list width encoded")
				}
			} else if cell != nil {
				t.Fatal("unexpected cell channel description")
			}
			allocations := v.ECMobileAllocationList.ECMobileAllocationStructList
			if !strings.HasPrefix(c.name, "allocation-") {
				if len(allocations) != 0 {
					t.Fatal("unexpected allocations")
				}
				return
			}
			if len(allocations) != 1 {
				t.Fatalf("allocation count %d", len(allocations))
			}
			a := allocations[0].ECMobileAllocationStruct
			if a.ECMANUMBER != 31 {
				t.Fatal("MA number lost")
			}
			hasMAIO := strings.HasPrefix(c.name, "allocation-1-")
			if (a.MAIO != nil) != hasMAIO || (a.MAIO != nil && *a.MAIO != 63) {
				t.Fatal("MAIO selector/value lost")
			}
			switch {
			case strings.Contains(c.name, "-single-"):
				if a.ARFCNChoice.Alternative != ECMobileAllocationStructARFCNChoiceAlternativeARFCN || a.ARFCNChoice.ARFCN == nil || a.ARFCNChoice.ARFCN.ARFCN != 1023 || a.ARFCNChoice.HSN != nil {
					t.Fatal("single ARFCN branch lost")
				}
			default:
				if a.ARFCNChoice.Alternative != ECMobileAllocationStructARFCNChoiceAlternativeHSN || a.ARFCNChoice.HSN == nil || a.ARFCNChoice.ARFCN != nil {
					t.Fatal("multiple ARFCN branch lost")
				}
				group := a.ARFCNChoice.HSN.HSNGroup
				if strings.Contains(c.name, "-previous-") {
					if group != nil {
						t.Fatal("previous allocation changed")
					}
					return
				}
				if group == nil {
					t.Fatal("indicated allocation absent")
				}
				hopping := strings.Contains(c.name, "-hopping-")
				if (group.HSN != nil) != hopping || (group.HSN != nil && *group.HSN != 63) {
					t.Fatal("HSN selector/value lost")
				}
				want := 0
				if strings.HasSuffix(c.name, "-63") {
					want = 63
				}
				if int(group.MALENGTH) != want || group.MABITMAP.BitLength != want+1 {
					t.Fatal("bitmap length lost")
				}
				for bit := 0; bit < group.MABITMAP.BitLength; bit++ {
					if group.MABITMAP.Bytes[bit/8]&(1<<uint(7-bit%8)) == 0 {
						t.Fatal("bitmap bit lost")
					}
				}
				group.MABITMAP.BitLength--
				if _, err := EncodeECSystemInformationType1CanonicalAtLength(v, len(wire)); err == nil {
					t.Fatal("invalid bitmap width encoded")
				}
			}
		})
	}
}

func TestECMobileAllocationListLinearGrowth(t *testing.T) {
	if raceEnabled {
		t.Skip("allocation counts vary under the race detector")
	}
	// TS 44.018 V19.0.0 §9.1.43p paragraph 9758 permits an indefinite list.
	previous := float64(0)
	for count := 16; count <= 256; count *= 2 {
		value := ECSystemInformationType1{MessageType: 1}
		for i := 0; i < count; i++ {
			value.ECMobileAllocationList.ECMobileAllocationStructList = append(value.ECMobileAllocationList.ECMobileAllocationStructList, ECMobileAllocationListStructECMobileAllocationStructListEntry{ECMobileAllocationStruct: ECMobileAllocationStruct{ARFCNChoice: ECMobileAllocationStructARFCNChoice{Alternative: ECMobileAllocationStructARFCNChoiceAlternativeARFCN, ARFCN: &ECMobileAllocationStructARFCNChoiceARFCN{ARFCN: 1}}}})
		}
		octets := (15 + 18*count + 7) / 8
		wire, err := EncodeECSystemInformationType1CanonicalAtLength(value, octets)
		if err != nil {
			t.Fatal(err)
		}
		var decoded runtime.Decoded[ECSystemInformationType1]
		allocations := testing.AllocsPerRun(3, func() { decoded, err = DecodeECSystemInformationType1(wire) })
		if err != nil || len(decoded.Value.ECMobileAllocationList.ECMobileAllocationStructList) != count {
			t.Fatalf("decode count=%d: %v", count, err)
		}
		if previous != 0 && allocations > previous*2+16 {
			t.Fatalf("allocation growth count=%d: %g after %g", count, allocations, previous)
		}
		previous = allocations
	}
}

func TestECSystemInformationType1TypedCanonical(t *testing.T) {
	value := ECSystemInformationType1{MessageType: 1}
	wire, err := EncodeECSystemInformationType1CanonicalAtLength(value, 4)
	if err != nil || !bytes.Equal(wire, []byte{0x20, 0x01, 0x2b, 0x2b}) {
		t.Fatalf("fresh typed SI1 %x: %v", wire, err)
	}
	value.ECSI1INDEX = 3
	value.ECSI1COUNT = 3
	value.ECSICHANGEMARK = 31
	value.BandIndicator = 1
	wire, err = EncodeECSystemInformationType1CanonicalAtLength(value, 4)
	if err != nil || !bytes.Equal(wire, []byte{0x3f, 0xf3, 0x2b, 0x2b}) {
		t.Fatalf("boundary typed SI1 %x: %v", wire, err)
	}
}
