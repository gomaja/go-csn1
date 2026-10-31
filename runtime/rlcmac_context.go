package runtime

// RLCMACDirection is the radio direction of a control message, not the D bit
// identifying an addressed TBF (TS 44.060 V19.0.0 §10.4.9d).
type RLCMACDirection uint8

const (
	RLCMACDownlink RLCMACDirection = iota + 1
	RLCMACUplink
)

// RLCMACMode selects the distinct message maps in TS 44.060 V19.0.0
// §§11.1.1.1–3 and 11.2.0.1–2. DBPSCH is Iu mode only.
type RLCMACMode uint8

const (
	RLCMACOrdinary RLCMACMode = iota + 1
	RLCMACEC
	RLCMACDBPSCH
)

// RLCMACChannel retains the logical bearer independently of the selector map.
// Dedicated carriers are identified in §§11.2.2, 11.2.6b–c and 11.2.34.
type RLCMACChannel uint8

const (
	RLCMACPACCH RLCMACChannel = iota + 1
	RLCMACPBCCH
	RLCMACPCCCH
	RLCMACFACCH
	RLCMACSACCH
	RLCMACSDCCH
	RLCMACADCH
)

// RLCMACAllocation is caller-owned prior allocation context. Bitmap bits
// identify timeslots; their numeric value is not a count (§11.2.29a).
// A nil pointer means unavailable context, distinct from an empty allocation.
type RLCMACAllocation struct{ Timeslots uint8 }

// RLCMACCodingScheme is a known physical carrier fact (§10.3), independent
// of the message map. Zero means unspecified for a bare message-content root.
type RLCMACCodingScheme uint8

const (
	RLCMACCS1 RLCMACCodingScheme = iota + 1
	RLCMACCS3
	RLCMACMCS0
	RLCMACECCoding
)

// RLCMACContext supplies external selection facts without protocol state or
// side effects. Nil optional facts mean unknown, not zero/false. DLDC/DLMC
// are the source's dual-/multi-carrier capability facts (§§11.2.7, 11.2.31).
// CoverageClass is the selected uplink CC1..CC5 (§11.2.0.2),
// independent of the two-bit USED_DL_COVERAGE_CLASS IE (§12.71). Callers retain ownership of pointers.
type RLCMACContext struct {
	Direction       RLCMACDirection
	Mode            RLCMACMode
	Channel         RLCMACChannel
	CodingScheme    RLCMACCodingScheme
	CoverageClass   *uint8
	DLDC            *bool
	DLMC            *bool // Downlink Multi Carrier capability (§§11.2.7, 11.2.31).
	FLO             *bool // §§11.2.6b–c distinguish the two DBPSCH selector 000010 roots.
	PriorAllocation *RLCMACAllocation
}

// Validate rejects missing or incompatible map/carrier coordinates. Specific
// message bindings additionally require the optional facts their source uses.
func (c RLCMACContext) Validate() error {
	if c.Direction != RLCMACDownlink && c.Direction != RLCMACUplink || c.Mode < RLCMACOrdinary || c.Mode > RLCMACDBPSCH || c.Channel < RLCMACPACCH || c.Channel > RLCMACADCH {
		return &ContextRequiredError{Standard: "TS 44.060 V19.0.0", Clause: "11.2.0", Name: "direction, mode and channel"}
	}
	// TS 44.060 V19.0.0 §§11.2.6b–c, 11.2.28a–b and 11.2.33–34
	// establish this DBPSCH carrier union, excluding PBCCH and PCCCH.
	// The resolver applies each message's narrower channel and FLO rules.
	if c.Mode == RLCMACDBPSCH {
		switch c.Channel {
		case RLCMACPACCH, RLCMACFACCH, RLCMACSACCH, RLCMACSDCCH, RLCMACADCH:
		default:
			return &DecodeError{Kind: InvalidValue, Path: "RLCMACContext", Detail: "TS 44.060 V19.0.0 §§11.2.6b–c, 11.2.28a–b, 11.2.33–34: DBPSCH requires PACCH, FACCH, SACCH, SDCCH or ADCH"}
		}
	}
	if c.Mode == RLCMACEC && c.Channel != RLCMACPACCH || c.CoverageClass != nil && (*c.CoverageClass < 1 || *c.CoverageClass > 5) || c.CodingScheme > RLCMACECCoding {
		return &DecodeError{Kind: InvalidValue, Path: "RLCMACContext", Detail: "TS 44.060 V19.0.0 §§11.1.1, 11.2.0.2: incompatible mode, channel or coverage class"}
	}
	return nil
}
