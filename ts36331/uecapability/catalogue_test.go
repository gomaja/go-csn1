package uecapability

import (
	"testing"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts24008/classmark"
)

var catalogueLookupDescriptor runtime.Descriptor
var catalogueLookupError error

func TestLookupAllocationsMatchClassmark(t *testing.T) {
	classmarkAllocations := testing.AllocsPerRun(100, func() {
		catalogueLookupDescriptor, catalogueLookupError = classmark.Lookup("Mobile Station Classmark 2 value part")
	})
	if catalogueLookupError != nil {
		t.Fatal(catalogueLookupError)
	}
	for _, name := range []string{"geran-cs", "geran-ps"} {
		allocations := testing.AllocsPerRun(100, func() {
			catalogueLookupDescriptor, catalogueLookupError = Lookup(name)
		})
		if catalogueLookupError != nil {
			t.Fatal(catalogueLookupError)
		}
		t.Logf("Lookup(%q): %g allocations; Classmark: %g", name, allocations, classmarkAllocations)
		if allocations > classmarkAllocations {
			t.Errorf("Lookup(%q) allocates more than Classmark", name)
		}
	}
}
