package nongprsoptions

import (
	"bytes"
	"errors"
	"fmt"
	goruntime "runtime"
	"runtime/debug"
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

func cellOptionsPack(text string) []byte {
	b := make([]byte, (len(text)+7)/8)
	for i, c := range text {
		if c == '1' {
			b[i/8] |= 1 << uint(7-i%8)
		}
	}
	return b
}

func cellOptionsValue(timer, extension bool, ecsc, ecsr uint8, length int) (NonGPRSCellOptionsIE, string) {
	v := NonGPRSCellOptionsIE{ATT: 1, NECI: 1, PWRC: 0, DTX: 2, RADIOLINKTIMEOUT: 13, BSAGBLKSRES: 5, CCCHCONF: 6, BSPAMFRMS: 3, MAXRETRANS: 2, TXINTEGER: 11, EC: 1, MSTXPWRMAXCCCH: 17}
	text := "1"
	if timer {
		x := uint8(173)
		v.T3212 = &x
		text += "1" + fmt.Sprintf("%08b", x)
	} else {
		text += "0"
	}
	text += fmt.Sprintf("%01b%01b%02b%04b%03b%03b%03b%02b%04b%01b%05b", v.NECI, v.PWRC, v.DTX, v.RADIOLINKTIMEOUT, v.BSAGBLKSRES, v.CCCHCONF, v.BSPAMFRMS, v.MAXRETRANS, v.TXINTEGER, v.EC, v.MSTXPWRMAXCCCH)
	if extension {
		v.ExtensionLengthGroup = &NonGPRSCellOptionsIEExtensionLengthGroup{ExtensionLength: uint8(length), Content: NonGPRSCellOptionsIEExtensionLengthGroupExtensionInformationExtensionInformationFallback{Known: &ExtensionInformation{ECSC: ecsc, N3GECSR: ecsr}}}
		text += "1" + fmt.Sprintf("%06b%01b%01b", length, ecsc, ecsr) + strings.Repeat("0", length-1)
	} else {
		text += "0"
	}
	return v, text
}

// TS 44.060 V19.0.0 Table 12.27.1, paragraphs 23460–23484,
// and paragraphs 23494–23496: 3G ECSR is exactly one binary bit.
// In particular it has no position-dependent L/H interpretation.
func TestNonGPRSCellOptions(t *testing.T) {
	for _, timer := range []bool{false, true} {
		for _, extension := range []bool{false, true} {
			for ecsc := uint8(0); ecsc < 2; ecsc++ {
				for ecsr := uint8(0); ecsr < 2; ecsr++ {
					lengths := []int{0}
					if extension {
						lengths = []int{1, 2, 7, 63}
					}
					for _, length := range lengths {
						t.Run(fmt.Sprintf("timer-%t-extension-%t-ecsc-%d-ecsr-%d-length-%d", timer, extension, ecsc, ecsr, length), func(t *testing.T) {
							fresh, text := cellOptionsValue(timer, extension, ecsc, ecsr, length)
							wire := cellOptionsPack(text)
							encoded, err := EncodeNonGPRSCellOptionsIE(fresh)
							if err != nil || !bytes.Equal(encoded, wire) {
								t.Fatalf("fresh %x want %x: %v", encoded, wire, err)
							}
							decoded, err := DecodeNonGPRSCellOptionsIE(wire)
							if err != nil || decoded.BitsConsumed != len(text) {
								t.Fatalf("extent %d want %d: %v", decoded.BitsConsumed, len(text), err)
							}
							v := decoded.Value
							if v.ATT != 1 || v.NECI != 1 || v.PWRC != 0 || v.DTX != 2 || v.RADIOLINKTIMEOUT != 13 || v.BSAGBLKSRES != 5 || v.CCCHCONF != 6 || v.BSPAMFRMS != 3 || v.MAXRETRANS != 2 || v.TXINTEGER != 11 || v.EC != 1 || v.MSTXPWRMAXCCCH != 17 {
								t.Fatalf("typed prefix: %+v", v)
							}
							if (v.T3212 != nil) != timer || timer && *v.T3212 != 173 {
								t.Fatal("timer arm")
							}
							if (v.ExtensionLengthGroup != nil) != extension {
								t.Fatal("extension arm")
							}
							if extension {
								g := v.ExtensionLengthGroup
								if int(g.ExtensionLength) != length || g.Content.Alternative != NonGPRSCellOptionsIEExtensionLengthGroupExtensionInformationExtensionInformationFallbackAlternativeKnown || g.Content.Known == nil || g.Content.Known.ECSC != ecsc || g.Content.Known.N3GECSR != ecsr || g.Content.Ignored != nil {
									t.Fatalf("typed extension: %+v", g)
								}
							}
							replay, err := EncodeNonGPRSCellOptionsIE(v)
							if err != nil || !bytes.Equal(replay, wire) {
								t.Fatalf("replay %x: %v", replay, err)
							}
							canonical, err := EncodeNonGPRSCellOptionsIECanonicalAtLength(v, len(wire))
							if err != nil || !bytes.Equal(canonical, wire) {
								t.Fatalf("canonical %x want %x: %v", canonical, wire, err)
							}
							edited := v
							edited.Wire = runtime.WireInfo{}
							edited.ATT = 0
							changed, err := EncodeNonGPRSCellOptionsIE(edited)
							if err != nil || !bytes.Equal(changed, cellOptionsPack("0"+text[1:])) {
								t.Fatalf("typed edit %x: %v", changed, err)
							}
							for phase := 0; phase < 8; phase++ {
								r := runtime.NewReader(cellOptionsPack(strings.Repeat("1", phase) + text + "10101"))
								if _, err := r.ReadUint(phase); err != nil {
									t.Fatal(err)
								}
								old, err := r.PushLimit(len(text))
								if err != nil {
									t.Fatal(err)
								}
								got, err := DecodeNonGPRSCellOptionsIEFrom(r)
								if err != nil || r.Remaining() != 0 {
									t.Fatalf("bounded phase %d: %v", phase, err)
								}
								if extension && (got.ExtensionLengthGroup.Content.Known == nil || got.ExtensionLengthGroup.Content.Known.N3GECSR != ecsr) {
									t.Fatalf("binary ECSR changed at phase %d", phase)
								}
								if err := r.PopLimit(old); err != nil {
									t.Fatal(err)
								}
								sentinel, err := r.ReadUint(5)
								if err != nil || sentinel != 21 {
									t.Fatalf("sentinel %d: %v", sentinel, err)
								}
							}
							for cut := 0; cut < len(text); cut++ {
								r := runtime.NewReader(wire)
								if _, err := r.PushLimit(cut); err != nil {
									t.Fatal(err)
								}
								if _, err := DecodeNonGPRSCellOptionsIEFrom(r); err == nil {
									t.Fatalf("strict prefix %d/%d accepted", cut, len(text))
								}
								if r.Position() > cut {
									t.Fatalf("crossed prefix %d", cut)
								}
							}
						})
					}
				}
			}
		}
	}
	invalid, _ := cellOptionsValue(false, false, 0, 0, 0)
	invalid.DTX = 4
	if _, err := EncodeNonGPRSCellOptionsIE(invalid); err == nil {
		t.Fatal("three-bit DTX encoded")
	}
	invalid, _ = cellOptionsValue(false, true, 0, 0, 1)
	invalid.ExtensionLengthGroup.Content.Known.N3GECSR = 2
	if _, err := EncodeNonGPRSCellOptionsIE(invalid); err == nil {
		t.Fatal("two-bit ECSR encoded")
	}
}

// The one-bit extension is too short for the known two-bit prefix. The printed
// error region retains its bit, and cannot consume the following sentinel.
func TestNonGPRSTruncatedExtensionErrorRegion(t *testing.T) {
	_, prefix := cellOptionsValue(false, false, 0, 0, 0)
	for _, bit := range []string{"0", "1"} {
		text := prefix[:len(prefix)-1] + "1" + "000000" + bit
		wire := cellOptionsPack(text)
		decoded, err := DecodeNonGPRSCellOptionsIE(wire)
		if err != nil {
			t.Fatal(err)
		}
		group := decoded.Value.ExtensionLengthGroup
		if group == nil || group.Content.Alternative != NonGPRSCellOptionsIEExtensionLengthGroupExtensionInformationExtensionInformationFallbackAlternativeIgnored || group.Content.Known != nil || group.Content.Ignored == nil || group.Content.Ignored.BitLength != 1 {
			t.Fatalf("error region: %+v", group)
		}
		replay, err := EncodeNonGPRSCellOptionsIE(decoded.Value)
		if err != nil || !bytes.Equal(replay, wire) {
			t.Fatalf("ignored replay %x: %v", replay, err)
		}
		canonical, err := EncodeNonGPRSCellOptionsIECanonicalAtLength(decoded.Value, len(wire))
		if err != nil || !bytes.Equal(canonical, wire) {
			t.Fatalf("ignored canonical %x: %v", canonical, err)
		}
		edited := decoded.Value
		edited.Wire = runtime.WireInfo{}
		edited.ATT ^= 1
		changed, err := EncodeNonGPRSCellOptionsIE(edited)
		want := append([]byte(nil), wire...)
		want[0] ^= 0x80
		if err != nil || !bytes.Equal(changed, want) {
			t.Fatalf("ignored typed edit %x want %x: %v", changed, want, err)
		}
		r := runtime.NewReader(cellOptionsPack(text + "10101"))
		old, err := r.PushLimit(len(text))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := DecodeNonGPRSCellOptionsIEFrom(r); err != nil {
			t.Fatal(err)
		}
		if err := r.PopLimit(old); err != nil {
			t.Fatal(err)
		}
		sentinel, err := r.ReadUint(5)
		if err != nil || sentinel != 21 {
			t.Fatalf("sentinel %d: %v", sentinel, err)
		}
	}
}

func TestNonGPRSCellOptionsIECanonicalRequiresExtent(t *testing.T) {
	_, err := EncodeNonGPRSCellOptionsIECanonical(NonGPRSCellOptionsIE{})
	var extent *runtime.ExtentError
	if !errors.As(err, &extent) || !extent.Required {
		t.Fatalf("required canonical extent: %v", err)
	}
}

func FuzzNonGPRSCellOptionsReplay(f *testing.F) {
	_, bits := cellOptionsValue(false, false, 0, 0, 0)
	f.Add(cellOptionsPack(bits))
	_, bits = cellOptionsValue(true, true, 1, 1, 7)
	f.Add(cellOptionsPack(bits))
	f.Fuzz(func(t *testing.T, wire []byte) {
		if len(wire) > 2048 {
			return
		}
		v, err := DecodeNonGPRSCellOptionsIE(wire)
		if err != nil {
			return
		}
		got, err := EncodeNonGPRSCellOptionsIE(v.Value)
		if err != nil || !bytes.Equal(got, wire) {
			t.Fatalf("replay %x want %x: %v", got, wire, err)
		}
	})
}

func TestNonGPRSCellOptionsInputGrowth(t *testing.T) {
	_, text := cellOptionsValue(true, true, 1, 1, 63)
	seed := cellOptionsPack(text)
	for _, n := range []int{64, 256, 1024, 4096, 8192} {
		wire := append(bytes.Clone(seed), bytes.Repeat([]byte{0xa5}, n)...)
		old := debug.SetGCPercent(-1)
		var before, after goruntime.MemStats
		goruntime.ReadMemStats(&before)
		for i := 0; i < 4; i++ {
			v, err := DecodeNonGPRSCellOptionsIE(wire)
			if err != nil {
				debug.SetGCPercent(old)
				t.Fatal(err)
			}
			if v.Value.ExtensionLengthGroup.Content.Known == nil {
				debug.SetGCPercent(old)
				t.Fatal("typed extension lost")
			}
		}
		goruntime.ReadMemStats(&after)
		debug.SetGCPercent(old)
		if used := (after.TotalAlloc - before.TotalAlloc) / 4; !raceEnabled && used > uint64(len(wire))*256+262144 {
			t.Fatalf("nonlinear storage: %d input, %d allocated", len(wire), used)
		}
	}
}
func FuzzExtensionInformationReplay(f *testing.F) {
	f.Add([]byte{0})
	f.Add([]byte{0xff})
	f.Add(bytes.Repeat([]byte{0xa5}, 8))
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 4096 {
			return
		}
		raw, err := DecodeExtensionInformation(data)
		if err != nil {
			return
		}
		out, err := EncodeExtensionInformation(raw.Value)
		if err != nil || !bytes.Equal(out, data) {
			t.Fatalf("extension replay %x: %v", out, err)
		}
	})
}
