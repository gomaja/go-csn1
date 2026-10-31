package controlblock

import (
	"errors"
	"fmt"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

func TestExtendedSegmentationCapabilityStates(t *testing.T) {
	// TS 44.060 V19.0.0 §§11.2.7, 11.2.29 and 11.2.31 permit
	// either Downlink Dual Carrier or Downlink Multi Carrier support.
	yes, no := true, false
	states := []*bool{nil, &no, &yes}
	for _, clause := range []string{"11.2.7", "11.2.7a", "11.2.29", "11.2.29a", "11.2.31", "11.2.31a", "11.2.37"} {
		for d, dldc := range states {
			for m, dlmc := range states {
				t.Run(fmt.Sprintf("%s/DLDC%d/DLMC%d", clause, d, m), func(t *testing.T) {
					ctx := ordinaryContext()
					ctx.DLDC, ctx.DLMC = dldc, dlmc
					multi := clause == "11.2.7" || clause == "11.2.29" || clause == "11.2.31"
					allowed := d == 2 || multi && m == 2
					unknown := !allowed && (d == 0 || multi && m == 0)
					for _, count := range []int{2, 3, 9} {
						_, err := Reassemble(ctx, clause, fragments(count))
						switch {
						case allowed || count == 2:
							if err != nil {
								t.Fatalf("%d blocks: %v", count, err)
							}
						case unknown:
							var required *runtime.ContextRequiredError
							if !errors.Is(err, runtime.ErrContextRequired) || !errors.As(err, &required) || required.Clause != clause {
								t.Fatalf("%d blocks: want located context-required error, got %T: %v", count, err, err)
							}
						default:
							var refused *SegmentError
							if errors.Is(err, runtime.ErrContextRequired) || !errors.As(err, &refused) || refused.Clause != clause {
								t.Fatalf("%d blocks: want source-permission refusal, got %T: %v", count, err, err)
							}
						}
					}
				})
			}
		}
	}
}
