package controlblock

import "github.com/gomaja/go-csn1/runtime"

// BlockFormat selects the logical block fields of §10.3, excluding physical
// coding, channel spare bits and EC's separately encoded USF.
type BlockFormat uint8

const (
	CS1 BlockFormat = iota + 1
	CS3
	ECPACCH
)

// BlockHeader contains exactly the header for the selected block format.
// All pointers are nil for EC-PACCH/U, which has no MAC header.
type BlockHeader struct {
	Context    runtime.RLCMACContext
	Downlink   *DownlinkHeader
	Uplink     *UplinkHeader
	ECDownlink *ECDownlinkHeader
}

// DecodeBlockHeaderFrom requires an exact containing logical block extent and
// leaves its contents on the SAME reader. TS 44.060 V19.0.0 §10.0b.2.1 defines
// the 176/304/75/67/64-bit fields; figures 10.3.1.1.1, 10.3.1.3.1/2 and
// 10.3.2.1/2 define the headers. Optional ordinary octets occupy part of the
// 176-bit field. CS-3 is uplink only (§10.3), before channel spare bits.
func DecodeBlockHeaderFrom(r *runtime.Reader, ctx runtime.RLCMACContext, format BlockFormat) (BlockHeader, error) {
	var h BlockHeader
	if err := ctx.Validate(); err != nil {
		return h, err
	}
	if err := r.Check(); err != nil {
		return h, err
	}
	if ctx.Mode == runtime.RLCMACDBPSCH {
		return h, unsupportedCarrier()
	}
	total := 184
	coding := runtime.RLCMACCS1
	switch format {
	case CS1:
		if ctx.Mode != runtime.RLCMACOrdinary {
			return h, headerError(r, "CS-1 block requires ordinary mode")
		}
	case CS3:
		if ctx.Mode != runtime.RLCMACOrdinary || ctx.Direction != runtime.RLCMACUplink {
			return h, headerError(r, "CS-3 control format requires ordinary uplink")
		}
		total = 312
		coding = runtime.RLCMACCS3
	case ECPACCH:
		if ctx.Mode != runtime.RLCMACEC {
			return h, headerError(r, "EC-PACCH requires EC mode")
		}
		total = 80
		coding = runtime.RLCMACECCoding
		if ctx.Direction == runtime.RLCMACUplink {
			total = 64
		}
	default:
		return h, headerError(r, "unsupported block format")
	}
	if ctx.CodingScheme != 0 && ctx.CodingScheme != coding {
		return h, headerError(r, "coding context disagrees with the selected block format")
	}
	ctx.CodingScheme = coding
	h.Context = ctx
	if r.BoundedRemaining() != total {
		return h, headerError(r, "containing extent differs from the logical control block")
	}
	if format == ECPACCH {
		if ctx.Direction == runtime.RLCMACDownlink {
			v, err := DecodeECDownlinkHeader(r)
			if err != nil {
				return h, err
			}
			h.ECDownlink = &v
		}
		return h, nil
	}
	if ctx.Direction == runtime.RLCMACDownlink {
		v, err := DecodeDownlinkHeader(r)
		if err != nil {
			return h, err
		}
		h.Downlink = &v
	} else {
		v, err := DecodeUplinkHeader(r)
		if err != nil {
			return h, err
		}
		h.Uplink = &v
	}
	return h, nil
}

func unsupportedCarrier() error {
	return &runtime.ContextRequiredError{Standard: "TS 44.060 V19.0.0", Clause: "11.1.1.3", Name: "DBPSCH physical carrier framing"}
}

// DownlinkMAC is the CS-1 MAC octet, TS 44.060 V19.0.0
// §10.3.1.1 figure 10.3.1.1.1. Values retain their transmitted coding.
type DownlinkMAC struct {
	PayloadType uint64 // §10.4.7 table 10.4.7.1, two bits: 01 or 10.
	RRBP        uint64 // §10.4.5, two bits.
	SP          uint64 // §10.4.4, one bit.
	USF         uint64 // §10.4.1, three bits.
}

// ControlHeader is the optional octet and its conditional following octets,
// TS 44.060 V19.0.0 §10.3.1.1 figure 10.3.1.1.1 and
// §10.3.1.2 figure 10.3.1.2.1. Address presence carries the AC bit.
type ControlHeader struct {
	RBSN      uint64            // §10.4.12a, one bit.
	RTI       uint64            // §10.4.9c, five bits.
	FS        uint64            // §10.4.9b, one bit.
	Address   *ControlAddress   // §10.4.9a AC: nil = 0, non-nil = 1.
	Extension *ControlExtension // Present exactly when RBSN=1 and FS=0.
}

// ControlAddress is the TFI/D octet of figure 10.3.1.1.1.
type ControlAddress struct {
	PR  uint64 // §10.4.10a, two bits.
	TFI uint64 // §10.4.10, five bits.
	D   uint64 // §10.4.9d, one bit: 0 uplink TBF, 1 downlink TBF.
}

// ControlExtension is the extended-segmentation octet of figure 10.3.1.1.1.
type ControlExtension struct {
	RBSNe    uint64 // §10.4.12b, three bits.
	FSe      uint64 // §10.4.9e, one bit.
	Spare    uint64 // Figure 10.3.1.1.1, four received spare bits, retained for replay.
	received bool
}

// DownlinkHeader includes the optional control header exactly for payload 10.
type DownlinkHeader struct {
	MAC     DownlinkMAC
	Control *ControlHeader
}

// UplinkHeader is figure 10.3.2.1, TS 44.060 V19.0.0 §10.3.2.
// Payload Type is 01 (§10.4.7); received spare bits replay, fresh sender
// spare bits are zero (§10.1); R is §10.4.2.
type UplinkHeader struct {
	Spare, R uint64
	received bool
}

// ECDownlinkHeader excludes the separately encoded USF (§10.3.1.3).
// Figure 10.3.1.3.1 defines PR(1), Payload Type(1), RRBP(2), S/P(1).
// Extended presence carries Payload Type, §10.4.7 table 10.4.7.3.
type ECDownlinkHeader struct {
	PR       uint64 // §10.4.10a table 10.4.10a.2, one bit.
	RRBP     uint64 // §10.4.5, two bits.
	SP       uint64 // §10.4.4, one bit.
	Extended *ECExtendedHeader
}

// ECExtendedHeader follows the first five bits in figure 10.3.1.3.2.
type ECExtendedHeader struct {
	PRe  uint64 // §10.4.10b, one bit.
	RBSN uint64 // §10.4.12a, one bit.
	FS   uint64 // §10.4.9b, one bit.
	TFI  uint64 // §10.4.10, five bits.
}

type headerField struct {
	name  string
	width int
	value *uint64
}

func readHeaderFields(r *runtime.Reader, fields ...headerField) error {
	for _, f := range fields {
		if err := r.Enter(f.name); err != nil {
			return err
		}
		v, err := r.ReadUint(f.width)
		r.Leave()
		if err != nil {
			return err
		}
		*f.value = v
	}
	return nil
}

func writeHeaderFields(w *runtime.Writer, fields ...headerField) error {
	for _, f := range fields {
		if err := w.WriteUint(*f.value, f.width); err != nil {
			return err
		}
	}
	return nil
}

func headerError(r *runtime.Reader, detail string) error {
	return r.Error(runtime.InvalidValue, "TS 44.060 V19.0.0 §10.3: "+detail)
}

func invalidHeader(detail string) error {
	return &runtime.DecodeError{Kind: runtime.InvalidValue, Path: "controlblock", Detail: "TS 44.060 V19.0.0 §10.3: " + detail}
}

// DecodeDownlinkHeader consumes only the CS-1 header on the supplied reader.
// The caller retains the reader and its original bit coordinates for the body.
func DecodeDownlinkHeader(r *runtime.Reader) (DownlinkHeader, error) {
	var h DownlinkHeader
	if err := r.Check(); err != nil {
		return h, err
	}
	m := &h.MAC
	if err := readHeaderFields(r, headerField{"PayloadType", 2, &m.PayloadType}, headerField{"RRBP", 2, &m.RRBP}, headerField{"SP", 1, &m.SP}, headerField{"USF", 3, &m.USF}); err != nil {
		return DownlinkHeader{}, err
	}
	if m.PayloadType != 1 && m.PayloadType != 2 {
		return DownlinkHeader{}, headerError(r, "payload is not an implemented control header (§10.4.7 table 10.4.7.1)")
	}
	if m.PayloadType == 2 {
		c, err := DecodeControlHeader(r)
		if err != nil {
			return DownlinkHeader{}, err
		}
		h.Control = &c
	}
	return h, nil
}

// DecodeControlHeader reads the optional control octets of figures
// 10.3.1.1.1/10.3.1.2.1 after their presence has been established by the MAC.
// MCS-0 physical field mapping (§10.0b.3.2) must be supplied by its carrier.
func DecodeControlHeader(r *runtime.Reader) (ControlHeader, error) {
	var c ControlHeader
	var ac uint64
	if err := r.Check(); err != nil {
		return c, err
	}
	if err := readHeaderFields(r, headerField{"RBSN", 1, &c.RBSN}, headerField{"RTI", 5, &c.RTI}, headerField{"FS", 1, &c.FS}, headerField{"AC", 1, &ac}); err != nil {
		return ControlHeader{}, err
	}
	if ac == 1 {
		a := new(ControlAddress)
		if err := readHeaderFields(r, headerField{"PR", 2, &a.PR}, headerField{"TFI", 5, &a.TFI}, headerField{"D", 1, &a.D}); err != nil {
			return ControlHeader{}, err
		}
		c.Address = a
	}
	if c.RBSN == 1 && c.FS == 0 {
		e := new(ControlExtension)
		if err := readHeaderFields(r, headerField{"RBSNe", 3, &e.RBSNe}, headerField{"FSe", 1, &e.FSe}, headerField{"Spare", 4, &e.Spare}); err != nil {
			return ControlHeader{}, err
		}
		c.Extension = e
		e.received = true
	}
	return c, nil
}

// EncodeDownlinkHeader writes figure 10.3.1.1.1, preserving received spare bits.
func EncodeDownlinkHeader(w *runtime.Writer, h DownlinkHeader) error {
	if (h.MAC.PayloadType != 1 && h.MAC.PayloadType != 2) || (h.MAC.PayloadType == 2) != (h.Control != nil) {
		return invalidHeader("payload type and optional control octet disagree")
	}
	m := &h.MAC
	if err := writeHeaderFields(w, headerField{"PayloadType", 2, &m.PayloadType}, headerField{"RRBP", 2, &m.RRBP}, headerField{"SP", 1, &m.SP}, headerField{"USF", 3, &m.USF}); err != nil {
		return err
	}
	if h.Control != nil {
		return EncodeControlHeader(w, *h.Control)
	}
	return nil
}

// CanonicalDownlinkHeader writes a fresh header with zero spare bits,
// TS 44.060 V19.0.0 §10.1. EncodeDownlinkHeader instead replays received bits.
func CanonicalDownlinkHeader(w *runtime.Writer, h DownlinkHeader) error {
	if h.Control != nil {
		c := *h.Control
		if c.Extension != nil {
			e := *c.Extension
			e.Spare = 0
			c.Extension = &e
		}
		h.Control = &c
	}
	return EncodeDownlinkHeader(w, h)
}

// EncodeControlHeader writes the control octets in logical figure order.
func EncodeControlHeader(w *runtime.Writer, c ControlHeader) error {
	if (c.RBSN == 1 && c.FS == 0) != (c.Extension != nil) {
		return invalidHeader("RBSN/FS and extension octet disagree")
	}
	var ac uint64
	if c.Address != nil {
		ac = 1
	}
	if err := writeHeaderFields(w, headerField{"RBSN", 1, &c.RBSN}, headerField{"RTI", 5, &c.RTI}, headerField{"FS", 1, &c.FS}, headerField{"AC", 1, &ac}); err != nil {
		return err
	}
	if a := c.Address; a != nil {
		if err := writeHeaderFields(w, headerField{"PR", 2, &a.PR}, headerField{"TFI", 5, &a.TFI}, headerField{"D", 1, &a.D}); err != nil {
			return err
		}
	}
	if e := c.Extension; e != nil {
		spare := uint64(0)
		if e.received {
			spare = e.Spare
		} // §10.1: fresh sender zero, received replay exact.
		return writeHeaderFields(w, headerField{"RBSNe", 3, &e.RBSNe}, headerField{"FSe", 1, &e.FSe}, headerField{"Spare", 4, &spare})
	}
	return nil
}

// CanonicalControlHeader applies §10.1 to the optional control octets.
func CanonicalControlHeader(w *runtime.Writer, c ControlHeader) error {
	if c.Extension != nil {
		e := *c.Extension
		e.Spare = 0
		c.Extension = &e
	}
	return EncodeControlHeader(w, c)
}

// DecodeUplinkHeader reads figure 10.3.2.1. EC-PACCH/U has no MAC header
// (§10.3.2 figure 10.3.2.2) and must not call this entry point.
func DecodeUplinkHeader(r *runtime.Reader) (UplinkHeader, error) {
	var h UplinkHeader
	var pt uint64
	if err := r.Check(); err != nil {
		return h, err
	}
	if err := readHeaderFields(r, headerField{"PayloadType", 2, &pt}, headerField{"Spare", 5, &h.Spare}, headerField{"R", 1, &h.R}); err != nil {
		return UplinkHeader{}, err
	}
	if pt != 1 {
		return UplinkHeader{}, headerError(r, "uplink payload type must be 01 (§10.4.7 table 10.4.7.1)")
	}
	h.received = true
	return h, nil
}

// EncodeUplinkHeader writes figure 10.3.2.1.
func EncodeUplinkHeader(w *runtime.Writer, h UplinkHeader) error {
	pt := uint64(1)
	spare := uint64(0)
	if h.received {
		spare = h.Spare
	} // §10.1.
	return writeHeaderFields(w, headerField{"PayloadType", 2, &pt}, headerField{"Spare", 5, &spare}, headerField{"R", 1, &h.R})
}

// CanonicalUplinkHeader writes the sender's zero spare bits (§10.1).
func CanonicalUplinkHeader(w *runtime.Writer, h UplinkHeader) error {
	h.Spare = 0
	return EncodeUplinkHeader(w, h)
}

// DecodeECDownlinkHeader reads the five or thirteen bits in figures
// 10.3.1.3.1/2, without resetting the body's L/H origin.
func DecodeECDownlinkHeader(r *runtime.Reader) (ECDownlinkHeader, error) {
	var h ECDownlinkHeader
	var pt uint64
	if err := r.Check(); err != nil {
		return h, err
	}
	if err := readHeaderFields(r, headerField{"PR", 1, &h.PR}, headerField{"PayloadType", 1, &pt}, headerField{"RRBP", 2, &h.RRBP}, headerField{"SP", 1, &h.SP}); err != nil {
		return ECDownlinkHeader{}, err
	}
	if pt == 1 {
		e := new(ECExtendedHeader)
		if err := readHeaderFields(r, headerField{"PRe", 1, &e.PRe}, headerField{"RBSN", 1, &e.RBSN}, headerField{"FS", 1, &e.FS}, headerField{"TFI", 5, &e.TFI}); err != nil {
			return ECDownlinkHeader{}, err
		}
		h.Extended = e
	}
	return h, nil
}

// EncodeECDownlinkHeader writes figures 10.3.1.3.1/2.
func EncodeECDownlinkHeader(w *runtime.Writer, h ECDownlinkHeader) error {
	var pt uint64
	if h.Extended != nil {
		pt = 1
	}
	if err := writeHeaderFields(w, headerField{"PR", 1, &h.PR}, headerField{"PayloadType", 1, &pt}, headerField{"RRBP", 2, &h.RRBP}, headerField{"SP", 1, &h.SP}); err != nil {
		return err
	}
	if e := h.Extended; e != nil {
		return writeHeaderFields(w, headerField{"PRe", 1, &e.PRe}, headerField{"RBSN", 1, &e.RBSN}, headerField{"FS", 1, &e.FS}, headerField{"TFI", 5, &e.TFI})
	}
	return nil
}
