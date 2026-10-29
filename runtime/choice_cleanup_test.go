package runtime

import (
	"errors"
	"strconv"
	"strings"
	"testing"
)

func TestChoiceFailurePathSurvivesScratchReuse(t *testing.T) {
	var retained []error
	var paths []string
	for _, depth := range []int{128, 1, 16, 0, 127, 2} {
		r := NewReader(nil)
		parts := make([]string, depth)
		for i := range parts {
			parts[i] = strings.Repeat("p", i+1)
			if err := r.Enter(parts[i]); err != nil {
				t.Fatal(err)
			}
		}
		var trials ChoiceErrors
		arm := trials.ForkScoped(r)
		_, err := arm.ReadUint(1)
		trials.Track(err)
		retained = append(retained, trials.Keep(r, err))
		trials.Release()
		paths = append(paths, strings.Join(parts, "/"))
		for i, kept := range retained {
			var cause *DecodeError
			if !errors.As(kept, &cause) || cause.Path != paths[i] {
				t.Fatalf("depth %d changed retained path %d: %v", depth, i, kept)
			}
		}
	}
}

func BenchmarkChoiceFailurePathCleanup(b *testing.B) {
	for _, depth := range []int{1, 8, 128} {
		name := "Depth" + strconv.Itoa(depth)
		b.Run(name, func(b *testing.B) {
			r := NewReader(nil)
			for range depth {
				if err := r.Enter("field"); err != nil {
					b.Fatal(err)
				}
			}
			var owner ChoiceErrors
			defer owner.Release()
			arm := owner.ForkScoped(r)
			b.ReportAllocs()
			for b.Loop() {
				var failures ChoiceErrors
				failures.Track(arm.Error(InvalidValue, "rejected"))
				failures.Release()
			}
		})
	}
}
