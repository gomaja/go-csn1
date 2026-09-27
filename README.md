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

`restoctets.DecodeIARestOctets` returns `runtime.ErrEmptyValue` for zero input; use `errors.Is` to distinguish an absent IE from malformed nonempty content. TS 44.018 V19.0.0 §10.5.2.16 permits a zero-octet length in the containing Immediate Assignment message, but the printed CSN.1 value grammar requires a two-bit discriminator. Newly constructed IA values use the octet-aligned `0x2B` L/H spare-padding pattern from TS 44.060 V19.0.0 §11 and TS 24.007 V20.0.0 Annex B §B.1.2.2. Decoded values preserve received padding.

`DecodeIARRestOctets` requires exactly three octets (TS 44.018 V19.0.0 §10.5.2.17); zero input is a length error. `DecodeIAXRestOctets` returns `runtime.ErrEmptyValue` for zero input because §10.5.2.18 permits a 0–4 octet IE length, while its value grammar requires a bit. The remaining rest-octet entry points will document their own length rules as they are compiled.

`registry.Lookup(specification, printedName)` succeeds only when the name is unique in that specification. `registry.LookupClause(specification, clause, printedName)` resolves a clause-qualified definition. The canonical key is the specification number, clause and definition name exactly as printed; the version is metadata. For example, `A5 bits` occurs in two TS 24.008 clauses with different layouts, so an unqualified lookup returns both candidate clauses in an ambiguity error.

The same printed `PEO IMM Cell Group Details struct` appears in TS 44.018 §§10.5.2.16–18. Its Go type names receive a `Clause105216`, `Clause105217`, or `Clause105218` suffix, while registry keys retain the printed name and exact clause. `restoctets.Lookup` rejects that unqualified name as ambiguous; `restoctets.LookupClause` resolves it.

The compiler turns a printed CSN.1 name into a Go identifier by joining runs of letters and digits, capitalizing each run, preserving existing uppercase runs, and prefixing `N` when the result begins with a digit. Anonymous optionals with one labeled value become pointers named after that value, such as `A5Bits *A5Bits`. An optional with several values becomes a pointer to a content-named `Group` struct. Multi-way choices use a content-named type and alternative fields named from labels unique to each branch; unlabeled branches use their bit pattern, such as `Alt01`. Repetitions contain element structs. Repeated labels in the same parent receive `Variant2`, `Variant3`, and so on; inserting a field with a different label does not rename existing public types. The current packages each contain one clause's definitions; an identifier collision within a package stops generation.

## Specification coverage

Packages marked **[compiled]** contain generated Go bindings. Planned packages are placeholders for CSN.1 content identified in the cited current specifications.

| Package | Source | Definition or content | Status |
| --- | --- | --- | --- |
| `ts24008/classmark` | TS 24.008 V20.1.0 §10.5.1.7 | Classmark 3 value part | **[compiled]** |
| `ts24008/msrac` | TS 24.008 V20.1.0 §10.5.5.12a | MS RA capability value part | **[compiled]** |
| `ts36331/uecapability` | TS 36.331 V19.4.0, UE-CapabilityRAT-ContainerList field descriptions; TS 24.008 V20.1.0 §10.5.1.6 | `geran-cs` and `geran-ps` containers, including Classmark 2 | **[compiled]** |
| `ts24008/msnetcap` | TS 24.008 V20.1.0 §10.5.5.12 | MS Network Capability value part | planned |
| `ts44018/restoctets` | TS 44.018 V19.0.0 §§10.5.2.16–18 | IA, IAR and IAX Rest Octets and their local definitions | **[compiled]** |
| `ts44018/restoctets` | TS 44.018 V19.0.0 §10.5.2, except §§10.5.2.16–18 | Remaining RR rest-octet information elements | planned |
| `ts44018/emr` | TS 44.018 V19.0.0 §9.1.55 | Enhanced Measurement Report body | planned |
| `ts44060/ies` | TS 44.060 V19.0.0 §§12.5.2, 12.10d, 12.10f, 12.12, 12.33 | Five table and CSN.1 IEs referenced by IA Rest Octets | **[compiled]** |
| `ts44060/rlcmac` | TS 44.060 V19.0.0 §11 | RLC/MAC control messages | planned |

The module has no tags or releases. Consumers track `main` at a pinned commit.
