package controlblock

import (
	"fmt"

	"github.com/gomaja/go-csn1/runtime"
)

// Segment contains a complete control-message contents field, including final
// filler octets. Header is the ordinary optional header; EC is the EC extended
// header. Both nil denotes an unsegmented block. It excludes the MAC octet.
// TS 44.060 V19.0.0 §§9.1.12a–b, 10.3.1.
type Segment struct {
	Format   BlockFormat
	Header   *ControlHeader
	EC       *ECExtendedHeader
	Contents runtime.BitString
}

// SegmentError locates a rejected block and the offending bit within that
// block's contents or control header. It does not request a protocol action.
type SegmentError struct {
	Block, Offset  int
	Clause, Detail string
	Cause          error
}

func (e *SegmentError) Error() string {
	return fmt.Sprintf("TS 44.060 V19.0.0 §%s block %d bit %d: %s", e.Clause, e.Block, e.Offset, e.Detail)
}

func (e *SegmentError) Unwrap() error { return e.Cause }

// ReassembledMessage owns concatenated message bits and the first contents
// field's original bit coordinate. Use NewReader for decoding so removing the
// header cannot silently change L/H phase. BitString alone is a logical view.
type ReassembledMessage struct {
	runtime.BitString
	OriginBits  int
	ecMultipart bool
}

// NewReader restores the original first-field coordinate without exposing a
// fabricated header as received data. For multipart EC, the source mapping of
// L/H across 67-bit fragments remains unresolved: TS 44.060 V19.0.0
// §§10.0b.2.1, 11 and TS 24.007 V20.0.0 Annex B §B.1.2.2 do not explicitly
// map that boundary. Return a located context error pending that decision.
func (m ReassembledMessage) NewReader(c *runtime.Collector) (*runtime.Reader, error) {
	if m.ecMultipart {
		return nil, &SegmentError{Block: 1, Offset: 0, Clause: "10.0b.2.1; 11", Detail: "EC multipart L/H origin across 67-bit fields requires a source decision", Cause: runtime.ErrContextRequired}
	}
	if m.OriginBits < 0 || m.OriginBits > 32 || m.BitLength < 0 || m.BitLength > 9*176 {
		return nil, invalidHeader("invalid reassembled message coordinates")
	}
	w := runtime.NewWriter()
	if err := w.WriteUint(0, m.OriginBits); err != nil {
		return nil, err
	}
	if err := w.WriteBitString(m.BitString, m.BitLength); err != nil {
		return nil, err
	}
	bits, err := w.FinishBits(runtime.BitString{})
	if err != nil {
		return nil, err
	}
	r, err := runtime.NewReaderBits(bits.Bytes, bits.BitLength, c)
	if err != nil {
		return nil, err
	}
	if _, err = r.ReadUint(m.OriginBits); err != nil {
		return nil, err
	}
	return r, nil
}

func segmentFailure(block, offset int, clause, detail string) error {
	return &SegmentError{Block: block, Offset: offset, Clause: clause, Detail: detail}
}

// segmentationMaximum keeps PSI/container continuation separate from control
// block segmentation. These clauses explicitly forbid splitting a message.
func segmentationMaximum(ctx runtime.RLCMACContext, clause string, blocks int) (int, error) {
	if ctx.Direction == runtime.RLCMACUplink {
		return 1, nil
	} // §9.1.12a.
	switch clause {
	case "11.2.9e", "11.2.17a", "11.2.17c", "11.2.18", "11.2.19", "11.2.20", "11.2.21", "11.2.21a", "11.2.21b", "11.2.23", "11.2.23a", "11.2.23b", "11.2.24", "11.2.25", "11.2.25c", "11.2.40":
		return 1, nil
	case "11.2.36", "11.2.43", "11.2.46", "11.2.47":
		if ctx.Mode != runtime.RLCMACEC {
			return 9, nil
		}
	case "11.2.7", "11.2.7a", "11.2.29", "11.2.29a", "11.2.31", "11.2.31a", "11.2.37":
		// TS 44.060 V19.0.0 §§11.2.7, 11.2.29 and 11.2.31 also
		// permit Downlink Multi Carrier support. Missing capability is not false.
		if blocks > 2 && ctx.Mode != runtime.RLCMACEC {
			multi := clause == "11.2.7" || clause == "11.2.29" || clause == "11.2.31"
			if ctx.DLDC != nil && *ctx.DLDC || multi && ctx.DLMC != nil && *ctx.DLMC {
				return 9, nil
			}
			if ctx.DLDC == nil || multi && ctx.DLMC == nil {
				return 0, &runtime.ContextRequiredError{Standard: "TS 44.060 V19.0.0", Clause: clause, Name: "extended segmentation dual-carrier capability"}
			}
		}
	}
	return 2, nil // §9.1.12a: extended segmentation requires explicit permission.
}

// Reassemble concatenates an ordered complete set for one message. The caller
// groups blocks by radio transaction and PDCH (§9.1.12b). This function neither
// sorts nor deduplicates; missing, repeated, out-of-order or mixed-RTI blocks
// return located errors. At most nine 176-bit downlink fields or one 304-bit uplink field are examined.
// Filler is retained: only the message codec can identify its semantic end.
// No timer, acknowledgement, polling, TFI admission or eviction is performed.
func Reassemble(ctx runtime.RLCMACContext, clause string, segments []Segment) (ReassembledMessage, error) {
	var zero ReassembledMessage
	if err := ctx.Validate(); err != nil {
		return zero, err
	}
	if clause == "" {
		return zero, &runtime.ContextRequiredError{Standard: "TS 44.060 V19.0.0", Clause: "9.1.12a", Name: "message clause"}
	}
	if len(segments) == 0 || len(segments) > 9 {
		return zero, segmentFailure(0, 0, "9.1.12a", "expected one to nine complete blocks")
	}
	maximum, err := segmentationMaximum(ctx, clause, len(segments))
	if err != nil {
		return zero, err
	}
	if len(segments) > maximum {
		return zero, segmentFailure(0, 0, clause, "segmentation exceeds this message's source permission")
	}
	for i, s := range segments {
		if s.Format == CS3 && clause != "11.2.48" {
			return zero, segmentFailure(i, 0, "10.3", "CS-3 control contents are defined only for §11.2.48")
		}
		capacity, err := segmentCapacity(ctx, s)
		if err != nil {
			return zero, segmentFailure(i, 0, "10.3", err.Error())
		}
		if s.Contents.BitLength != capacity || len(s.Contents.Bytes) != (capacity+7)/8 {
			return zero, segmentFailure(i, 0, "10.0b.2.1", "contents must contain the exact block field including final filler")
		}
		if ctx.Mode == runtime.RLCMACEC {
			if err := checkECSequence(i, len(segments), s.EC); err != nil {
				return zero, err
			}
		} else if err := checkSequence(i, len(segments), s.Header, segments[0].Header); err != nil {
			return zero, err
		}
	}
	origin := 8
	first := segments[0]
	if ctx.Mode == runtime.RLCMACEC {
		origin = 5
		if first.EC != nil {
			origin = 13
		}
		if ctx.Direction == runtime.RLCMACUplink {
			origin = 0
		}
	} else if first.Header != nil {
		w := runtime.NewWriter()
		if err := EncodeControlHeader(w, *first.Header); err != nil {
			return zero, err
		}
		origin += w.Position()
	}
	w := runtime.NewWriter()
	for _, s := range segments {
		if err := w.WriteBitString(s.Contents, s.Contents.BitLength); err != nil {
			return zero, err
		}
	}
	bits, err := w.FinishBits(runtime.BitString{})
	if err != nil {
		return zero, err
	}
	return ReassembledMessage{BitString: bits, OriginBits: origin, ecMultipart: ctx.Mode == runtime.RLCMACEC && len(segments) > 1}, nil
}

func segmentCapacity(ctx runtime.RLCMACContext, s Segment) (int, error) {
	coding := runtime.RLCMACCS1
	switch s.Format {
	case CS1:
	case CS3:
		coding = runtime.RLCMACCS3
	case ECPACCH:
		coding = runtime.RLCMACECCoding
	default:
		return 0, invalidHeader("unsupported segment format")
	}
	if ctx.CodingScheme != 0 && ctx.CodingScheme != coding {
		return 0, invalidHeader("coding context disagrees with segment format")
	}
	if ctx.Mode == runtime.RLCMACDBPSCH {
		return 0, unsupportedCarrier()
	}
	if ctx.Mode == runtime.RLCMACEC {
		if s.Format != ECPACCH {
			return 0, invalidHeader("EC segment requires EC-PACCH format")
		}
		if s.Header != nil {
			return 0, invalidHeader("ordinary header in EC segment")
		}
		if ctx.Direction == runtime.RLCMACUplink {
			if s.EC != nil {
				return 0, invalidHeader("EC uplink has no MAC header")
			}
			return 64, nil
		}
		if s.EC != nil {
			w := runtime.NewWriter()
			if err := EncodeECDownlinkHeader(w, ECDownlinkHeader{Extended: s.EC}); err != nil {
				return 0, err
			}
			return 67, nil
		}
		return 75, nil
	}
	if s.Format != CS1 && s.Format != CS3 {
		return 0, invalidHeader("ordinary segment requires CS-1 or CS-3 format")
	}
	if s.Format == CS3 {
		if ctx.Direction != runtime.RLCMACUplink || s.Header != nil || s.EC != nil {
			return 0, invalidHeader("CS-3 requires ordinary uplink without optional control header")
		}
		return 304, nil
	}
	if s.EC != nil {
		return 0, invalidHeader("EC header in ordinary segment")
	}
	if ctx.Direction == runtime.RLCMACUplink && s.Header != nil {
		return 0, invalidHeader("uplink has no optional control header")
	}
	capacity := 176
	if s.Header != nil {
		w := runtime.NewWriter()
		if err := EncodeControlHeader(w, *s.Header); err != nil {
			return 0, err
		}
		capacity -= w.Position()
	}
	return capacity, nil
}

func checkSequence(i, count int, c, first *ControlHeader) error {
	if c == nil {
		if count != 1 {
			return segmentFailure(i, 0, "9.1.12a", "segmented block requires control header")
		}
		return nil
	}
	if first == nil || c.RTI != first.RTI {
		return segmentFailure(i, 1, "10.4.9c", "RTI differs within one message")
	}
	if i == 0 && c.RBSN != 0 || i > 0 && c.RBSN != 1 {
		return segmentFailure(i, 0, "10.4.12a", "unexpected RBSN order")
	}
	if count <= 2 {
		want := uint64(0)
		if i == count-1 {
			want = 1
		}
		if c.FS != want || c.Extension != nil {
			return segmentFailure(i, 6, "10.4.9b", "FS does not identify the final segment")
		}
		return nil
	}
	if c.FS != 0 {
		return segmentFailure(i, 6, "10.4.12b", "extended segmentation requires FS=0")
	}
	if i == 0 {
		if c.Extension != nil {
			return segmentFailure(i, 0, "10.4.12b", "first block has no extension")
		}
		return nil
	}
	want := uint64(0)
	if i == count-1 {
		want = 1
	}
	if c.Extension == nil || c.Extension.RBSNe != uint64(i-1) || c.Extension.FSe != want {
		return segmentFailure(i, 0, "10.4.12b", "RBSNe/FSe does not match complete ordered sequence")
	}
	return nil
}

func checkECSequence(i, count int, e *ECExtendedHeader) error {
	if e == nil {
		if count != 1 {
			return segmentFailure(i, 0, "10.3.1.3", "segmented EC block requires extended header")
		}
		return nil
	}
	wantRBSN := uint64(0)
	if i > 0 {
		wantRBSN = 1
	}
	wantFS := uint64(0)
	if i == count-1 {
		wantFS = 1
	}
	if e.RBSN != wantRBSN || e.FS != wantFS {
		return segmentFailure(i, 6, "10.4.12a", "EC RBSN/FS does not match complete sequence")
	}
	return nil
}
