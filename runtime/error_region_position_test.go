package runtime

import (
	"bytes"
	"errors"
	"testing"
)

// TS 44.060 V19.0.0 §11.1.3.2: the receiver applies the label at
// the selected place in the message, rather than at an arbitrary sender field.
func TestErrorRegionRefusesMovedPosition(t *testing.T) {
	for _, tc := range []struct {
		name        string
		destination []string
	}{
		{"same_message", []string{"Message", "Second"}},
		{"different_message_type", []string{"OtherMessage", "First"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := NewReader([]byte{0xa5})
			for _, part := range []string{"Message", "First", "Ignore"} {
				if err := r.Enter(part); err != nil {
					t.Fatal(err)
				}
			}
			region, err := r.ReadErrorRegion(Ignore, -1, fillSource)
			if err != nil {
				t.Fatal(err)
			}
			message := struct{ First, Second *ErrorRegion }{First: &region}
			message.Second, message.First = message.First, nil
			w := NewWriter()
			for _, part := range tc.destination {
				if err := w.Enter(part); err != nil {
					t.Fatal(err)
				}
			}
			err = w.WriteErrorRegion(*message.Second)
			var operation *OperationError
			if !errors.As(err, &operation) || operation.Kind != InvalidValue {
				t.Fatalf("moved receiver region accepted or wrong error: %v", err)
			}
			if w.Position() != 0 {
				t.Fatal("refused region wrote bits")
			}
		})
	}
}

func TestErrorRegionSamePositionCopy(t *testing.T) {
	decode := func(input byte) ErrorRegion {
		t.Helper()
		r := NewReader([]byte{input})
		for _, part := range []string{"Message", "First", "Ignore"} {
			if err := r.Enter(part); err != nil {
				t.Fatal(err)
			}
		}
		region, err := r.ReadErrorRegion(Ignore, -1, fillSource)
		if err != nil {
			t.Fatal(err)
		}
		return region
	}
	first, second := decode(0xa5), decode(0x5a)
	if first.Location != second.Location {
		t.Fatal("decoded values selected different positions")
	}
	second = first
	w := NewWriter()
	for _, part := range []string{"Message", "First"} {
		if err := w.Enter(part); err != nil {
			t.Fatal(err)
		}
	}
	// A preceding edit may change the bit offset without moving the field.
	if err := w.WriteUint(0, 8); err != nil {
		t.Fatal(err)
	}
	if err := w.WriteErrorRegion(second); err != nil {
		t.Fatal(err)
	}
	out, err := w.Bytes()
	if err != nil || !bytes.Equal(out, []byte{0, 0xa5}) {
		t.Fatalf("same-path copied region replay: %v", err)
	}
}

func TestErrorRegionParentScope(t *testing.T) {
	for _, path := range [][]string{nil, {"Ignore"}, {"Message", "Ignore"}, {"Message", "Entries[2]", "Body", "Ignore"}} {
		r := NewReader([]byte{0xa5})
		for _, part := range path {
			if err := r.Enter(part); err != nil {
				t.Fatal(err)
			}
		}
		region, err := r.ReadErrorRegion(Ignore, -1, fillSource)
		if err != nil {
			t.Fatal(err)
		}
		w := NewWriter()
		if len(path) > 0 {
			for _, part := range path[:len(path)-1] {
				if err = w.Enter(part); err != nil {
					t.Fatal(err)
				}
			}
		}
		if err = w.WriteErrorRegion(region); err != nil {
			t.Fatalf("parent scope for %v: %v", path, err)
		}
		out, err := w.Bytes()
		if err != nil || !bytes.Equal(out, []byte{0xa5}) {
			t.Fatalf("parent replay for %v: %v", path, err)
		}
	}
}
