package notification

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	goruntime "runtime"
	"runtime/debug"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

type notificationVector struct {
	Name, Hex       string
	Bits, Mandatory int
	Numbers         map[string]uint64
	Absent          []string
	Strings         map[string]string
}

// TS 44.018 V19.0.0 §9.1.21a, paragraphs 7281-7304, 7337-7345; message assignment:
// §10.4 table 10.4.2. These synthetic values have an independent field reading.
const notificationVectorJSON = `[{"Name":"form-0-release-0","Hex":"0455e6f7891b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":46,"Numbers":{"GroupCallReference":46118400291},"Absent":["ChannelDescription","CELLGLOBALCOUNT","B22COUNT","VSTKRAND","PriorityUplinkAccess","EmergencyInd","AMRConfig","PagingCause","SMSDataConfidentialityInd","SMSGuaranteedPrivacyInd"],"Strings":{}},{"Name":"form-0-release-1","Hex":"0455e6f78919fabcdef1232b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":46,"Numbers":{"GroupCallReference":46118400291,"CELLGLOBALCOUNT":3,"B22COUNT":1,"VSTKRAND":46118400291},"Absent":["ChannelDescription","PriorityUplinkAccess","EmergencyInd","AMRConfig","PagingCause","SMSDataConfidentialityInd","SMSGuaranteedPrivacyInd"],"Strings":{}},{"Name":"form-0-release-2","Hex":"0455e6f7891afff32b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":46,"Numbers":{"GroupCallReference":46118400291,"PriorityUplinkAccess":1,"EmergencyInd":1,"AMRConfig":15,"PagingCause":3,"SMSDataConfidentialityInd":1,"SMSGuaranteedPrivacyInd":0},"Absent":["ChannelDescription","CELLGLOBALCOUNT","B22COUNT","VSTKRAND"],"Strings":{}},{"Name":"form-0-release-3","Hex":"0455e6f78919fabcdef123fffb2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":46,"Numbers":{"GroupCallReference":46118400291,"CELLGLOBALCOUNT":3,"B22COUNT":1,"VSTKRAND":46118400291,"PriorityUplinkAccess":1,"EmergencyInd":1,"AMRConfig":15,"PagingCause":3,"SMSDataConfidentialityInd":1,"SMSGuaranteedPrivacyInd":0},"Absent":["ChannelDescription"],"Strings":{}},{"Name":"form-1-release-0","Hex":"0455e6f7891c48d1592b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":71,"Numbers":{"GroupCallReference":46118400291,"ChannelDescription":1193046},"Absent":["CELLGLOBALCOUNT","B22COUNT","VSTKRAND","PriorityUplinkAccess","EmergencyInd","AMRConfig","PagingCause","SMSDataConfidentialityInd","SMSGuaranteedPrivacyInd"],"Strings":{}},{"Name":"form-1-release-1","Hex":"0455e6f7891c48d158fd5e6f7891ab2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":71,"Numbers":{"GroupCallReference":46118400291,"ChannelDescription":1193046,"CELLGLOBALCOUNT":3,"B22COUNT":1,"VSTKRAND":46118400291},"Absent":["PriorityUplinkAccess","EmergencyInd","AMRConfig","PagingCause","SMSDataConfidentialityInd","SMSGuaranteedPrivacyInd"],"Strings":{}},{"Name":"form-1-release-2","Hex":"0455e6f7891c48d159fffb2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":71,"Numbers":{"GroupCallReference":46118400291,"ChannelDescription":1193046,"PriorityUplinkAccess":1,"EmergencyInd":1,"AMRConfig":15,"PagingCause":3,"SMSDataConfidentialityInd":1,"SMSGuaranteedPrivacyInd":0},"Absent":["CELLGLOBALCOUNT","B22COUNT","VSTKRAND"],"Strings":{}},{"Name":"form-1-release-3","Hex":"0455e6f7891c48d158fd5e6f7891fffd2b2b2b2b2b2b2b","Bits":184,"Mandatory":71,"Numbers":{"GroupCallReference":46118400291,"ChannelDescription":1193046,"CELLGLOBALCOUNT":3,"B22COUNT":1,"VSTKRAND":46118400291,"PriorityUplinkAccess":1,"EmergencyInd":1,"AMRConfig":15,"PagingCause":3,"SMSDataConfidentialityInd":1,"SMSGuaranteedPrivacyInd":0},"Absent":[],"Strings":{}},{"Name":"form-2-release-0","Hex":"0455e6f7891c48d15a01a52b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":88,"Numbers":{"GroupCallReference":46118400291,"ChannelDescription":1193046,"Length":1},"Absent":["CELLGLOBALCOUNT","B22COUNT","VSTKRAND","PriorityUplinkAccess","EmergencyInd","AMRConfig","PagingCause","SMSDataConfidentialityInd","SMSGuaranteedPrivacyInd"],"Strings":{"Value":"a5"}},{"Name":"form-2-release-1","Hex":"0455e6f7891c48d15a01a5feaf37bc48eb2b2b2b2b2b2b","Bits":184,"Mandatory":88,"Numbers":{"GroupCallReference":46118400291,"ChannelDescription":1193046,"Length":1,"CELLGLOBALCOUNT":3,"B22COUNT":1,"VSTKRAND":46118400291},"Absent":["PriorityUplinkAccess","EmergencyInd","AMRConfig","PagingCause","SMSDataConfidentialityInd","SMSGuaranteedPrivacyInd"],"Strings":{"Value":"a5"}},{"Name":"form-2-release-2","Hex":"0455e6f7891c48d15a01a57ffd2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":88,"Numbers":{"GroupCallReference":46118400291,"ChannelDescription":1193046,"Length":1,"PriorityUplinkAccess":1,"EmergencyInd":1,"AMRConfig":15,"PagingCause":3,"SMSDataConfidentialityInd":1,"SMSGuaranteedPrivacyInd":0},"Absent":["CELLGLOBALCOUNT","B22COUNT","VSTKRAND"],"Strings":{"Value":"a5"}},{"Name":"form-2-release-3","Hex":"0455e6f7891c48d15a01a5feaf37bc48dffe2b2b2b2b2b","Bits":184,"Mandatory":88,"Numbers":{"GroupCallReference":46118400291,"ChannelDescription":1193046,"Length":1,"CELLGLOBALCOUNT":3,"B22COUNT":1,"VSTKRAND":46118400291,"PriorityUplinkAccess":1,"EmergencyInd":1,"AMRConfig":15,"PagingCause":3,"SMSDataConfidentialityInd":1,"SMSGuaranteedPrivacyInd":0},"Absent":[],"Strings":{"Value":"a5"}},{"Name":"form-3-release-0","Hex":"0455e6f7891c48d15b0123456789abcdef2b2b2b2b2b2b","Bits":184,"Mandatory":136,"Numbers":{"GroupCallReference":46118400291,"ChannelDescription":1193046,"FrequencyShortList":81985529216486895},"Absent":["CELLGLOBALCOUNT","B22COUNT","VSTKRAND","PriorityUplinkAccess","EmergencyInd","AMRConfig","PagingCause","SMSDataConfidentialityInd","SMSGuaranteedPrivacyInd"],"Strings":{}},{"Name":"form-3-release-1","Hex":"0455e6f7891c48d15b0123456789abcdeffeaf37bc48eb","Bits":184,"Mandatory":136,"Numbers":{"GroupCallReference":46118400291,"ChannelDescription":1193046,"FrequencyShortList":81985529216486895,"CELLGLOBALCOUNT":3,"B22COUNT":1,"VSTKRAND":46118400291},"Absent":["PriorityUplinkAccess","EmergencyInd","AMRConfig","PagingCause","SMSDataConfidentialityInd","SMSGuaranteedPrivacyInd"],"Strings":{}},{"Name":"form-3-release-2","Hex":"0455e6f7891c48d15b0123456789abcdef7ffd2b2b2b2b","Bits":184,"Mandatory":136,"Numbers":{"GroupCallReference":46118400291,"ChannelDescription":1193046,"FrequencyShortList":81985529216486895,"PriorityUplinkAccess":1,"EmergencyInd":1,"AMRConfig":15,"PagingCause":3,"SMSDataConfidentialityInd":1,"SMSGuaranteedPrivacyInd":0},"Absent":["CELLGLOBALCOUNT","B22COUNT","VSTKRAND"],"Strings":{}},{"Name":"form-3-release-3","Hex":"0455e6f7891c48d15b0123456789abcdeffeaf37bc48dffe","Bits":192,"Mandatory":136,"Numbers":{"GroupCallReference":46118400291,"ChannelDescription":1193046,"FrequencyShortList":81985529216486895,"CELLGLOBALCOUNT":3,"B22COUNT":1,"VSTKRAND":46118400291,"PriorityUplinkAccess":1,"EmergencyInd":1,"AMRConfig":15,"PagingCause":3,"SMSDataConfidentialityInd":1,"SMSGuaranteedPrivacyInd":0},"Absent":[],"Strings":{}},{"Name":"form-4-release-0","Hex":"0480d28b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":28,"Numbers":{"Length":1,"ChannelFirst":0},"Absent":["EMLPPPriority","CELLGLOBALCOUNT","B22COUNT","VSTKRAND","PriorityUplinkAccess","EmergencyInd","AMRConfig","PagingCause","SMSDataConfidentialityInd","SMSGuaranteedPrivacyInd"],"Strings":{"Value":"a5"}},{"Name":"form-4-release-1","Hex":"0480d287eaf37bc48f2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":28,"Numbers":{"Length":1,"ChannelFirst":0,"CELLGLOBALCOUNT":3,"B22COUNT":1,"VSTKRAND":46118400291},"Absent":["EMLPPPriority","PriorityUplinkAccess","EmergencyInd","AMRConfig","PagingCause","SMSDataConfidentialityInd","SMSGuaranteedPrivacyInd"],"Strings":{"Value":"a5"}},{"Name":"form-4-release-2","Hex":"0480d28fffcb2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":28,"Numbers":{"Length":1,"ChannelFirst":0,"PriorityUplinkAccess":1,"EmergencyInd":1,"AMRConfig":15,"PagingCause":3,"SMSDataConfidentialityInd":1,"SMSGuaranteedPrivacyInd":0},"Absent":["EMLPPPriority","CELLGLOBALCOUNT","B22COUNT","VSTKRAND"],"Strings":{"Value":"a5"}},{"Name":"form-4-release-3","Hex":"0480d287eaf37bc48dffeb2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":28,"Numbers":{"Length":1,"ChannelFirst":0,"CELLGLOBALCOUNT":3,"B22COUNT":1,"VSTKRAND":46118400291,"PriorityUplinkAccess":1,"EmergencyInd":1,"AMRConfig":15,"PagingCause":3,"SMSDataConfidentialityInd":1,"SMSGuaranteedPrivacyInd":0},"Absent":["EMLPPPriority"],"Strings":{"Value":"a5"}},{"Name":"form-5-release-0","Hex":"0480d2bf2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":31,"Numbers":{"Length":1,"ChannelFirst":1,"EMLPPPriority":7},"Absent":["CELLGLOBALCOUNT","B22COUNT","VSTKRAND","PriorityUplinkAccess","EmergencyInd","AMRConfig","PagingCause","SMSDataConfidentialityInd","SMSGuaranteedPrivacyInd"],"Strings":{"Value":"a5"}},{"Name":"form-5-release-1","Hex":"0480d2befd5e6f7891ab2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":31,"Numbers":{"Length":1,"ChannelFirst":1,"EMLPPPriority":7,"CELLGLOBALCOUNT":3,"B22COUNT":1,"VSTKRAND":46118400291},"Absent":["PriorityUplinkAccess","EmergencyInd","AMRConfig","PagingCause","SMSDataConfidentialityInd","SMSGuaranteedPrivacyInd"],"Strings":{"Value":"a5"}},{"Name":"form-5-release-2","Hex":"0480d2bffffb2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":31,"Numbers":{"Length":1,"ChannelFirst":1,"EMLPPPriority":7,"PriorityUplinkAccess":1,"EmergencyInd":1,"AMRConfig":15,"PagingCause":3,"SMSDataConfidentialityInd":1,"SMSGuaranteedPrivacyInd":0},"Absent":["CELLGLOBALCOUNT","B22COUNT","VSTKRAND"],"Strings":{"Value":"a5"}},{"Name":"form-5-release-3","Hex":"0480d2befd5e6f7891fffd2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":31,"Numbers":{"Length":1,"ChannelFirst":1,"EMLPPPriority":7,"CELLGLOBALCOUNT":3,"B22COUNT":1,"VSTKRAND":46118400291,"PriorityUplinkAccess":1,"EmergencyInd":1,"AMRConfig":15,"PagingCause":3,"SMSDataConfidentialityInd":1,"SMSGuaranteedPrivacyInd":0},"Absent":[],"Strings":{"Value":"a5"}},{"Name":"form-6-release-0","Hex":"0481d2d2d2cb2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":44,"Numbers":{"Length":3,"ChannelFirst":2},"Absent":["EMLPPPriority","CELLGLOBALCOUNT","B22COUNT","VSTKRAND","PriorityUplinkAccess","EmergencyInd","AMRConfig","PagingCause","SMSDataConfidentialityInd","SMSGuaranteedPrivacyInd"],"Strings":{"Value":"a5a5a5"}},{"Name":"form-6-release-1","Hex":"0481d2d2d2c7eaf37bc48f2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":44,"Numbers":{"Length":3,"ChannelFirst":2,"CELLGLOBALCOUNT":3,"B22COUNT":1,"VSTKRAND":46118400291},"Absent":["EMLPPPriority","PriorityUplinkAccess","EmergencyInd","AMRConfig","PagingCause","SMSDataConfidentialityInd","SMSGuaranteedPrivacyInd"],"Strings":{"Value":"a5a5a5"}},{"Name":"form-6-release-2","Hex":"0481d2d2d2cfffcb2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":44,"Numbers":{"Length":3,"ChannelFirst":2,"PriorityUplinkAccess":1,"EmergencyInd":1,"AMRConfig":15,"PagingCause":3,"SMSDataConfidentialityInd":1,"SMSGuaranteedPrivacyInd":0},"Absent":["EMLPPPriority","CELLGLOBALCOUNT","B22COUNT","VSTKRAND"],"Strings":{"Value":"a5a5a5"}},{"Name":"form-6-release-3","Hex":"0481d2d2d2c7eaf37bc48dffeb2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":44,"Numbers":{"Length":3,"ChannelFirst":2,"CELLGLOBALCOUNT":3,"B22COUNT":1,"VSTKRAND":46118400291,"PriorityUplinkAccess":1,"EmergencyInd":1,"AMRConfig":15,"PagingCause":3,"SMSDataConfidentialityInd":1,"SMSGuaranteedPrivacyInd":0},"Absent":["EMLPPPriority"],"Strings":{"Value":"a5a5a5"}},{"Name":"form-7-release-0","Hex":"0484d2d2d2d2d2d2d2d2d2ff2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":95,"Numbers":{"Length":9,"ChannelFirst":3,"EMLPPPriority":7},"Absent":["CELLGLOBALCOUNT","B22COUNT","VSTKRAND","PriorityUplinkAccess","EmergencyInd","AMRConfig","PagingCause","SMSDataConfidentialityInd","SMSGuaranteedPrivacyInd"],"Strings":{"Value":"a5a5a5a5a5a5a5a5a5"}},{"Name":"form-7-release-1","Hex":"0484d2d2d2d2d2d2d2d2d2fefd5e6f7891ab2b2b2b2b2b","Bits":184,"Mandatory":95,"Numbers":{"Length":9,"ChannelFirst":3,"EMLPPPriority":7,"CELLGLOBALCOUNT":3,"B22COUNT":1,"VSTKRAND":46118400291},"Absent":["PriorityUplinkAccess","EmergencyInd","AMRConfig","PagingCause","SMSDataConfidentialityInd","SMSGuaranteedPrivacyInd"],"Strings":{"Value":"a5a5a5a5a5a5a5a5a5"}},{"Name":"form-7-release-2","Hex":"0484d2d2d2d2d2d2d2d2d2fffffb2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":95,"Numbers":{"Length":9,"ChannelFirst":3,"EMLPPPriority":7,"PriorityUplinkAccess":1,"EmergencyInd":1,"AMRConfig":15,"PagingCause":3,"SMSDataConfidentialityInd":1,"SMSGuaranteedPrivacyInd":0},"Absent":["CELLGLOBALCOUNT","B22COUNT","VSTKRAND"],"Strings":{"Value":"a5a5a5a5a5a5a5a5a5"}},{"Name":"form-7-release-3","Hex":"0484d2d2d2d2d2d2d2d2d2fefd5e6f7891fffd2b2b2b2b","Bits":184,"Mandatory":95,"Numbers":{"Length":9,"ChannelFirst":3,"EMLPPPriority":7,"CELLGLOBALCOUNT":3,"B22COUNT":1,"VSTKRAND":46118400291,"PriorityUplinkAccess":1,"EmergencyInd":1,"AMRConfig":15,"PagingCause":3,"SMSDataConfidentialityInd":1,"SMSGuaranteedPrivacyInd":0},"Absent":[],"Strings":{"Value":"a5a5a5a5a5a5a5a5a5"}},{"Name":"optional-absent","Hex":"040000000000412b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":46,"Numbers":{"GroupCallReference":0,"PriorityUplinkAccess":0,"EmergencyInd":0},"Absent":["AMRConfig","B22COUNT","CELLGLOBALCOUNT","ChannelDescription","PagingCause","SMSDataConfidentialityInd","SMSGuaranteedPrivacyInd","VSTKRAND"],"Strings":{}},{"Name":"optional-cell-count","Hex":"04000000000105032b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":46,"Numbers":{"GroupCallReference":0,"PriorityUplinkAccess":1,"EmergencyInd":0,"CELLGLOBALCOUNT":0,"B22COUNT":0,"AMRConfig":0},"Absent":["ChannelDescription","PagingCause","SMSDataConfidentialityInd","SMSGuaranteedPrivacyInd","VSTKRAND"],"Strings":{}},{"Name":"optional-vstk","Hex":"04000000000080000000054b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":46,"Numbers":{"GroupCallReference":0,"PriorityUplinkAccess":0,"EmergencyInd":1,"VSTKRAND":0,"PagingCause":0,"SMSDataConfidentialityInd":0,"SMSGuaranteedPrivacyInd":1},"Absent":["AMRConfig","B22COUNT","CELLGLOBALCOUNT","ChannelDescription"],"Strings":{}}]`

func notificationVectors(t testing.TB) []notificationVector {
	t.Helper()
	var rows []notificationVector
	if err := json.Unmarshal([]byte(notificationVectorJSON), &rows); err != nil {
		t.Fatal(err)
	}
	return rows
}

func notificationFields(value any) map[string][]reflect.Value {
	fields := map[string][]reflect.Value{}
	var walk func(reflect.Value, string)
	walk = func(v reflect.Value, name string) {
		if !v.IsValid() {
			return
		}
		for v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
			if v.IsNil() {
				return
			}
			v = v.Elem()
		}
		if v.Type() == reflect.TypeFor[runtime.WireInfo]() {
			return
		}
		if v.Type() == reflect.TypeFor[runtime.BitString]() {
			fields[name] = append(fields[name], v)
			return
		}
		switch v.Kind() {
		case reflect.Struct:
			for i := 0; i < v.NumField(); i++ {
				if v.Type().Field(i).IsExported() {
					walk(v.Field(i), v.Type().Field(i).Name)
				}
			}
		case reflect.Slice, reflect.Array:
			for i := 0; i < v.Len(); i++ {
				walk(v.Index(i), name)
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			fields[name] = append(fields[name], v)
		}
	}
	walk(reflect.ValueOf(value), "")
	return fields
}

func notificationCheckFields(t *testing.T, v NOTIFICATIONFACCH, row notificationVector) {
	t.Helper()
	if v.RRShortPD != 0 || v.MessageType != 1 || v.ShortLayer2Header != 0 {
		t.Fatalf("header: %#v", v)
	}
	fields := notificationFields(v)
	for name, want := range row.Numbers {
		got := fields[name]
		if len(got) != 1 || got[0].Kind() == reflect.Struct || got[0].Uint() != want {
			t.Fatalf("%s: got %v want %d", name, got, want)
		}
	}
	for _, name := range row.Absent {
		if len(fields[name]) != 0 {
			t.Fatalf("%s unexpectedly present", name)
		}
	}
	for name, want := range row.Strings {
		got := fields[name]
		if len(got) != 1 {
			t.Fatalf("%s: %d bit strings", name, len(got))
		}
		b, ok := got[0].Interface().(runtime.BitString)
		if !ok || b.BitLength != len(want)*4 || hex.EncodeToString(b.Bytes) != want {
			t.Fatalf("%s: %#v want %s", name, b, want)
		}
	}
}

func notificationEqualBits(a, b []byte, n int) bool {
	if len(a)*8 < n || len(b)*8 < n {
		return false
	}
	for i := 0; i < n; i++ {
		if (a[i/8]>>uint(7-i%8))&1 != (b[i/8]>>uint(7-i%8))&1 {
			return false
		}
	}
	return true
}

func TestNOTIFICATIONFACCHSourceVectors(t *testing.T) {
	d, err := LookupClause("9.1.21a", "NOTIFICATION FACCH")
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range notificationVectors(t) {
		t.Run(row.Name, func(t *testing.T) {
			wire, err := hex.DecodeString(row.Hex)
			if err != nil {
				t.Fatal(err)
			}
			raw, err := d.DecodeBits(wire, row.Bits)
			if err != nil {
				t.Fatal(err)
			}
			decoded, ok := raw.(runtime.Decoded[NOTIFICATIONFACCH])
			if !ok {
				t.Fatalf("typed result %T", raw)
			}
			if decoded.BitsConsumed != row.Bits || decoded.Tail.BitLength != 0 {
				t.Fatalf("consumed %d of %d; tail %d", decoded.BitsConsumed, row.Bits, decoded.Tail.BitLength)
			}
			notificationCheckFields(t, decoded.Value, row)
			w := runtime.NewWriter()
			if err := EncodeNOTIFICATIONFACCHTo(w, decoded.Value); err != nil {
				t.Fatal(err)
			}
			replay, err := w.FinishBits(decoded.Tail)
			if err != nil || replay.BitLength != row.Bits || !notificationEqualBits(replay.Bytes, wire, row.Bits) {
				t.Fatalf("replay %#v: %v", replay, err)
			}
			canonicalWriter := runtime.NewWriter()
			fresh := runtime.CanonicalTo(decoded.Value, row.Bits)
			canonicalWriter.WithWire(fresh.Wire)
			if err := EncodeNOTIFICATIONFACCHTo(canonicalWriter, fresh); err != nil {
				t.Fatal(err)
			}
			canonical, err := canonicalWriter.FinishBits(runtime.BitString{})
			if err != nil || canonical.BitLength != row.Bits {
				t.Fatalf("canonical %#v: %v", canonical, err)
			}
			reread, err := d.DecodeBits(canonical.Bytes, canonical.BitLength)
			if err != nil {
				t.Fatal(err)
			}
			typed := reread.(runtime.Decoded[NOTIFICATIONFACCH])
			notificationCheckFields(t, typed.Value, row)
			if !runtime.SemanticallyEqual(typed.Value, decoded.Value) {
				t.Fatal("canonical changed typed fields")
			}

			for phase := 0; phase < 8; phase++ {
				fresh := runtime.CanonicalTo(decoded.Value, row.Bits+phase)
				positioned := runtime.NewWriter()
				if err := positioned.WriteUint(0, phase); err != nil {
					t.Fatal(err)
				}
				positioned.WithWire(fresh.Wire)
				if _, err := positioned.PushLimit(row.Bits); err != nil {
					t.Fatal(err)
				}
				if err := EncodeNOTIFICATIONFACCHTo(positioned, fresh); err != nil {
					t.Fatalf("phase %d: %v", phase, err)
				}
				at, err := positioned.FinishBits(runtime.BitString{})
				if err != nil || at.BitLength != row.Bits+phase {
					t.Fatalf("phase %d extent %#v: %v", phase, at, err)
				}
				pr, err := runtime.NewChoiceReaderBits(at.Bytes, at.BitLength, nil)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := pr.ReadUint(phase); err != nil {
					t.Fatal(err)
				}
				reread, err := d.DecodeFrom(pr)
				if err != nil {
					t.Fatalf("phase %d decode: %v", phase, err)
				}
				notificationCheckFields(t, reread.(runtime.Decoded[NOTIFICATIONFACCH]).Value, row)
			}
			edited := decoded.Value
			edited.RRShortPD = 1
			ew := runtime.NewWriter()
			if err := EncodeNOTIFICATIONFACCHTo(ew, edited); err != nil {
				t.Fatal(err)
			}
			edit, err := ew.FinishBits(decoded.Tail)
			want := bytes.Clone(wire)
			want[0] ^= 0x80
			if err != nil || edit.BitLength != row.Bits || !notificationEqualBits(edit.Bytes, want, row.Bits) {
				t.Fatalf("edited replay %#v: %v", edit, err)
			}
			for cut := 0; cut < row.Mandatory; cut++ {
				r, err := runtime.NewChoiceReaderBits(wire, cut, nil)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := d.DecodeFrom(r); err == nil {
					t.Fatalf("accepted mandatory truncation %d", cut)
				}
				if r.Position() > cut {
					t.Fatalf("crossed %d-bit bound", cut)
				}
			}
			containing := make([]byte, (row.Bits+8+7)/8)
			for bit := 0; bit < row.Bits+8; bit++ {
				value := byte(0)
				if bit < row.Bits {
					value = wire[bit/8] >> uint(7-bit%8) & 1
				} else {
					value = 0xd7 >> uint(7-(bit-row.Bits)) & 1
				}
				containing[bit/8] |= value << uint(7-bit%8)
			}
			r, err := runtime.NewChoiceReaderBits(containing, row.Bits+8, nil)
			if err != nil {
				t.Fatal(err)
			}
			old, err := r.PushLimit(row.Bits)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := d.DecodeFrom(r); err != nil {
				t.Fatal(err)
			}
			if r.Position() != row.Bits {
				t.Fatal("wrong bounded consumption")
			}
			if err := r.PopLimit(old); err != nil {
				t.Fatal(err)
			}
			sentinel, err := r.ReadUint(8)
			if err != nil || sentinel != 0xd7 {
				t.Fatalf("sentinel %x: %v", sentinel, err)
			}
			if row.Bits%8 == 0 {
				canonicalBytes, err := d.CanonicalAtLength(decoded, len(wire))
				if len(wire) > 23 {
					// TS 44.006 V19.0.0 §8.8.3: the main-DCCH Bter
					// sender cap leaves oversized reception and replay intact.
					var extent *runtime.ExtentError
					if canonicalBytes != nil || !errors.As(err, &extent) || extent.Maximum != 23 || extent.Actual != len(wire) {
						t.Fatalf("past carrier cap: %v", err)
					}
				} else {
					if err != nil {
						t.Fatal(err)
					}
					reread, err := d.Decode(canonicalBytes)
					if err != nil {
						t.Fatal(err)
					}
					notificationCheckFields(t, reread.(runtime.Decoded[NOTIFICATIONFACCH]).Value, row)
					stable, err := d.CanonicalAtLength(reread, len(wire))
					if err != nil || !bytes.Equal(stable, canonicalBytes) {
						t.Fatalf("unstable canonical %x: %v", stable, err)
					}
				}
			}
			for candidate := uint8(0); candidate < 32; candidate++ {
				if candidate == 1 {
					continue
				}
				bad := bytes.Clone(wire)
				bad[0] = (bad[0] & 0x83) | (candidate << 2)
				if _, err := d.DecodeBits(bad, row.Bits); err == nil {
					t.Fatalf("accepted message type %d", candidate)
				}
			}
		})
	}
}

func TestNOTIFICATIONFACCHInputGrowth(t *testing.T) {
	d, err := LookupClause("9.1.21a", "NOTIFICATION FACCH")
	if err != nil {
		t.Fatal(err)
	}
	rows := notificationVectors(t)
	seed, err := hex.DecodeString(rows[len(rows)-1].Hex)
	if err != nil {
		t.Fatal(err)
	}
	// This is a decoder-storage test with caller-supplied bounds; these larger
	// buffers do not assert a radio carrier's permissible wire extent.
	for _, n := range []int{64, 256, 1024, 4096, 8192} {
		wire := append(bytes.Clone(seed), bytes.Repeat([]byte{0x2b}, n)...)
		beforeGC := debug.SetGCPercent(-1)
		var before, after goruntime.MemStats
		goruntime.ReadMemStats(&before)
		for i := 0; i < 4; i++ {
			if _, err := d.Decode(wire); err != nil {
				debug.SetGCPercent(beforeGC)
				t.Fatal(err)
			}
		}
		goruntime.ReadMemStats(&after)
		debug.SetGCPercent(beforeGC)
		allocated := (after.TotalAlloc - before.TotalAlloc) / 4
		if !raceEnabled && allocated > uint64(len(wire))*256+262144 {
			t.Fatalf("storage grew beyond linear bound: %d input bytes, %d allocated bytes", len(wire), allocated)
		}
	}
}

func FuzzNOTIFICATIONFACCH(f *testing.F) {
	d, err := LookupClause("9.1.21a", "NOTIFICATION FACCH")
	if err != nil {
		f.Fatal(err)
	}
	for _, row := range notificationVectors(f) {
		b, err := hex.DecodeString(row.Hex)
		if err != nil {
			f.Fatal(err)
		}
		f.Add(b, row.Bits)
	}
	f.Fuzz(func(t *testing.T, data []byte, bits int) {
		if len(data) > 4096 || bits < 0 || bits > len(data)*8 {
			return
		}
		raw, err := d.DecodeBits(data, bits)
		if err != nil {
			return
		}
		decoded, ok := raw.(runtime.Decoded[NOTIFICATIONFACCH])
		if !ok {
			t.Fatalf("typed decode %T", raw)
		}
		w := runtime.NewWriter()
		if err := EncodeNOTIFICATIONFACCHTo(w, decoded.Value); err != nil {
			t.Fatal(err)
		}
		out, err := w.FinishBits(decoded.Tail)
		if err != nil || out.BitLength != bits || !notificationEqualBits(out.Bytes, data, bits) {
			t.Fatalf("accepted input replay %#v: %v", out, err)
		}
	})
}

func TestNOTIFICATIONFACCHFieldBounds(t *testing.T) {
	row := notificationVectors(t)[0]
	wire, err := hex.DecodeString(row.Hex)
	if err != nil {
		t.Fatal(err)
	}
	d, err := LookupClause("9.1.21a", "NOTIFICATION FACCH")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := d.DecodeBits(wire, row.Bits)
	if err != nil {
		t.Fatal(err)
	}
	original := raw.(runtime.Decoded[NOTIFICATIONFACCH]).Value
	for _, field := range []string{"RRShortPD", "ShortLayer2Header", "MessageType"} {
		v := runtime.CanonicalTo(original, row.Bits)
		switch field {
		case "RRShortPD":
			v.RRShortPD = 2
		case "ShortLayer2Header":
			v.ShortLayer2Header = 4
		case "MessageType":
			v.MessageType = 31
		}
		if err := EncodeNOTIFICATIONFACCHTo(runtime.NewWriter(), v); err == nil {
			t.Fatalf("accepted invalid %s", field)
		}
	}
}

// TS 44.018 V19.0.0 §10.5.2.21 p16717 and TS 24.008 V20.1.0
// §10.5.1.4 p20956 require at least one LV value octet in these messages.
func TestNOTIFICATIONFACCHZeroLV(t *testing.T) {
	d, err := LookupClause("9.1.21a", "NOTIFICATION FACCH")
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		Hex  string
		Bits int
	}
	if err := json.Unmarshal([]byte(`[{"Hex":"04800b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184},{"Hex":"048007eaf37bc48f2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184},{"Hex":"04800fffcb2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184},{"Hex":"048007eaf37bc48dffeb2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184}]`), &rows); err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		wire, err := hex.DecodeString(row.Hex)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := d.DecodeBits(wire, row.Bits); err == nil {
			t.Fatalf("accepted zero-length LV %x", wire)
		}
	}
}

func FuzzPagingInformation(f *testing.F) {
	d, err := LookupClause("9.1.21a", "Paging Information")
	if err != nil {
		f.Fatal(err)
	}
	for _, row := range notificationVectors(f) {
		wire, err := hex.DecodeString(row.Hex)
		if err != nil {
			f.Fatal(err)
		}
		f.Add(wire)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 4096 {
			return
		}
		raw, err := d.Decode(data)
		if err != nil {
			return
		}
		if _, ok := raw.(runtime.Decoded[PagingInformation]); !ok {
			t.Fatalf("typed helper %T", raw)
		}
		out, err := d.Encode(raw)
		if err != nil || !bytes.Equal(out, data) {
			t.Fatalf("helper replay %x: %v", out, err)
		}
	})
}
