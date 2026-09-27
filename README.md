# go-csn1

Go decoders and byte-exact encoders for CSN.1 definitions extracted from current 3GPP specifications. The compiler and authenticated source lock live in `github.com/gomaja/asn1-go-compiler`.

## Usage

```go
import "github.com/gomaja/go-csn1/ts36331/uecapability"

decoded, err := uecapability.DecodeGERANPS(container)
if err != nil {
    return err
}
encoded, err := uecapability.EncodeGERANPS(decoded.Value)
```

`BitsConsumed` and `Tail` are reported by every decoder. Values contain an explicit `Wire` record for received spare bits, padding, implicit extension, truncation and tail. An unchanged decoded value re-encodes byte for byte; an edited or newly constructed value runs through the generated bit encoder and validates its choices, lengths and constraints.

`registry.Lookup(specification, printedName)` succeeds only when the name is unique in that specification. `registry.LookupClause(specification, clause, printedName)` resolves a clause-qualified definition. The canonical key is the specification number, clause and definition name exactly as printed; the version is metadata. For example, `A5 bits` occurs in two TS 24.008 clauses with different layouts, so an unqualified lookup returns both candidate clauses in an ambiguity error.

The compiler turns a printed CSN.1 name into a Go identifier by joining runs of letters and digits, capitalizing each run, preserving existing uppercase runs, and prefixing `N` when the result begins with a digit. Anonymous structural nodes receive stable ordinal suffixes within their definition. The current packages each contain one clause's definitions; an identifier collision within a package stops generation.

## Specification coverage

Packages marked **[compiled]** contain generated Go bindings. Planned packages are placeholders for CSN.1 content identified in the cited current specifications.

| Package | Source | Definition or content | Status |
| --- | --- | --- | --- |
| `ts24008/classmark` | TS 24.008 V20.1.0 §10.5.1.7 | Classmark 3 value part | **[compiled]** |
| `ts24008/msrac` | TS 24.008 V20.1.0 §10.5.5.12a | MS RA capability value part | **[compiled]** |
| `ts36331/uecapability` | TS 36.331 V19.4.0, UE-CapabilityRAT-ContainerList field descriptions; TS 24.008 V20.1.0 §10.5.1.6 | `geran-cs` and `geran-ps` containers, including Classmark 2 | **[compiled]** |
| `ts24008/msnetcap` | TS 24.008 V20.1.0 §10.5.5.12 | MS Network Capability value part | planned |
| `ts44018/restoctets` | TS 44.018 V19.0.0 §10.5.2 | RR rest-octet information elements | planned |
| `ts44018/emr` | TS 44.018 V19.0.0 §9.1.55 | Enhanced Measurement Report body | planned |
| `ts44060/ies` | TS 44.060 V19.0.0 §11 | CSN.1 IEs referenced by RR definitions | planned |
| `ts44060/rlcmac` | TS 44.060 V19.0.0 §11 | RLC/MAC control messages | planned |

The module has no tags or releases. Consumers track `main` at a pinned commit.
