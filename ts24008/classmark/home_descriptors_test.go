package classmark

import (
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

var (
	homeLookupDescriptor runtime.Descriptor
	homeLookupError      error
)

func TestLookupRetainsOnlyReturnedDescriptor(t *testing.T) {
	for _, name := range []string{"Classmark 3 Value part", "Mobile Station Classmark 2 value part"} {
		if _, err := Lookup(name); err != nil {
			t.Fatal(err)
		}
		allocations := testing.AllocsPerRun(100, func() { homeLookupDescriptor, homeLookupError = Lookup(name) })
		if homeLookupError != nil {
			t.Fatal(homeLookupError)
		}
		if allocations > 1 {
			t.Fatalf("Lookup(%q) allocated %g times; maximum one returned descriptor", name, allocations)
		}
	}
}
