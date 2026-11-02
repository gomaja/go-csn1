package ies

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

var priorityParametersCodec = clauseCodec[N3GPriorityParametersIE]{DecodeN3GPriorityParametersIE, DecodeN3GPriorityParametersIEFrom, EncodeN3GPriorityParametersIE, EncodeN3GPriorityParametersIETo, EncodeN3GPriorityParametersIECanonicalAtLength}

// TS 44.060 V19.0.0 Table 12.52.1, paragraphs 25319–25329:
// optional defaults; terminated parameter/frequency lists; three nested options.
func priorityParametersVector(defaults bool, mask, count int) (N3GPriorityParametersIE, string) {
	var v N3GPriorityParametersIE
	var bits strings.Builder
	if defaults {
		v.DEFAULTUTRANPRIORITYGroup = &N3GPriorityParametersIEDEFAULTUTRANPRIORITYGroup{DEFAULTUTRANPRIORITY: 7, DEFAULTTHRESHUTRAN: 21, DEFAULTUTRANQRXLEVMIN: 31}
		bits.WriteString("11111010111111")
	} else {
		bits.WriteByte('0')
	}
	for i := 0; i < count; i++ {
		bits.WriteByte('1')
		p := RepeatedUTRANPriorityParametersStruct{THRESHUTRANHigh: uint8(i & 31)}
		for j := 0; j < i%3; j++ {
			n := uint8((i + j) & 31)
			p.UTRANFREQUENCYINDEXList = append(p.UTRANFREQUENCYINDEXList, RepeatedUTRANPriorityParametersStructUTRANFREQUENCYINDEXListEntry{UTRANFREQUENCYINDEX: n})
			fmt.Fprintf(&bits, "1%05b", n)
		}
		bits.WriteByte('0')
		if mask&1 != 0 {
			x := uint8(7)
			p.UTRANPRIORITY = &x
			bits.WriteString("1111")
		} else {
			bits.WriteByte('0')
		}
		fmt.Fprintf(&bits, "%05b", p.THRESHUTRANHigh)
		if mask&2 != 0 {
			x := uint8(21)
			p.THRESHUTRANLow = &x
			bits.WriteString("110101")
		} else {
			bits.WriteByte('0')
		}
		if mask&4 != 0 {
			x := uint8(31)
			p.UTRANQRXLEVMIN = &x
			bits.WriteString("111111")
		} else {
			bits.WriteByte('0')
		}
		v.RepeatedUTRANPriorityParametersList = append(v.RepeatedUTRANPriorityParametersList, N3GPriorityParametersIERepeatedUTRANPriorityParametersListEntry{RepeatedUTRANPriorityParameters: p})
	}
	bits.WriteByte('0')
	return v, bits.String()
}
func TestPriorityParameters(t *testing.T) {
	for _, defaults := range []bool{false, true} {
		for mask := 0; mask < 8; mask++ {
			for _, count := range []int{0, 1, 3} {
				t.Run(fmt.Sprintf("defaults-%t-options-%d-count-%d", defaults, mask, count), func(t *testing.T) {
					v, bits := priorityParametersVector(defaults, mask, count)
					wire := clausePack(bits)
					checkClauseVector(t, priorityParametersCodec, clauseVector[N3GPriorityParametersIE]{hex: hex.EncodeToString(wire), bits: len(bits), value: v}, len(bits))
					d, err := DecodeN3GPriorityParametersIE(wire)
					if err != nil || d.BitsConsumed != len(bits) {
						t.Fatalf("extent %d want %d: %v", d.BitsConsumed, len(bits), err)
					}
					if count > 0 {
						d.Value.RepeatedUTRANPriorityParametersList[0].RepeatedUTRANPriorityParameters.THRESHUTRANHigh = 31
						out, err := EncodeN3GPriorityParametersIE(d.Value)
						if err != nil {
							t.Fatal(err)
						}
						again, err := DecodeN3GPriorityParametersIE(out)
						if err != nil || again.Value.RepeatedUTRANPriorityParametersList[0].RepeatedUTRANPriorityParameters.THRESHUTRANHigh != 31 {
							t.Fatalf("nested edit: %+v %v", again, err)
						}
					}
				})
			}
		}
	}
	invalid := N3GPriorityParametersIE{DEFAULTUTRANPRIORITYGroup: &N3GPriorityParametersIEDEFAULTUTRANPRIORITYGroup{DEFAULTUTRANPRIORITY: 8}}
	checkClauseInvalid(t, priorityParametersCodec, invalid, 8)
	v, _ := priorityParametersVector(false, 7, 1)
	v.RepeatedUTRANPriorityParametersList[0].RepeatedUTRANPriorityParameters.THRESHUTRANHigh = 32
	checkClauseInvalid(t, priorityParametersCodec, v, 8)
}
func TestPriorityParametersGrowth(t *testing.T) {
	for _, count := range []int{1, 4, 16, 64, 256} {
		v, bits := priorityParametersVector(true, 7, count)
		wire := clausePack(bits)
		d, err := DecodeN3GPriorityParametersIE(wire)
		if err != nil || len(d.Value.RepeatedUTRANPriorityParametersList) != count || d.BitsConsumed != len(bits) {
			t.Fatalf("count %d: %v", count, err)
		}
		if len(wire) > 8*count+2 {
			t.Fatalf("nonlinear wire size %d for count %d", len(wire), count)
		}
		allocations := testing.AllocsPerRun(1, func() {
			_, decodeErr := DecodeN3GPriorityParametersIE(wire)
			if decodeErr != nil {
				panic(decodeErr)
			}
		})
		if !raceEnabled && allocations > float64(120*count+200) {
			t.Fatalf("nonlinear allocation count %g for %d", allocations, count)
		}
		if !runtime.SemanticallyEqual(d.Value, v) {
			t.Fatalf("typed list changed for count %d", count)
		}
	}
}

// The same table's local helper remains directly typed, including present zeroes.
func TestRepeatedPriorityParameterFields(t *testing.T) {
	codec := clauseCodec[RepeatedUTRANPriorityParametersStruct]{DecodeRepeatedUTRANPriorityParametersStruct, DecodeRepeatedUTRANPriorityParametersStructFrom, EncodeRepeatedUTRANPriorityParametersStruct, EncodeRepeatedUTRANPriorityParametersStructTo, EncodeRepeatedUTRANPriorityParametersStructCanonicalAtLength}
	for mask := 0; mask < 8; mask++ {
		for _, value := range []uint8{0, 1, 21, 31} {
			t.Run(fmt.Sprintf("options-%d-value-%d", mask, value), func(t *testing.T) {
				v := RepeatedUTRANPriorityParametersStruct{UTRANFREQUENCYINDEXList: []RepeatedUTRANPriorityParametersStructUTRANFREQUENCYINDEXListEntry{{UTRANFREQUENCYINDEX: value}}, THRESHUTRANHigh: value}
				var bits strings.Builder
				fmt.Fprintf(&bits, "1%05b0", value)
				if mask&1 != 0 {
					x := value & 7
					v.UTRANPRIORITY = &x
					fmt.Fprintf(&bits, "1%03b", x)
				} else {
					bits.WriteByte('0')
				}
				fmt.Fprintf(&bits, "%05b", value)
				if mask&2 != 0 {
					x := value
					v.THRESHUTRANLow = &x
					fmt.Fprintf(&bits, "1%05b", x)
				} else {
					bits.WriteByte('0')
				}
				if mask&4 != 0 {
					x := value
					v.UTRANQRXLEVMIN = &x
					fmt.Fprintf(&bits, "1%05b", x)
				} else {
					bits.WriteByte('0')
				}
				checkClauseVector(t, codec, clauseVector[RepeatedUTRANPriorityParametersStruct]{hex: hex.EncodeToString(clausePack(bits.String())), bits: bits.Len(), value: v}, bits.Len())
			})
		}
	}
	for _, field := range []string{"frequency", "priority", "high", "low", "minimum"} {
		t.Run("overflow-"+field, func(t *testing.T) {
			v := RepeatedUTRANPriorityParametersStruct{}
			bad := uint8(32)
			switch field {
			case "frequency":
				v.UTRANFREQUENCYINDEXList = []RepeatedUTRANPriorityParametersStructUTRANFREQUENCYINDEXListEntry{{UTRANFREQUENCYINDEX: 32}}
			case "priority":
				bad = 8
				v.UTRANPRIORITY = &bad
			case "high":
				v.THRESHUTRANHigh = 32
			case "low":
				v.THRESHUTRANLow = &bad
			case "minimum":
				v.UTRANQRXLEVMIN = &bad
			}
			checkClauseInvalid(t, codec, v, 4)
		})
	}
}

func TestPriorityParametersRequiresCanonicalExtent(t *testing.T) {
	_, err := EncodeN3GPriorityParametersIECanonical(N3GPriorityParametersIE{})
	var extent *runtime.ExtentError
	if !errors.As(err, &extent) || !extent.Required {
		t.Fatalf("required canonical extent: %v", err)
	}
}
