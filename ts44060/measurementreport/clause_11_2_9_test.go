package measurementreport_test

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44060/controlblock"
	"github.com/gomaja/go-csn1/ts44060/measurementreport"
)

type measurementCase struct {
	name                                       string
	release, nc, n3g, ne, ba, psi5, grnti, csg int
	bsic, plmn                                 bool
}

func measurementCases() []measurementCase {
	out := []measurementCase{}
	add := func(name string, rel int) *measurementCase {
		grnti := -1
		if rel >= 3 {
			grnti = 15
		}
		out = append(out, measurementCase{name: name, release: rel, nc: 1, psi5: -1, grnti: grnti})
		return &out[len(out)-1]
	}
	for rel := 0; rel <= 5; rel++ {
		add(fmt.Sprintf("release%d", rel), rel)
	}
	for _, n := range []int{0, 7} {
		c := add(fmt.Sprintf("nc%d", n), 0)
		c.nc = n
	}
	c := add("nc-bsic", 0)
	c.nc = 2
	c.bsic = true
	c.psi5 = 3
	for _, ba := range []int{1, 2, 3} {
		c = add(fmt.Sprintf("ba%d", ba), 1)
		c.ba = ba
	}
	for _, n := range []int{1, 8} {
		c = add(fmt.Sprintf("3g%d", n), 1)
		c.n3g = n
	}
	for _, x := range []int{0, 15} {
		c = add(fmt.Sprintf("grnti%d", x), 2)
		c.grnti = x
	}
	for _, n := range []int{1, 4} {
		c = add(fmt.Sprintf("eutran%d", n), 3)
		c.ne = n
	}
	for _, rel := range []int{4, 5} {
		for _, kind := range []int{1, 2} {
			for _, plmn := range []bool{false, true} {
				c = add(fmt.Sprintf("release%d-csg%d-plmn%t", rel, kind, plmn), rel)
				c.csg = kind
				c.plmn = plmn
			}
		}
	}
	for _, plmn := range []bool{false, true} {
		c = add(fmt.Sprintf("both-csg-plmn%t", plmn), 5)
		c.csg = 3
		c.plmn = plmn
	}
	c = add("release8-no-grnti", 3)
	c.grnti = -1
	c.ne = 1
	c = add("combined", 5)
	c.nc = 1
	c.bsic = true
	c.psi5 = 0
	c.ba = 3
	c.n3g = 1
	c.ne = 1
	c.grnti = 15
	return out
}

// These synthetic bit strings follow TS 44.060 V19.0.0 §11.2.9
// paragraphs 15438–15479, with component widths from §§12.60–12.61.
// The identity is 0x10203040; report quantities exercise high bits.
func measurementPrefix(c measurementCase) string {
	s := fmt.Sprintf("%032b", uint32(0x10203040))
	if c.psi5 < 0 {
		s += "0"
	} else {
		s += fmt.Sprintf("1%02b", c.psi5)
	}
	s += fmt.Sprintf("01%06b0%03b", 45, c.nc)
	for i := 0; i < c.nc; i++ {
		s += fmt.Sprintf("%06b", i+1)
		if c.bsic {
			s += fmt.Sprintf("1%06b", 63-i)
		} else {
			s += "0"
		}
		s += fmt.Sprintf("%06b", 50-i)
	}
	if c.release == 0 {
		return s
	}
	s += "1"
	switch c.ba {
	case 0:
		s += "0"
	case 1:
		s += "10101"
	case 2:
		s += "10011"
	default:
		s += "11111"
	}
	if c.n3g == 0 {
		s += "0"
	} else {
		s += fmt.Sprintf("1%03b", c.n3g-1)
		for i := 0; i < c.n3g; i++ {
			s += fmt.Sprintf("%07b%06b", 127-i, 63-i)
		}
	}
	if c.release == 1 {
		return s
	}
	s += "1"
	if c.grnti < 0 {
		s += "0"
	} else {
		s += fmt.Sprintf("1%04b", c.grnti)
	}
	if c.release == 2 {
		return s
	}
	s += "1"
	if c.ne == 0 {
		s += "0"
	} else {
		s += fmt.Sprintf("1%02b", c.ne-1)
		for i := 0; i < c.ne; i++ {
			s += fmt.Sprintf("%03b%09b%06b", 7-i, 511-i, 60-i)
		}
	}
	if c.release == 3 {
		return s
	}
	s += "1"
	for _, kind := range []int{1, 2} {
		if c.csg&kind == 0 {
			s += "0"
			continue
		}
		s += fmt.Sprintf("1%028b", 0x1234567)
		if kind == 2 {
			s += fmt.Sprintf("%016b", 0x5aa5)
		}
		if c.plmn {
			s += fmt.Sprintf("1%012b%012b", 0x123, 0x456)
		} else {
			s += "0"
		}
		s += fmt.Sprintf("%027b1%06b", 0x3456789, 53)
	}
	if c.release == 4 {
		return s
	}
	return s + "11"
}

func measurementBits(t testing.TB, s string) []byte {
	t.Helper()
	out := make([]byte, (len(s)+7)/8)
	for i, b := range s {
		if b == '1' {
			out[i/8] |= 1 << uint(7-i%8)
		} else if b != '0' {
			t.Fatalf("invalid bit %q", b)
		}
	}
	return out
}

// Uplink CS-1 or CS-3 (§10.0b.2.1 paragraphs 7388–7390): MAC octet, external selector 000100 (§11.2.0.2
// paragraph 13428), then content at absolute bit 14. Padding's leading
// zero and positional 0x2b pattern are §11 paragraphs 12937–12940.
func measurementBlock(t testing.TB, c measurementCase) []byte {
	t.Helper()
	s := "01000000000100" + measurementPrefix(c)
	extent := 184
	if len(s) >= extent {
		extent = 312
	}
	if len(s) >= extent {
		t.Fatalf("vector %s uses %d bits", c.name, len(s))
	}
	s += "0"
	for len(s) < extent {
		if c.release < 5 {
			s += "1"
		} else {
			s += fmt.Sprint((0x2b >> uint(7-len(s)%8)) & 1)
		}
	}
	return measurementBits(t, s)
}

func measurementReader(t testing.TB, data []byte) *runtime.Reader {
	t.Helper()
	r, err := runtime.NewChoiceReaderBits(data, len(data)*8, nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx := runtime.RLCMACContext{Direction: runtime.RLCMACUplink, Mode: runtime.RLCMACOrdinary, Channel: runtime.RLCMACPACCH}
	if _, err = controlblock.DecodeBlockHeaderFrom(r, ctx, func() controlblock.BlockFormat {
		if len(data) == 39 {
			return controlblock.CS3
		}
		return controlblock.CS1
	}()); err != nil {
		t.Fatal(err)
	}
	selector, err := r.ReadUint(6)
	if err != nil || selector != 4 || r.Position() != 14 {
		t.Fatalf("selector %d at %d: %v", selector, r.Position(), err)
	}
	return r
}

// TS 44.060 V19.0.0 sections 10.3.1-10.3.2: ordinary content starts
// after the eight-bit header and six-bit message selector.
func measurementContentReader(t testing.TB, data []byte, bits int) *runtime.Reader {
	t.Helper()
	w := runtime.NewWriter()
	if err := w.WriteUint(0, 14); err != nil {
		t.Fatal(err)
	}
	if err := w.WriteBitString(runtime.BitString{Bytes: data, BitLength: bits}, bits); err != nil {
		t.Fatal(err)
	}
	raw, err := w.FinishBits(runtime.BitString{})
	if err != nil {
		t.Fatal(err)
	}
	r, err := runtime.NewChoiceReaderBits(raw.Bytes, raw.BitLength, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.ReadUint(14); err != nil {
		t.Fatal(err)
	}
	return r
}

func checkMeasurement(t testing.TB, c measurementCase, v measurementreport.PacketMeasurementReportMessageContent) {
	t.Helper()
	if v.TLLIGRNTI != 0x10203040 {
		t.Fatalf("identity %x", v.TLLIGRNTI)
	}
	if c.psi5 < 0 {
		if v.PSI5CHANGEMARK != nil {
			t.Fatal("unexpected PSI5")
		}
	} else if v.PSI5CHANGEMARK == nil || int(*v.PSI5CHANGEMARK) != c.psi5 {
		t.Fatal("PSI5 mismatch")
	}
	nc := v.NCMeasurementReport
	if nc.NCMODE != 1 || nc.RXLEVSERVINGCELL != 45 || int(nc.NUMBEROFNCMEASUREMENTS) != c.nc || len(nc.FREQUENCYNGroupList) != c.nc {
		t.Fatalf("NC report %+v", nc)
	}
	for i, x := range nc.FREQUENCYNGroupList {
		if int(x.FREQUENCYN) != i+1 || int(x.RXLEVN) != 50-i {
			t.Fatalf("NC item %+v", x)
		}
		if c.bsic {
			if x.BSICN == nil || int(*x.BSICN) != 63-i {
				t.Fatal("BSIC mismatch")
			}
		} else if x.BSICN != nil {
			t.Fatal("unexpected BSIC")
		}
	}
	if c.release == 0 {
		if v.BAUSEDChoice.NoString == nil {
			t.Fatal("missing earlier-release arm")
		}
		return
	}
	r99 := v.BAUSEDChoice.BAUSED
	if r99 == nil {
		t.Fatal("missing release 99")
	}
	if c.ba == 0 {
		if r99.BAUSEDChoiceGroup != nil {
			t.Fatal("unexpected BA")
		}
	} else {
		ba := r99.BAUSEDChoiceGroup
		if ba == nil || ba.PMOUSED != 1 {
			t.Fatal("BA/PMO mismatch")
		}
		if c.ba == 3 {
			if ba.BAUSEDChoice.PSI3CHANGEMARK == nil || ba.BAUSEDChoice.PSI3CHANGEMARK.PSI3CHANGEMARK != 3 {
				t.Fatal("PSI3 mismatch")
			}
		} else {
			x := ba.BAUSEDChoice.BAUSED
			if x == nil || int(x.BAUSED) != c.ba%2 || int(x.N3GBAUSED) != 1-c.ba%2 {
				t.Fatal("BA flags mismatch")
			}
		}
	}
	if c.n3g == 0 {
		if r99.N3GMeasurementReport != nil {
			t.Fatal("unexpected 3G")
		}
	} else {
		x := r99.N3GMeasurementReport
		if x == nil || int(x.N3G) != c.n3g-1 || len(x.N3GCELLLISTINDEXGroupList) != c.n3g {
			t.Fatal("3G count mismatch")
		}
		for i, q := range x.N3GCELLLISTINDEXGroupList {
			if int(q.N3GCELLLISTINDEX) != 127-i || int(q.REPORTINGQUANTITY) != 63-i {
				t.Fatal("3G item mismatch")
			}
		}
	}
	if c.release == 1 {
		if r99.GRNTIExtensionChoice.NoString == nil {
			t.Fatal("missing release 99 stop")
		}
		return
	}
	r5 := r99.GRNTIExtensionChoice.GRNTIExtension
	if r5 == nil {
		t.Fatal("missing release 5")
	}
	if c.grnti < 0 {
		if r5.GRNTIExtension != nil {
			t.Fatal("unexpected G-RNTI extension")
		}
	} else if r5.GRNTIExtension == nil || int(*r5.GRNTIExtension) != c.grnti {
		t.Fatal("G-RNTI mismatch")
	}
	if c.release == 2 {
		if r5.EUTRANMeasurementReportChoice.NoString == nil {
			t.Fatal("missing release 5 stop")
		}
		return
	}
	r8 := r5.EUTRANMeasurementReportChoice.EUTRANMeasurementReport
	if r8 == nil {
		t.Fatal("missing release 8")
	}
	if c.ne == 0 {
		if r8.EUTRANMeasurementReport != nil {
			t.Fatal("unexpected E-UTRAN")
		}
	} else {
		x := r8.EUTRANMeasurementReport
		if x == nil || int(x.NEUTRAN) != c.ne-1 || len(x.EUTRANFREQUENCYINDEXGroupList) != c.ne {
			t.Fatal("E-UTRAN count mismatch")
		}
		for i, q := range x.EUTRANFREQUENCYINDEXGroupList {
			if int(q.EUTRANFREQUENCYINDEX) != 7-i || int(q.CELLIDENTITY) != 511-i || int(q.REPORTINGQUANTITY) != 60-i {
				t.Fatal("E-UTRAN item mismatch")
			}
		}
	}
	if c.release == 3 {
		if r8.UTRANCSGMeasurementReportChoice.NoString == nil {
			t.Fatal("missing release 8 stop")
		}
		return
	}
	r9 := r8.UTRANCSGMeasurementReportChoice.UTRANCSGMeasurementReport
	if r9 == nil {
		t.Fatal("missing release 9")
	}
	if c.csg&1 != 0 {
		x := r9.UTRANCSGMeasurementReport
		if x == nil || x.UTRANCGI != 0x1234567 || x.CSGID != 0x3456789 || x.AccessMode != 1 || x.REPORTINGQUANTITY != 53 {
			t.Fatal("UTRAN CSG mismatch")
		}
		if c.plmn {
			if x.PLMNID == nil || x.PLMNID.MCC != 0x123 || x.PLMNID.MNC != 0x456 {
				t.Fatal("UTRAN PLMN mismatch")
			}
		} else if x.PLMNID != nil {
			t.Fatal("unexpected PLMN")
		}
	} else if r9.UTRANCSGMeasurementReport != nil {
		t.Fatal("unexpected UTRAN CSG")
	}
	if c.csg&2 != 0 {
		x := r9.EUTRANCSGMeasurementReport
		if x == nil || x.EUTRANCI != 0x1234567 || x.TrackingAreaCode != 0x5aa5 || x.CSGID != 0x3456789 || x.AccessMode != 1 || x.REPORTINGQUANTITY != 53 {
			t.Fatal("E-UTRAN CSG mismatch")
		}
		if c.plmn {
			if x.PLMNID == nil || x.PLMNID.MCC != 0x123 || x.PLMNID.MNC != 0x456 {
				t.Fatal("E-UTRAN PLMN mismatch")
			}
		} else if x.PLMNID != nil {
			t.Fatal("unexpected PLMN")
		}
	} else if r9.EUTRANCSGMeasurementReport != nil {
		t.Fatal("unexpected E-UTRAN CSG")
	}
	if c.release == 4 {
		if r9.SI23BAUSEDChoice.NoString == nil {
			t.Fatal("missing release 9 stop")
		}
		return
	}
	if r9.SI23BAUSEDChoice.SI23BAUSED == nil || r9.SI23BAUSEDChoice.SI23BAUSED.SI23BAUSED != 1 {
		t.Fatal("SI23 BA mismatch")
	}
}

func TestClause1129CompleteUplinkBlocks(t *testing.T) {
	for _, c := range measurementCases() {
		t.Run(c.name, func(t *testing.T) {
			data := measurementBlock(t, c)
			total := len(data) * 8
			t.Logf("synthetic=%x", data)
			r := measurementReader(t, data)
			v, err := measurementreport.DecodePacketMeasurementReportMessageContentFrom(r)
			if err != nil {
				t.Fatal(err)
			}
			checkMeasurement(t, c, v)
			if r.Position() != total {
				t.Fatalf("position %d", r.Position())
			}
			wire, err := runtime.SealBits(struct{}{}, runtime.BitString{Bytes: data, BitLength: total}, total, runtime.BitString{}, r.Wire())
			if err != nil {
				t.Fatal(err)
			}
			w := runtime.NewWriter()
			w.WithWire(wire)
			if err = w.WriteUint(0x40, 8); err != nil {
				t.Fatal(err)
			}
			if err = w.WriteUint(4, 6); err != nil {
				t.Fatal(err)
			}
			if err = measurementreport.EncodePacketMeasurementReportMessageContentTo(w, v); err != nil {
				t.Fatal(err)
			}
			replay, err := w.FinishBits(runtime.BitString{})
			if err != nil || replay.BitLength != total || !bytes.Equal(replay.Bytes, data) {
				t.Fatalf("replay %x: %v", replay.Bytes, err)
			}
			fresh := runtime.Canonical(v)
			extent := total
			if c.release < 5 {
				extent = 14 + len(measurementPrefix(c)) + 1
			}
			cw := runtime.NewWriter()
			cw.WithWire(runtime.CanonicalTo(struct{ Wire runtime.WireInfo }{}, extent).Wire)
			if _, err = cw.PushLimit(extent); err != nil {
				t.Fatal(err)
			}
			if err = cw.WriteUint(0x40, 8); err != nil {
				t.Fatal(err)
			}
			if err = cw.WriteUint(4, 6); err != nil {
				t.Fatal(err)
			}
			if err = measurementreport.EncodePacketMeasurementReportMessageContentTo(cw, fresh); err != nil {
				t.Fatal(err)
			}
			canonical, err := cw.FinishBits(runtime.BitString{})
			if err != nil {
				t.Fatal(err)
			}
			want := "01000000000100" + measurementPrefix(c) + "0"
			if c.release == 5 {
				if canonical.BitLength != total || !bytes.Equal(canonical.Bytes, data) {
					t.Fatal("canonical padding changed")
				}
			} else if canonical.BitLength != extent || !bytes.Equal(canonical.Bytes, measurementBits(t, want)) {
				t.Fatalf("canonical earlier release %x/%d", canonical.Bytes, canonical.BitLength)
			}
			v.TLLIGRNTI ^= 1
			ew := runtime.NewWriter()
			ew.WithWire(wire)
			if err = ew.WriteUint(0x40, 8); err != nil {
				t.Fatal(err)
			}
			if err = ew.WriteUint(4, 6); err != nil {
				t.Fatal(err)
			}
			if err = measurementreport.EncodePacketMeasurementReportMessageContentTo(ew, v); err != nil {
				t.Fatal(err)
			}
			edited, err := ew.FinishBits(runtime.BitString{})
			if err != nil {
				t.Fatal(err)
			}
			er := measurementReader(t, edited.Bytes)
			ev, err := measurementreport.DecodePacketMeasurementReportMessageContentFrom(er)
			if err != nil || ev.TLLIGRNTI != 0x10203041 {
				t.Fatalf("edited identity %x: %v", ev.TLLIGRNTI, err)
			}
		})
	}
}

func TestClause1129NullTruncationAndSentinel(t *testing.T) {
	for rel := 0; rel <= 5; rel++ {
		c := measurementCase{name: "null", release: rel, psi5: -1, grnti: -1}
		s := measurementPrefix(c)
		if rel == 5 {
			s += "0"
			for (14+len(s))%8 != 0 {
				s += fmt.Sprint((0x2b >> uint(7-(14+len(s))%8)) & 1)
			}
		}
		raw := measurementBits(t, s+"101011")
		r := measurementContentReader(t, raw, len(s)+6)
		prior, err := r.PushLimit(len(s))
		if err != nil {
			t.Fatal(err)
		}
		if _, err = measurementreport.DecodePacketMeasurementReportMessageContentFrom(r); err != nil {
			t.Fatalf("release%d null: %v", rel, err)
		}
		if r.Position() != 14+len(s) {
			t.Fatalf("release%d crossed limit", rel)
		}
		if err = r.PopLimit(prior); err != nil {
			t.Fatal(err)
		}
		sentinel, err := r.ReadUint(6)
		if err != nil || sentinel != 43 {
			t.Fatalf("sentinel %d: %v", sentinel, err)
		}
	}
	c := measurementCase{release: 0, psi5: -1, grnti: -1}
	s := measurementPrefix(c)
	for n := 0; n < len(s); n++ {
		r := measurementContentReader(t, measurementBits(t, s), n)
		if _, err := measurementreport.DecodePacketMeasurementReportMessageContentFrom(r); err == nil {
			t.Fatalf("mandatory prefix accepted at %d bits", n)
		}
	}
	bad := []byte(measurementBits(t, s))
	bad[4] |= 0x40
	r := measurementContentReader(t, bad, len(s))
	if _, err := measurementreport.DecodePacketMeasurementReportMessageContentFrom(r); err == nil {
		t.Fatal("reserved report selector accepted")
	}
}

func TestClause1129ReportCountBoundaries(t *testing.T) {
	var nc measurementreport.NCMeasurementReportStruct
	if err := nc.SetFREQUENCYNGroupList(make([]measurementreport.NCMeasurementReportStructFREQUENCYNGroupListEntry, 8)); err == nil {
		t.Fatal("eight NC reports accepted")
	}
	var g measurementreport.N3GMeasurementReportStruct
	if err := g.SetN3GCELLLISTINDEXGroupList(nil); err == nil {
		t.Fatal("empty 3G list accepted")
	}
	if err := g.SetN3GCELLLISTINDEXGroupList(make([]measurementreport.N3GMeasurementReportStructN3GCELLLISTINDEXGroupListEntry, 9)); err == nil {
		t.Fatal("nine 3G reports accepted")
	}
	var e measurementreport.EUTRANMeasurementReportStruct
	if err := e.SetEUTRANFREQUENCYINDEXGroupList(nil); err == nil {
		t.Fatal("empty E-UTRAN list accepted")
	}
	if err := e.SetEUTRANFREQUENCYINDEXGroupList(make([]measurementreport.EUTRANMeasurementReportStructEUTRANFREQUENCYINDEXGroupListEntry, 5)); err == nil {
		t.Fatal("five E-UTRAN reports accepted")
	}
	for _, n := range []int{0, 1, 7} {
		items := make([]measurementreport.NCMeasurementReportStructFREQUENCYNGroupListEntry, n)
		if err := nc.SetFREQUENCYNGroupList(items); err != nil {
			t.Fatal(err)
		}
		if int(nc.NUMBEROFNCMEASUREMENTS) != n {
			t.Fatal("NC count offset")
		}
	}
	for _, n := range []int{1, 8} {
		items := make([]measurementreport.N3GMeasurementReportStructN3GCELLLISTINDEXGroupListEntry, n)
		if err := g.SetN3GCELLLISTINDEXGroupList(items); err != nil {
			t.Fatal(err)
		}
		if int(g.N3G) != n-1 {
			t.Fatal("3G count offset")
		}
	}
	for _, n := range []int{1, 4} {
		items := make([]measurementreport.EUTRANMeasurementReportStructEUTRANFREQUENCYINDEXGroupListEntry, n)
		if err := e.SetEUTRANFREQUENCYINDEXGroupList(items); err != nil {
			t.Fatal(err)
		}
		if int(e.NEUTRAN) != n-1 {
			t.Fatal("E-UTRAN count offset")
		}
	}
}

func TestClause1129StandaloneCanonicalExtent(t *testing.T) {
	// The content has no standalone containing-block maximum; callers supply it.
	v, err := measurementreport.DecodePacketMeasurementReportMessageContent([]byte{0x10, 0x20, 0x30, 0x40, 0x36, 0x80})
	if err != nil {
		t.Fatal(err)
	}
	_, err = measurementreport.EncodePacketMeasurementReportMessageContentCanonical(v.Value)
	var extent *runtime.ExtentError
	if !errors.As(err, &extent) {
		t.Fatalf("unbounded canonical: %v", err)
	}
}

func FuzzClause1129Content(f *testing.F) {
	f.Add([]byte{0x10, 0x20, 0x30, 0x40, 0x36, 0x80}, uint16(48))
	f.Fuzz(func(t *testing.T, data []byte, n uint16) {
		if len(data) > 4096 {
			return
		}
		extent := min(len(data)*8, int(n))
		shifted := make([]byte, (14+extent+7)/8)
		for i := 0; i < extent; i++ {
			shifted[(14+i)/8] |= ((data[i/8] >> uint(7-i%8)) & 1) << uint(7-(14+i)%8)
		}
		r, err := runtime.NewChoiceReaderBits(shifted, 14+extent, nil)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = r.ReadUint(14); err != nil {
			t.Fatal(err)
		}
		_, _ = measurementreport.DecodePacketMeasurementReportMessageContentFrom(r)
		if r.Position() > 14+extent {
			t.Fatal("crossed exact extent")
		}
	})
}

func TestClause1129PaddingPhasesAndEdits(t *testing.T) {
	c := measurementCase{release: 5, nc: 1, psi5: -1, grnti: 15}
	for _, phase := range []int{0, 1, 2, 3, 4, 5, 6, 7, 14, 22} {
		s := ""
		for i := 0; i < phase; i++ {
			s += "1"
		}
		s += measurementPrefix(c)
		s += "0"
		for len(s)%8 != 0 {
			s += fmt.Sprint((0x2b >> uint(7-len(s)%8)) & 1)
		}
		data := measurementBits(t, s+"101011")
		r, err := runtime.NewChoiceReaderBits(data, len(s)+6, nil)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = r.ReadUint(phase); err != nil {
			t.Fatal(err)
		}
		prior, err := r.PushLimit(len(s) - phase)
		if err != nil {
			t.Fatal(err)
		}
		v, err := measurementreport.DecodePacketMeasurementReportMessageContentFrom(r)
		if phase%8 != 6 {
			var phaseError *measurementreport.ContentPhaseError
			if !errors.As(err, &phaseError) || phaseError.SourceOrigin() != 14 || phaseError.BitPosition() != phase || r.Position() != phase {
				t.Fatalf("reader phase %d changed cursor or error: %v", phase, err)
			}
			target := runtime.NewWriter()
			if e := target.WriteUint(0, phase); e != nil {
				t.Fatal(e)
			}
			writerErr := measurementreport.EncodePacketMeasurementReportMessageContentTo(target, v)
			if !errors.As(writerErr, &phaseError) || phaseError.SourceOrigin() != 14 || phaseError.BitPosition() != phase || target.Position() != phase {
				t.Fatalf("writer phase %d changed cursor or error: %v", phase, writerErr)
			}
			continue
		}
		if err != nil {
			t.Fatalf("phase%d: %v", phase, err)
		}
		checkMeasurement(t, c, v)
		if err = r.PopLimit(prior); err != nil {
			t.Fatal(err)
		}
		sentinel, err := r.ReadUint(6)
		if err != nil || sentinel != 43 {
			t.Fatal("padding crossed sentinel")
		}
		fresh := runtime.Canonical(v)
		fresh.NCMeasurementReport.RXLEVSERVINGCELL = 44
		w := runtime.NewWriter()
		w.WithWire(runtime.CanonicalTo(struct{ Wire runtime.WireInfo }{}, len(s)).Wire)
		if _, err = w.PushLimit(len(s)); err != nil {
			t.Fatal(err)
		}
		for i := 0; i < phase; i++ {
			if err = w.WriteUint(1, 1); err != nil {
				t.Fatal(err)
			}
		}
		if err = measurementreport.EncodePacketMeasurementReportMessageContentTo(w, fresh); err != nil {
			t.Fatal(err)
		}
		out, err := w.FinishBits(runtime.BitString{})
		if err != nil || out.BitLength != len(s) {
			t.Fatalf("phase%d canonical %d: %v", phase, out.BitLength, err)
		}
		er, err := runtime.NewChoiceReaderBits(out.Bytes, out.BitLength, nil)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = er.ReadUint(phase); err != nil {
			t.Fatal(err)
		}
		ev, err := measurementreport.DecodePacketMeasurementReportMessageContentFrom(er)
		if err != nil || ev.NCMeasurementReport.RXLEVSERVINGCELL != 44 || er.Position() != len(s) {
			t.Fatalf("edited phase%d: %v", phase, err)
		}
	}
}

func TestClause1129TruncatedReleaseFields(t *testing.T) {
	for _, c := range []measurementCase{
		{release: 1, nc: 1, n3g: 1, psi5: -1, grnti: -1},
		{release: 2, nc: 1, psi5: -1, grnti: 15},
		{release: 3, nc: 1, ne: 1, psi5: -1, grnti: 15},
		{release: 4, nc: 1, csg: 2, plmn: true, psi5: -1, grnti: 15},
	} {
		s := measurementPrefix(c)
		// The release-9 root has an absent second CSG arm after a UTRAN report;
		// using the E-UTRAN arm makes the final six bits a mandatory quantity.
		maximumMissing := 5
		if c.release == 2 {
			maximumMissing = 3
		}
		for missing := 1; missing <= maximumMissing; missing++ {
			r := measurementContentReader(t, measurementBits(t, s), len(s)-missing)
			if _, err := measurementreport.DecodePacketMeasurementReportMessageContentFrom(r); err == nil {
				t.Fatalf("release%d accepted %d missing field bits", c.release, missing)
			}
		}
	}
}

func TestClause1129IgnoredTailGrowth(t *testing.T) {
	c := measurementCase{release: 0, nc: 1, psi5: -1, grnti: -1}
	prefix := measurementPrefix(c) + "0"
	for _, size := range []int{64, 256, 1024, 4096} {
		s := prefix + strings.Repeat("1", size*8-len(prefix))
		data := measurementBits(t, s)
		v, err := measurementreport.DecodePacketMeasurementReportMessageContent(data)
		if err != nil {
			t.Fatal(err)
		}
		if v.Value.TLLIGRNTI != 0x10203040 || v.Value.NCMeasurementReport.NUMBEROFNCMEASUREMENTS != 1 {
			t.Fatal("tail changed accepted prefix")
		}
		replay, err := measurementreport.EncodePacketMeasurementReportMessageContent(v.Value)
		if err != nil || !bytes.Equal(replay, data) {
			t.Fatalf("tail size%d: %v", size, err)
		}
		allocations := testing.AllocsPerRun(3, func() { _, err = measurementreport.DecodePacketMeasurementReportMessageContent(data) })
		if err != nil {
			t.Fatal(err)
		}
		// A tail's bits may be copied, but must not allocate a node per bit.
		if !raceEnabled && allocations > 200 {
			t.Fatalf("tail size%d: %.0f allocations", size, allocations)
		}
	}
}

func TestClause1129PaddingIgnoreKeepsPrefix(t *testing.T) {
	c := measurementCase{release: 5, nc: 1, psi5: -1, grnti: 15}
	data := measurementBlock(t, c)
	offset := 14 + len(measurementPrefix(c))
	data[offset/8] |= 1 << uint(7-offset%8)
	collector := runtime.NewReceiverCollector()
	r, err := runtime.NewChoiceReaderBits(data, 184, collector)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = r.ReadUint(14); err != nil {
		t.Fatal(err)
	}
	v, err := measurementreport.DecodePacketMeasurementReportMessageContentFrom(r)
	if err != nil {
		t.Fatal(err)
	}
	checkMeasurement(t, c, v)
	report := collector.TakeReceiverReport()
	if len(report.Regions) != 1 || report.Regions[0].Label != runtime.Ignore || r.Position() != 184 {
		t.Fatalf("padding Ignore report %+v at %d", report, r.Position())
	}
	wire, err := runtime.SealBits(struct{}{}, runtime.BitString{Bytes: data, BitLength: 184}, 184, runtime.BitString{}, r.Wire())
	if err != nil {
		t.Fatal(err)
	}
	w := runtime.NewWriter()
	w.WithWire(wire)
	if err = w.WriteUint(0x40, 8); err != nil {
		t.Fatal(err)
	}
	if err = w.WriteUint(4, 6); err != nil {
		t.Fatal(err)
	}
	if err = measurementreport.EncodePacketMeasurementReportMessageContentTo(w, v); err != nil {
		t.Fatal(err)
	}
	out, err := w.FinishBits(runtime.BitString{})
	if err != nil || out.BitLength != 184 || !bytes.Equal(out.Bytes, data) {
		t.Fatalf("Ignore replay %x: %v", out.Bytes, err)
	}
}

// TS 44.060 V19.0.0 §11.2.9 paragraph 15442: a canonical sender
// discards the earlier-release ignored tail and cannot fill a larger extent.
func TestClause1129LegacyCanonicalExtent(t *testing.T) {
	data := []byte{0x10, 0x20, 0x30, 0x40, 0x36, 0x80}
	v, err := measurementreport.DecodePacketMeasurementReportMessageContent(data)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := measurementreport.EncodePacketMeasurementReportMessageContent(v.Value)
	if err != nil || !bytes.Equal(replay, data) {
		t.Fatalf("legacy replay %x: %v", replay, err)
	}
	_, err = measurementreport.EncodePacketMeasurementReportMessageContentCanonicalAtLength(v.Value, 7)
	var extent *runtime.ExtentError
	if !errors.As(err, &extent) || extent.Required || extent.Actual != 6 || extent.Minimum != 7 || extent.Maximum != 7 {
		t.Fatalf("unfilled legacy target: %v", err)
	}
}
