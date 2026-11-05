package registry

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	goruntime "runtime"
	"strconv"
	"sync"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

var retainedIndex lookupIndex

func TestLookupIndexStorageBound(t *testing.T) {
	const count = 1024
	entries := make([]runtime.Descriptor, count)
	for i := range entries {
		entries[i] = runtime.Descriptor{Standard: "synthetic", Version: "1", Clause: fmt.Sprint(i), Name: fmt.Sprint(i)}
	}
	goruntime.GC()
	var before, after goruntime.MemStats
	goruntime.ReadMemStats(&before)
	retainedIndex = newLookupIndex(entries)
	goruntime.ReadMemStats(&after)
	if got := after.TotalAlloc - before.TotalAlloc; got > 512*count {
		t.Fatalf("index allocated %d bytes for %d entries; maximum %d", got, count, 512*count)
	}
	goruntime.KeepAlive(entries)
}

func TestRegistryLazyInitialization(t *testing.T) {
	if os.Getenv("REGISTRY_FIRST_USE_CHILD") == "idle" {
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestRegistryLazyInitialization$")
	cmd.Env = append(os.Environ(), "REGISTRY_FIRST_USE_CHILD=idle", "GODEBUG=inittrace=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("idle import: %v: %s", err, out)
	}
	match := regexp.MustCompile(`(?m)^init github.com/gomaja/go-csn1/registry .* ([0-9]+) bytes,`).FindSubmatch(out)
	if len(match) == 0 {
		t.Fatal("registry initialization trace missing")
	}
	bytes, err := strconv.Atoi(string(match[1]))
	if err != nil {
		t.Fatal(err)
	}
	if bytes > 32768 {
		t.Fatalf("idle registry import allocated %d bytes; maximum 32768", bytes)
	}
}

func TestRegistryConcurrentFirstUse(t *testing.T) {
	if os.Getenv("REGISTRY_FIRST_USE_CHILD") != "concurrent" {
		cmd := exec.Command(os.Args[0], "-test.run=^TestRegistryConcurrentFirstUse$")
		cmd.Env = append(os.Environ(), "REGISTRY_FIRST_USE_CHILD=concurrent")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("concurrent first use: %v: %s", err, out)
		}
		return
	}
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := range 48 {
		wg.Go(func() {
			<-start
			const standard = "TS 24.008"
			const clause = "10.5.1.6"
			const name = "Mobile Station Classmark 2 value part"
			var d runtime.Descriptor
			var err error
			switch i % 6 {
			case 0:
				d, err = Lookup(standard, name)
			case 1:
				d, err = LookupClause(standard, clause, name)
			case 2:
				d, err = LookupContext(standard, clause, name, "")
			case 3:
				entries := Descriptors()
				if len(entries) != 632 {
					t.Errorf("catalogue length %d", len(entries))
				}
				return
			case 4:
				if SourceOrigin(runtime.Descriptor{}) != 0 {
					t.Error("unknown origin")
				}
				return
			case 5:
				_, err = Decode(standard, name, []byte{0x20, 0, 0})
				if err != nil {
					t.Error(err)
				}
				return
			}
			if err != nil || d.Name != name {
				t.Errorf("first lookup: %s, %v", d.Name, err)
			}
		})
	}
	close(start)
	wg.Wait()
}

func TestSourceOriginUsesCatalogue(t *testing.T) {
	entry := runtime.Descriptor{Standard: "synthetic", Version: "1", Clause: "1", Name: "Value", SourceOrigin: 14}
	installLookupCatalogue(t, []runtime.Descriptor{entry})
	entry.SourceOrigin = 0
	if got := SourceOrigin(entry); got != 14 {
		t.Fatalf("recorded origin %d; want 14", got)
	}
	entry.Version = "other"
	if got := SourceOrigin(entry); got != 0 {
		t.Fatalf("unrecorded version origin %d", got)
	}
}
