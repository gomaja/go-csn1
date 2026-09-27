//go:build integration

package uecapability

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"

	rrc "github.com/gomaja/go-asn1/telecom/lte/rrc"
)

// TestLocalUuContainers reads local trace data only. No trace bytes or values
// derived from them belong in the repository.
func TestLocalUuContainers(t *testing.T) {
	path := os.Getenv("GO_CSN1_LTE_RRC_UU_JSON")
	if path == "" {
		t.Skip("GO_CSN1_LTE_RRC_UU_JSON is unset")
	}
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		t.Skip("local trace file absent")
	}
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 65536), 8<<20)
	var messages, cs, ps int
	for scanner.Scan() {
		var event struct {
			Name string `json:"event_name"`
			PDU  string `json:"pdu_hex"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			t.Fatal(err)
		}
		if event.Name != "UECapabilityInformation" {
			continue
		}
		messages++
		data, err := hex.DecodeString(event.PDU)
		if err != nil {
			t.Fatal(err)
		}
		var message rrc.ULDCCHMessage
		if err := message.UnmarshalUPER(data); err != nil {
			t.Fatalf("PDU %d: %v", messages, err)
		}
		if message.Message.C1 == nil || message.Message.C1.UeCapabilityInformation == nil {
			t.Fatalf("PDU %d: missing capability message", messages)
		}
		ce := message.Message.C1.UeCapabilityInformation.CriticalExtensions.C1
		if ce == nil || ce.UeCapabilityInformationR8 == nil {
			t.Fatalf("PDU %d: missing r8 IEs", messages)
		}
		for _, container := range ce.UeCapabilityInformationR8.UeCapabilityRATContainerList {
			switch container.RatType {
			case rrc.RATTypeGeranCs:
				cs++
				decoded, err := DecodeGERANCS(container.UeCapabilityRATContainer)
				if err != nil {
					t.Fatalf("GERAN CS %d: %v", cs, err)
				}
				encoded, err := EncodeGERANCS(decoded.Value)
				if err != nil || !bytes.Equal(encoded, container.UeCapabilityRATContainer) {
					t.Fatalf("GERAN CS %d: encode=%v", cs, err)
				}
			case rrc.RATTypeGeranPs:
				ps++
				decoded, err := DecodeGERANPS(container.UeCapabilityRATContainer)
				if err != nil {
					t.Fatalf("GERAN PS %d: %v", ps, err)
				}
				encoded, err := EncodeGERANPS(decoded.Value)
				if err != nil || !bytes.Equal(encoded, container.UeCapabilityRATContainer) {
					t.Fatalf("GERAN PS %d: encode=%v", ps, err)
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	t.Logf("UECapabilityInformation=%d GERAN-CS=%d GERAN-PS=%d", messages, cs, ps)
	if messages == 0 || cs == 0 || ps == 0 {
		t.Fatalf("missing expected GERAN containers")
	}
}
