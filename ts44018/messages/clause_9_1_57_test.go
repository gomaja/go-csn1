package messages

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"reflect"
	goruntime "runtime"
	"runtime/debug"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

type neighbourInformationVector struct {
	Name, Hex                string
	Bits, Mandatory, Content int
	Numbers                  map[string]uint64
	Absent                   []string
	Strings                  map[string]struct {
		Hex  string
		Bits int
	}
}

// TS 44.018 V19.0.0 §9.1.57, paragraphs 11475–11507; message assignment:
// §10.4 table 10.4.2. These synthetic values have an independent field reading.
const neighbourInformationVectorJSON = `[{"Name":"sequence-0-bitmap-None-bsic-None","Hex":"30072b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":15,"Numbers":{"SI10bisSequence":0},"Absent":["BSIC","ValidityBitmap"],"Strings":{},"Content":15},{"Name":"sequence-0-bitmap-None-bsic-0","Hex":"30101b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":21,"Numbers":{"SI10bisSequence":0,"BSIC":0},"Absent":["ValidityBitmap"],"Strings":{},"Content":21},{"Name":"sequence-0-bitmap-None-bsic-63","Hex":"301fdb2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":21,"Numbers":{"SI10bisSequence":0,"BSIC":63},"Absent":["ValidityBitmap"],"Strings":{},"Content":21},{"Name":"sequence-0-bitmap-0-bsic-None","Hex":"3008eb2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":18,"Numbers":{"SI10bisSequence":0,"ValidityBitmap":0},"Absent":["BSIC"],"Strings":{},"Content":18},{"Name":"sequence-0-bitmap-0-bsic-0","Hex":"3010232b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":24,"Numbers":{"SI10bisSequence":0,"BSIC":0,"ValidityBitmap":0},"Absent":[],"Strings":{},"Content":24},{"Name":"sequence-0-bitmap-0-bsic-63","Hex":"301fe32b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":24,"Numbers":{"SI10bisSequence":0,"BSIC":63,"ValidityBitmap":0},"Absent":[],"Strings":{},"Content":24},{"Name":"sequence-0-bitmap-1-bsic-None","Hex":"3009eb2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":18,"Numbers":{"SI10bisSequence":0,"ValidityBitmap":1},"Absent":["BSIC"],"Strings":{},"Content":18},{"Name":"sequence-0-bitmap-1-bsic-0","Hex":"3010272b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":24,"Numbers":{"SI10bisSequence":0,"BSIC":0,"ValidityBitmap":1},"Absent":[],"Strings":{},"Content":24},{"Name":"sequence-0-bitmap-1-bsic-63","Hex":"301fe72b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":24,"Numbers":{"SI10bisSequence":0,"BSIC":63,"ValidityBitmap":1},"Absent":[],"Strings":{},"Content":24},{"Name":"sequence-0-bitmap-2-bsic-None","Hex":"300aeb2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":18,"Numbers":{"SI10bisSequence":0,"ValidityBitmap":2},"Absent":["BSIC"],"Strings":{},"Content":18},{"Name":"sequence-0-bitmap-2-bsic-0","Hex":"30102b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":24,"Numbers":{"SI10bisSequence":0,"BSIC":0,"ValidityBitmap":2},"Absent":[],"Strings":{},"Content":24},{"Name":"sequence-0-bitmap-2-bsic-63","Hex":"301feb2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":24,"Numbers":{"SI10bisSequence":0,"BSIC":63,"ValidityBitmap":2},"Absent":[],"Strings":{},"Content":24},{"Name":"sequence-0-bitmap-3-bsic-None","Hex":"300beb2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":18,"Numbers":{"SI10bisSequence":0,"ValidityBitmap":3},"Absent":["BSIC"],"Strings":{},"Content":18},{"Name":"sequence-0-bitmap-3-bsic-0","Hex":"30102f2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":24,"Numbers":{"SI10bisSequence":0,"BSIC":0,"ValidityBitmap":3},"Absent":[],"Strings":{},"Content":24},{"Name":"sequence-0-bitmap-3-bsic-63","Hex":"301fef2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":24,"Numbers":{"SI10bisSequence":0,"BSIC":63,"ValidityBitmap":3},"Absent":[],"Strings":{},"Content":24},{"Name":"sequence-0-bitmap-4-bsic-None","Hex":"300ceb2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":18,"Numbers":{"SI10bisSequence":0,"ValidityBitmap":4},"Absent":["BSIC"],"Strings":{},"Content":18},{"Name":"sequence-0-bitmap-4-bsic-0","Hex":"3010332b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":24,"Numbers":{"SI10bisSequence":0,"BSIC":0,"ValidityBitmap":4},"Absent":[],"Strings":{},"Content":24},{"Name":"sequence-0-bitmap-4-bsic-63","Hex":"301ff32b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":24,"Numbers":{"SI10bisSequence":0,"BSIC":63,"ValidityBitmap":4},"Absent":[],"Strings":{},"Content":24},{"Name":"sequence-0-bitmap-5-bsic-None","Hex":"300deb2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":18,"Numbers":{"SI10bisSequence":0,"ValidityBitmap":5},"Absent":["BSIC"],"Strings":{},"Content":18},{"Name":"sequence-0-bitmap-5-bsic-0","Hex":"3010372b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":24,"Numbers":{"SI10bisSequence":0,"BSIC":0,"ValidityBitmap":5},"Absent":[],"Strings":{},"Content":24},{"Name":"sequence-0-bitmap-5-bsic-63","Hex":"301ff72b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":24,"Numbers":{"SI10bisSequence":0,"BSIC":63,"ValidityBitmap":5},"Absent":[],"Strings":{},"Content":24},{"Name":"sequence-0-bitmap-6-bsic-None","Hex":"300eeb2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":18,"Numbers":{"SI10bisSequence":0,"ValidityBitmap":6},"Absent":["BSIC"],"Strings":{},"Content":18},{"Name":"sequence-0-bitmap-6-bsic-0","Hex":"30103b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":24,"Numbers":{"SI10bisSequence":0,"BSIC":0,"ValidityBitmap":6},"Absent":[],"Strings":{},"Content":24},{"Name":"sequence-0-bitmap-6-bsic-63","Hex":"301ffb2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":24,"Numbers":{"SI10bisSequence":0,"BSIC":63,"ValidityBitmap":6},"Absent":[],"Strings":{},"Content":24},{"Name":"sequence-0-bitmap-7-bsic-None","Hex":"300feb2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":18,"Numbers":{"SI10bisSequence":0,"ValidityBitmap":7},"Absent":["BSIC"],"Strings":{},"Content":18},{"Name":"sequence-0-bitmap-7-bsic-0","Hex":"30103f2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":24,"Numbers":{"SI10bisSequence":0,"BSIC":0,"ValidityBitmap":7},"Absent":[],"Strings":{},"Content":24},{"Name":"sequence-0-bitmap-7-bsic-63","Hex":"301fff2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":24,"Numbers":{"SI10bisSequence":0,"BSIC":63,"ValidityBitmap":7},"Absent":[],"Strings":{},"Content":24},{"Name":"sequence-1-bitmap-None-bsic-None","Hex":"30472b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":15,"Numbers":{"SI10bisSequence":1},"Absent":["BSIC","ValidityBitmap"],"Strings":{},"Content":15},{"Name":"sequence-1-bitmap-None-bsic-0","Hex":"30501b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":21,"Numbers":{"SI10bisSequence":1,"BSIC":0},"Absent":["ValidityBitmap"],"Strings":{},"Content":21},{"Name":"sequence-1-bitmap-None-bsic-63","Hex":"305fdb2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":21,"Numbers":{"SI10bisSequence":1,"BSIC":63},"Absent":["ValidityBitmap"],"Strings":{},"Content":21},{"Name":"sequence-1-bitmap-0-bsic-None","Hex":"3048eb2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":18,"Numbers":{"SI10bisSequence":1,"ValidityBitmap":0},"Absent":["BSIC"],"Strings":{},"Content":18},{"Name":"sequence-1-bitmap-0-bsic-0","Hex":"3050232b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":24,"Numbers":{"SI10bisSequence":1,"BSIC":0,"ValidityBitmap":0},"Absent":[],"Strings":{},"Content":24},{"Name":"sequence-1-bitmap-0-bsic-63","Hex":"305fe32b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":24,"Numbers":{"SI10bisSequence":1,"BSIC":63,"ValidityBitmap":0},"Absent":[],"Strings":{},"Content":24},{"Name":"sequence-1-bitmap-1-bsic-None","Hex":"3049eb2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":18,"Numbers":{"SI10bisSequence":1,"ValidityBitmap":1},"Absent":["BSIC"],"Strings":{},"Content":18},{"Name":"sequence-1-bitmap-1-bsic-0","Hex":"3050272b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":24,"Numbers":{"SI10bisSequence":1,"BSIC":0,"ValidityBitmap":1},"Absent":[],"Strings":{},"Content":24},{"Name":"sequence-1-bitmap-1-bsic-63","Hex":"305fe72b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":24,"Numbers":{"SI10bisSequence":1,"BSIC":63,"ValidityBitmap":1},"Absent":[],"Strings":{},"Content":24},{"Name":"sequence-1-bitmap-2-bsic-None","Hex":"304aeb2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":18,"Numbers":{"SI10bisSequence":1,"ValidityBitmap":2},"Absent":["BSIC"],"Strings":{},"Content":18},{"Name":"sequence-1-bitmap-2-bsic-0","Hex":"30502b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":24,"Numbers":{"SI10bisSequence":1,"BSIC":0,"ValidityBitmap":2},"Absent":[],"Strings":{},"Content":24},{"Name":"sequence-1-bitmap-2-bsic-63","Hex":"305feb2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":24,"Numbers":{"SI10bisSequence":1,"BSIC":63,"ValidityBitmap":2},"Absent":[],"Strings":{},"Content":24},{"Name":"sequence-1-bitmap-3-bsic-None","Hex":"304beb2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":18,"Numbers":{"SI10bisSequence":1,"ValidityBitmap":3},"Absent":["BSIC"],"Strings":{},"Content":18},{"Name":"sequence-1-bitmap-3-bsic-0","Hex":"30502f2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":24,"Numbers":{"SI10bisSequence":1,"BSIC":0,"ValidityBitmap":3},"Absent":[],"Strings":{},"Content":24},{"Name":"sequence-1-bitmap-3-bsic-63","Hex":"305fef2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":24,"Numbers":{"SI10bisSequence":1,"BSIC":63,"ValidityBitmap":3},"Absent":[],"Strings":{},"Content":24},{"Name":"sequence-1-bitmap-4-bsic-None","Hex":"304ceb2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":18,"Numbers":{"SI10bisSequence":1,"ValidityBitmap":4},"Absent":["BSIC"],"Strings":{},"Content":18},{"Name":"sequence-1-bitmap-4-bsic-0","Hex":"3050332b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":24,"Numbers":{"SI10bisSequence":1,"BSIC":0,"ValidityBitmap":4},"Absent":[],"Strings":{},"Content":24},{"Name":"sequence-1-bitmap-4-bsic-63","Hex":"305ff32b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":24,"Numbers":{"SI10bisSequence":1,"BSIC":63,"ValidityBitmap":4},"Absent":[],"Strings":{},"Content":24},{"Name":"sequence-1-bitmap-5-bsic-None","Hex":"304deb2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":18,"Numbers":{"SI10bisSequence":1,"ValidityBitmap":5},"Absent":["BSIC"],"Strings":{},"Content":18},{"Name":"sequence-1-bitmap-5-bsic-0","Hex":"3050372b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":24,"Numbers":{"SI10bisSequence":1,"BSIC":0,"ValidityBitmap":5},"Absent":[],"Strings":{},"Content":24},{"Name":"sequence-1-bitmap-5-bsic-63","Hex":"305ff72b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":24,"Numbers":{"SI10bisSequence":1,"BSIC":63,"ValidityBitmap":5},"Absent":[],"Strings":{},"Content":24},{"Name":"sequence-1-bitmap-6-bsic-None","Hex":"304eeb2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":18,"Numbers":{"SI10bisSequence":1,"ValidityBitmap":6},"Absent":["BSIC"],"Strings":{},"Content":18},{"Name":"sequence-1-bitmap-6-bsic-0","Hex":"30503b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":24,"Numbers":{"SI10bisSequence":1,"BSIC":0,"ValidityBitmap":6},"Absent":[],"Strings":{},"Content":24},{"Name":"sequence-1-bitmap-6-bsic-63","Hex":"305ffb2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":24,"Numbers":{"SI10bisSequence":1,"BSIC":63,"ValidityBitmap":6},"Absent":[],"Strings":{},"Content":24},{"Name":"sequence-1-bitmap-7-bsic-None","Hex":"304feb2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":18,"Numbers":{"SI10bisSequence":1,"ValidityBitmap":7},"Absent":["BSIC"],"Strings":{},"Content":18},{"Name":"sequence-1-bitmap-7-bsic-0","Hex":"30503f2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":24,"Numbers":{"SI10bisSequence":1,"BSIC":0,"ValidityBitmap":7},"Absent":[],"Strings":{},"Content":24},{"Name":"sequence-1-bitmap-7-bsic-63","Hex":"305fff2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":24,"Numbers":{"SI10bisSequence":1,"BSIC":63,"ValidityBitmap":7},"Absent":[],"Strings":{},"Content":24},{"Name":"sequence-2-bitmap-None-bsic-None","Hex":"30872b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":15,"Numbers":{"SI10bisSequence":2},"Absent":["BSIC","ValidityBitmap"],"Strings":{},"Content":15},{"Name":"sequence-2-bitmap-None-bsic-0","Hex":"30901b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":21,"Numbers":{"SI10bisSequence":2,"BSIC":0},"Absent":["ValidityBitmap"],"Strings":{},"Content":21},{"Name":"sequence-2-bitmap-None-bsic-63","Hex":"309fdb2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":21,"Numbers":{"SI10bisSequence":2,"BSIC":63},"Absent":["ValidityBitmap"],"Strings":{},"Content":21},{"Name":"sequence-2-bitmap-0-bsic-None","Hex":"3088eb2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":18,"Numbers":{"SI10bisSequence":2,"ValidityBitmap":0},"Absent":["BSIC"],"Strings":{},"Content":18},{"Name":"sequence-2-bitmap-0-bsic-0","Hex":"3090232b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":24,"Numbers":{"SI10bisSequence":2,"BSIC":0,"ValidityBitmap":0},"Absent":[],"Strings":{},"Content":24},{"Name":"sequence-2-bitmap-0-bsic-63","Hex":"309fe32b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":24,"Numbers":{"SI10bisSequence":2,"BSIC":63,"ValidityBitmap":0},"Absent":[],"Strings":{},"Content":24},{"Name":"sequence-2-bitmap-1-bsic-None","Hex":"3089eb2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":18,"Numbers":{"SI10bisSequence":2,"ValidityBitmap":1},"Absent":["BSIC"],"Strings":{},"Content":18},{"Name":"sequence-2-bitmap-1-bsic-0","Hex":"3090272b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":24,"Numbers":{"SI10bisSequence":2,"BSIC":0,"ValidityBitmap":1},"Absent":[],"Strings":{},"Content":24},{"Name":"sequence-2-bitmap-1-bsic-63","Hex":"309fe72b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":24,"Numbers":{"SI10bisSequence":2,"BSIC":63,"ValidityBitmap":1},"Absent":[],"Strings":{},"Content":24},{"Name":"sequence-2-bitmap-2-bsic-None","Hex":"308aeb2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":18,"Numbers":{"SI10bisSequence":2,"ValidityBitmap":2},"Absent":["BSIC"],"Strings":{},"Content":18},{"Name":"sequence-2-bitmap-2-bsic-0","Hex":"30902b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":24,"Numbers":{"SI10bisSequence":2,"BSIC":0,"ValidityBitmap":2},"Absent":[],"Strings":{},"Content":24},{"Name":"sequence-2-bitmap-2-bsic-63","Hex":"309feb2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":24,"Numbers":{"SI10bisSequence":2,"BSIC":63,"ValidityBitmap":2},"Absent":[],"Strings":{},"Content":24},{"Name":"sequence-2-bitmap-3-bsic-None","Hex":"308beb2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":18,"Numbers":{"SI10bisSequence":2,"ValidityBitmap":3},"Absent":["BSIC"],"Strings":{},"Content":18},{"Name":"sequence-2-bitmap-3-bsic-0","Hex":"30902f2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":24,"Numbers":{"SI10bisSequence":2,"BSIC":0,"ValidityBitmap":3},"Absent":[],"Strings":{},"Content":24},{"Name":"sequence-2-bitmap-3-bsic-63","Hex":"309fef2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":24,"Numbers":{"SI10bisSequence":2,"BSIC":63,"ValidityBitmap":3},"Absent":[],"Strings":{},"Content":24},{"Name":"sequence-2-bitmap-4-bsic-None","Hex":"308ceb2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":18,"Numbers":{"SI10bisSequence":2,"ValidityBitmap":4},"Absent":["BSIC"],"Strings":{},"Content":18},{"Name":"sequence-2-bitmap-4-bsic-0","Hex":"3090332b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":24,"Numbers":{"SI10bisSequence":2,"BSIC":0,"ValidityBitmap":4},"Absent":[],"Strings":{},"Content":24},{"Name":"sequence-2-bitmap-4-bsic-63","Hex":"309ff32b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":24,"Numbers":{"SI10bisSequence":2,"BSIC":63,"ValidityBitmap":4},"Absent":[],"Strings":{},"Content":24},{"Name":"sequence-2-bitmap-5-bsic-None","Hex":"308deb2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":18,"Numbers":{"SI10bisSequence":2,"ValidityBitmap":5},"Absent":["BSIC"],"Strings":{},"Content":18},{"Name":"sequence-2-bitmap-5-bsic-0","Hex":"3090372b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":24,"Numbers":{"SI10bisSequence":2,"BSIC":0,"ValidityBitmap":5},"Absent":[],"Strings":{},"Content":24},{"Name":"sequence-2-bitmap-5-bsic-63","Hex":"309ff72b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":24,"Numbers":{"SI10bisSequence":2,"BSIC":63,"ValidityBitmap":5},"Absent":[],"Strings":{},"Content":24},{"Name":"sequence-2-bitmap-6-bsic-None","Hex":"308eeb2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":18,"Numbers":{"SI10bisSequence":2,"ValidityBitmap":6},"Absent":["BSIC"],"Strings":{},"Content":18},{"Name":"sequence-2-bitmap-6-bsic-0","Hex":"30903b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":24,"Numbers":{"SI10bisSequence":2,"BSIC":0,"ValidityBitmap":6},"Absent":[],"Strings":{},"Content":24},{"Name":"sequence-2-bitmap-6-bsic-63","Hex":"309ffb2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":24,"Numbers":{"SI10bisSequence":2,"BSIC":63,"ValidityBitmap":6},"Absent":[],"Strings":{},"Content":24},{"Name":"sequence-2-bitmap-7-bsic-None","Hex":"308feb2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":18,"Numbers":{"SI10bisSequence":2,"ValidityBitmap":7},"Absent":["BSIC"],"Strings":{},"Content":18},{"Name":"sequence-2-bitmap-7-bsic-0","Hex":"30903f2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":24,"Numbers":{"SI10bisSequence":2,"BSIC":0,"ValidityBitmap":7},"Absent":[],"Strings":{},"Content":24},{"Name":"sequence-2-bitmap-7-bsic-63","Hex":"309fff2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":24,"Numbers":{"SI10bisSequence":2,"BSIC":63,"ValidityBitmap":7},"Absent":[],"Strings":{},"Content":24},{"Name":"sequence-3-bitmap-None-bsic-None","Hex":"30c72b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":15,"Numbers":{"SI10bisSequence":3},"Absent":["BSIC","ValidityBitmap"],"Strings":{},"Content":15},{"Name":"sequence-3-bitmap-None-bsic-0","Hex":"30d01b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":21,"Numbers":{"SI10bisSequence":3,"BSIC":0},"Absent":["ValidityBitmap"],"Strings":{},"Content":21},{"Name":"sequence-3-bitmap-None-bsic-63","Hex":"30dfdb2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":21,"Numbers":{"SI10bisSequence":3,"BSIC":63},"Absent":["ValidityBitmap"],"Strings":{},"Content":21},{"Name":"sequence-3-bitmap-0-bsic-None","Hex":"30c8eb2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":18,"Numbers":{"SI10bisSequence":3,"ValidityBitmap":0},"Absent":["BSIC"],"Strings":{},"Content":18},{"Name":"sequence-3-bitmap-0-bsic-0","Hex":"30d0232b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":24,"Numbers":{"SI10bisSequence":3,"BSIC":0,"ValidityBitmap":0},"Absent":[],"Strings":{},"Content":24},{"Name":"sequence-3-bitmap-0-bsic-63","Hex":"30dfe32b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":24,"Numbers":{"SI10bisSequence":3,"BSIC":63,"ValidityBitmap":0},"Absent":[],"Strings":{},"Content":24},{"Name":"sequence-3-bitmap-1-bsic-None","Hex":"30c9eb2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":18,"Numbers":{"SI10bisSequence":3,"ValidityBitmap":1},"Absent":["BSIC"],"Strings":{},"Content":18},{"Name":"sequence-3-bitmap-1-bsic-0","Hex":"30d0272b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":24,"Numbers":{"SI10bisSequence":3,"BSIC":0,"ValidityBitmap":1},"Absent":[],"Strings":{},"Content":24},{"Name":"sequence-3-bitmap-1-bsic-63","Hex":"30dfe72b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":24,"Numbers":{"SI10bisSequence":3,"BSIC":63,"ValidityBitmap":1},"Absent":[],"Strings":{},"Content":24},{"Name":"sequence-3-bitmap-2-bsic-None","Hex":"30caeb2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":18,"Numbers":{"SI10bisSequence":3,"ValidityBitmap":2},"Absent":["BSIC"],"Strings":{},"Content":18},{"Name":"sequence-3-bitmap-2-bsic-0","Hex":"30d02b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":24,"Numbers":{"SI10bisSequence":3,"BSIC":0,"ValidityBitmap":2},"Absent":[],"Strings":{},"Content":24},{"Name":"sequence-3-bitmap-2-bsic-63","Hex":"30dfeb2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":24,"Numbers":{"SI10bisSequence":3,"BSIC":63,"ValidityBitmap":2},"Absent":[],"Strings":{},"Content":24},{"Name":"sequence-3-bitmap-3-bsic-None","Hex":"30cbeb2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":18,"Numbers":{"SI10bisSequence":3,"ValidityBitmap":3},"Absent":["BSIC"],"Strings":{},"Content":18},{"Name":"sequence-3-bitmap-3-bsic-0","Hex":"30d02f2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":24,"Numbers":{"SI10bisSequence":3,"BSIC":0,"ValidityBitmap":3},"Absent":[],"Strings":{},"Content":24},{"Name":"sequence-3-bitmap-3-bsic-63","Hex":"30dfef2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":24,"Numbers":{"SI10bisSequence":3,"BSIC":63,"ValidityBitmap":3},"Absent":[],"Strings":{},"Content":24},{"Name":"sequence-3-bitmap-4-bsic-None","Hex":"30cceb2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":18,"Numbers":{"SI10bisSequence":3,"ValidityBitmap":4},"Absent":["BSIC"],"Strings":{},"Content":18},{"Name":"sequence-3-bitmap-4-bsic-0","Hex":"30d0332b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":24,"Numbers":{"SI10bisSequence":3,"BSIC":0,"ValidityBitmap":4},"Absent":[],"Strings":{},"Content":24},{"Name":"sequence-3-bitmap-4-bsic-63","Hex":"30dff32b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":24,"Numbers":{"SI10bisSequence":3,"BSIC":63,"ValidityBitmap":4},"Absent":[],"Strings":{},"Content":24},{"Name":"sequence-3-bitmap-5-bsic-None","Hex":"30cdeb2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":18,"Numbers":{"SI10bisSequence":3,"ValidityBitmap":5},"Absent":["BSIC"],"Strings":{},"Content":18},{"Name":"sequence-3-bitmap-5-bsic-0","Hex":"30d0372b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":24,"Numbers":{"SI10bisSequence":3,"BSIC":0,"ValidityBitmap":5},"Absent":[],"Strings":{},"Content":24},{"Name":"sequence-3-bitmap-5-bsic-63","Hex":"30dff72b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":24,"Numbers":{"SI10bisSequence":3,"BSIC":63,"ValidityBitmap":5},"Absent":[],"Strings":{},"Content":24},{"Name":"sequence-3-bitmap-6-bsic-None","Hex":"30ceeb2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":18,"Numbers":{"SI10bisSequence":3,"ValidityBitmap":6},"Absent":["BSIC"],"Strings":{},"Content":18},{"Name":"sequence-3-bitmap-6-bsic-0","Hex":"30d03b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":24,"Numbers":{"SI10bisSequence":3,"BSIC":0,"ValidityBitmap":6},"Absent":[],"Strings":{},"Content":24},{"Name":"sequence-3-bitmap-6-bsic-63","Hex":"30dffb2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":24,"Numbers":{"SI10bisSequence":3,"BSIC":63,"ValidityBitmap":6},"Absent":[],"Strings":{},"Content":24},{"Name":"sequence-3-bitmap-7-bsic-None","Hex":"30cfeb2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":18,"Numbers":{"SI10bisSequence":3,"ValidityBitmap":7},"Absent":["BSIC"],"Strings":{},"Content":18},{"Name":"sequence-3-bitmap-7-bsic-0","Hex":"30d03f2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":24,"Numbers":{"SI10bisSequence":3,"BSIC":0,"ValidityBitmap":7},"Absent":[],"Strings":{},"Content":24},{"Name":"sequence-3-bitmap-7-bsic-63","Hex":"30dfff2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":24,"Numbers":{"SI10bisSequence":3,"BSIC":63,"ValidityBitmap":7},"Absent":[],"Strings":{},"Content":24},{"Name":"new-neighbour-nonhopping","Hex":"30a1234563ab2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":43,"Numbers":{"SI10bisSequence":2,"ChannelDescription":1193046},"Absent":[],"Strings":{},"Content":43}]`

func neighbourInformationVectors(t testing.TB) []neighbourInformationVector {
	t.Helper()
	var rows []neighbourInformationVector
	if err := json.Unmarshal([]byte(neighbourInformationVectorJSON), &rows); err != nil {
		t.Fatal(err)
	}
	return rows
}

func neighbourInformationFields(value any) map[string][]reflect.Value {
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

func neighbourInformationCheckFields(t *testing.T, v VGCSNeighbourCellInformation, row neighbourInformationVector) {
	t.Helper()
	if v.RRShortPD != 0 || v.MessageType != 12 || v.ShortLayer2Header != 0 {
		t.Fatalf("header: %#v", v)
	}
	fields := neighbourInformationFields(v)
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
		if !ok || b.BitLength != want.Bits || hex.EncodeToString(b.Bytes) != want.Hex {
			t.Fatalf("%s: %#v want %#v", name, b, want)
		}
	}
}

func neighbourInformationEqualBits(a, b []byte, n int) bool {
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

func TestVGCSNeighbourCellInformationSourceVectors(t *testing.T) {
	d, err := LookupClause("9.1.57", "VGCS Neighbour Cell Information")
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range neighbourInformationVectors(t) {
		t.Run(row.Name, func(t *testing.T) {
			wire, err := hex.DecodeString(row.Hex)
			if err != nil {
				t.Fatal(err)
			}
			raw, err := d.DecodeBits(wire, row.Bits)
			if err != nil {
				t.Fatal(err)
			}
			decoded, ok := raw.(runtime.Decoded[VGCSNeighbourCellInformation])
			if !ok {
				t.Fatalf("typed result %T", raw)
			}
			if decoded.BitsConsumed != row.Bits || decoded.Tail.BitLength != 0 {
				t.Fatalf("consumed %d of %d; tail %d", decoded.BitsConsumed, row.Bits, decoded.Tail.BitLength)
			}
			neighbourInformationCheckFields(t, decoded.Value, row)
			w := runtime.NewWriter()
			if err := EncodeVGCSNeighbourCellInformationTo(w, decoded.Value); err != nil {
				t.Fatal(err)
			}
			replay, err := w.FinishBits(decoded.Tail)
			if err != nil || replay.BitLength != row.Bits || !neighbourInformationEqualBits(replay.Bytes, wire, row.Bits) {
				t.Fatalf("replay %#v: %v", replay, err)
			}
			canonicalWriter := runtime.NewWriter()
			fresh := runtime.CanonicalTo(decoded.Value, row.Bits)
			canonicalWriter.WithWire(fresh.Wire)
			if err := EncodeVGCSNeighbourCellInformationTo(canonicalWriter, fresh); err != nil {
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
			typed := reread.(runtime.Decoded[VGCSNeighbourCellInformation])
			neighbourInformationCheckFields(t, typed.Value, row)
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
				if err := EncodeVGCSNeighbourCellInformationTo(positioned, fresh); err != nil {
					t.Fatalf("phase %d: %v", phase, err)
				}
				at, err := positioned.FinishBits(runtime.BitString{})
				if err != nil || at.BitLength != row.Bits+phase {
					t.Fatalf("phase %d extent %#v: %v", phase, at, err)
				}

				// TS 24.007 V20.0.0 Annex B §B.1.2.2: terminal spare
				// padding follows absolute 0x2B phase, independently of fields.
				expected := make([]byte, (row.Bits+phase+7)/8)
				for bit := 0; bit < row.Bits; bit++ {
					value := wire[bit/8] >> uint(7-bit%8) & 1
					if bit >= row.Content {
						value ^= byte(0x2b>>uint(7-bit%8)&1) ^ byte(0x2b>>uint(7-(bit+phase)%8)&1)
					}
					expected[(bit+phase)/8] |= value << uint(7-(bit+phase)%8)
				}
				if !bytes.Equal(at.Bytes, expected) {
					t.Fatalf("absolute phase %d: %x want %x", phase, at.Bytes, expected)
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
				neighbourInformationCheckFields(t, reread.(runtime.Decoded[VGCSNeighbourCellInformation]).Value, row)
			}
			edited := decoded.Value
			edited.RRShortPD = 1
			ew := runtime.NewWriter()
			if err := EncodeVGCSNeighbourCellInformationTo(ew, edited); err != nil {
				t.Fatal(err)
			}
			edit, err := ew.FinishBits(decoded.Tail)
			want := bytes.Clone(wire)
			want[0] ^= 0x80
			if err != nil || edit.BitLength != row.Bits || !neighbourInformationEqualBits(edit.Bytes, want, row.Bits) {
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
				if err != nil {
					t.Fatal(err)
				}
				reread, err := d.Decode(canonicalBytes)
				if err != nil {
					t.Fatal(err)
				}
				neighbourInformationCheckFields(t, reread.(runtime.Decoded[VGCSNeighbourCellInformation]).Value, row)
				stable, err := d.CanonicalAtLength(reread, len(wire))
				if err != nil || !bytes.Equal(stable, canonicalBytes) {
					t.Fatalf("unstable canonical %x: %v", stable, err)
				}
			}
			for candidate := uint8(0); candidate < 32; candidate++ {
				if candidate == 12 {
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

func TestVGCSNeighbourCellInformationInputGrowth(t *testing.T) {
	d, err := LookupClause("9.1.57", "VGCS Neighbour Cell Information")
	if err != nil {
		t.Fatal(err)
	}
	rows := neighbourInformationVectors(t)
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

func FuzzVGCSNeighbourCellInformation(f *testing.F) {
	d, err := LookupClause("9.1.57", "VGCS Neighbour Cell Information")
	if err != nil {
		f.Fatal(err)
	}
	for _, row := range neighbourInformationVectors(f) {
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
		decoded, ok := raw.(runtime.Decoded[VGCSNeighbourCellInformation])
		if !ok {
			t.Fatalf("typed decode %T", raw)
		}
		w := runtime.NewWriter()
		if err := EncodeVGCSNeighbourCellInformationTo(w, decoded.Value); err != nil {
			t.Fatal(err)
		}
		out, err := w.FinishBits(decoded.Tail)
		if err != nil || out.BitLength != bits || !neighbourInformationEqualBits(out.Bytes, data, bits) {
			t.Fatalf("accepted input replay %#v: %v", out, err)
		}
	})
}

func TestVGCSNeighbourCellInformationFieldBounds(t *testing.T) {
	row := neighbourInformationVectors(t)[0]
	wire, err := hex.DecodeString(row.Hex)
	if err != nil {
		t.Fatal(err)
	}
	d, err := LookupClause("9.1.57", "VGCS Neighbour Cell Information")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := d.DecodeBits(wire, row.Bits)
	if err != nil {
		t.Fatal(err)
	}
	original := raw.(runtime.Decoded[VGCSNeighbourCellInformation]).Value
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
		if err := EncodeVGCSNeighbourCellInformationTo(runtime.NewWriter(), v); err == nil {
			t.Fatalf("accepted invalid %s", field)
		}
	}
}

// TS 44.018 V19.0.0 §9.1.57 p11507 assigns the all-zero Validity
// bitmap diagnosis to the receiving mobile station's application procedure.
// The printed structure accepts all three-bit values and preserves their bits.
func TestVGCSNeighbourCellInformationZeroValidity(t *testing.T) {
	wire, err := hex.DecodeString("3008eb2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b")
	if err != nil {
		t.Fatal(err)
	}
	d, err := LookupClause("9.1.57", "VGCS Neighbour Cell Information")
	if err != nil {
		t.Fatal(err)
	}
	raw, reserved, receiver, err := d.DecodeBitsWithReport(wire, 184)
	if err != nil {
		t.Fatal(err)
	}
	decoded, ok := raw.(runtime.Decoded[VGCSNeighbourCellInformation])
	if !ok {
		t.Fatalf("typed result %T", raw)
	}
	fields := neighbourInformationFields(decoded.Value)
	if got := fields["ValidityBitmap"]; len(got) != 1 || got[0].Uint() != 0 {
		t.Fatalf("zero bitmap lost: %v", got)
	}
	if len(reserved.Observations) != 0 || len(receiver.Regions) != 0 {
		t.Fatalf("application procedure became a codec report: %#v %#v", reserved, receiver)
	}
	out, err := d.Encode(raw)
	if err != nil || !bytes.Equal(out, wire) {
		t.Fatalf("zero bitmap replay %x: %v", out, err)
	}
}

const neighbourRepeatVectors = `[{"Counts":[0,0],"Hex":"30bb2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":12,"Expected":[],"OracleValue":[0,12,0,2,[],"1",[],"1",["L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L"]]},{"Counts":[2,0],"Hex":"3094614a7b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":36,"Expected":[{"BSIC":17,"ValidityBitmap":0},{"BSIC":18,"ValidityBitmap":1}],"OracleValue":[0,12,0,2,[["0",["1",17],["1",0]],["0",["1",18],["1",1]]],"1",[],"1",["L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L"]]},{"Counts":[0,2],"Hex":"30a123456382468ae76b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":74,"Expected":[],"OracleValue":[0,12,0,2,[],"1",[["0",[[1193046,["0"]],["0"],3,1,["0"]]],["0",[[1193047,["0"]],["0"],3,1,["0"]]]],"1",["L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L"]]},{"Counts":[2,2],"Hex":"3094614a6123456382468ae76b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":98,"Expected":[{"BSIC":17,"ValidityBitmap":0},{"BSIC":18,"ValidityBitmap":1}],"OracleValue":[0,12,0,2,[["0",["1",17],["1",0]],["0",["1",18],["1",1]]],"1",[["0",[[1193046,["0"]],["0"],3,1,["0"]]],["0",[[1193047,["0"]],["0"],3,1,["0"]]]],"1",["L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L","L"]]}]`

// TS 44.018 V19.0.0 §9.1.57 pp11480–11484: both terminated lists
// independently allow zero and multiple entries.
func TestVGCSNeighbourCellInformationRepeatedLists(t *testing.T) {
	var rows []struct {
		Counts                   [2]int
		Hex                      string
		Bits, Mandatory, Content int
	}
	if err := json.Unmarshal([]byte(neighbourRepeatVectors), &rows); err != nil {
		t.Fatal(err)
	}
	d, err := LookupClause("9.1.57", "VGCS Neighbour Cell Information")
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		wire, err := hex.DecodeString(row.Hex)
		if err != nil {
			t.Fatal(err)
		}
		raw, err := d.DecodeBits(wire, row.Bits)
		if err != nil {
			t.Fatal(err)
		}
		v := raw.(runtime.Decoded[VGCSNeighbourCellInformation]).Value
		if len(v.BSICGroupList) != row.Counts[0] || len(v.SI10bisNeighbourCellInfoList) != row.Counts[1] {
			t.Fatalf("list counts: %+v", v)
		}
		for i, e := range v.BSICGroupList {
			if e.BSIC == nil || *e.BSIC != uint8(17+i) || e.ValidityInformation.ValidityBitmap == nil || *e.ValidityInformation.ValidityBitmap != uint8(i) {
				t.Fatalf("validity entry %d: %+v", i, e)
			}
		}
		for i, e := range v.SI10bisNeighbourCellInfoList {
			if e.SI10bisNeighbourCellInfo.GroupChannelDescription.ChannelDescription != uint32(0x123456+i) {
				t.Fatalf("neighbour entry %d", i)
			}
		}
		replay, err := d.Encode(raw)
		if err != nil || !bytes.Equal(replay, wire) {
			t.Fatalf("repeat replay %x: %v", replay, err)
		}
		canonical, err := d.CanonicalAtLength(raw, len(wire))
		if err != nil || !bytes.Equal(canonical, wire) {
			t.Fatalf("repeat canonical %x: %v", canonical, err)
		}
		if len(v.BSICGroupList) > 0 {
			*v.BSICGroupList[0].ValidityInformation.ValidityBitmap = 7
			edited, err := EncodeVGCSNeighbourCellInformation(v)
			if err != nil {
				t.Fatal(err)
			}
			reread, err := DecodeVGCSNeighbourCellInformation(edited)
			if err != nil || *reread.Value.BSICGroupList[0].ValidityInformation.ValidityBitmap != 7 {
				t.Fatalf("bitmap edit: %v", err)
			}
			*v.BSICGroupList[0].ValidityInformation.ValidityBitmap = 8
			if _, err := EncodeVGCSNeighbourCellInformation(v); err == nil {
				t.Fatal("accepted four-bit bitmap")
			}
		}
	}
}
func FuzzValidityInformation(f *testing.F) {
	d, err := LookupClause("9.1.57", "Validity Information")
	if err != nil {
		f.Fatal(err)
	}
	for i := 0; i < 16; i++ {
		f.Add([]byte{byte(i << 4)})
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 4096 {
			return
		}
		raw, err := d.Decode(data)
		if err != nil {
			return
		}
		if _, ok := raw.(runtime.Decoded[ValidityInformation]); !ok {
			t.Fatalf("typed validity %T", raw)
		}
		out, err := d.Encode(raw)
		if err != nil || !bytes.Equal(out, data) {
			t.Fatalf("validity replay %x: %v", out, err)
		}
	})
}
