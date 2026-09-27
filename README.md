# go-csn1

Go decoders and byte-exact encoders for CSN.1 definitions extracted from current 3GPP specifications. The compiler and authenticated source lock live in `github.com/gomaja/asn1-go-compiler`.

## Installation

```bash
go get github.com/gomaja/go-csn1@main
```

No tagged releases are published. Depend on the `main` branch and update with
`go get github.com/gomaja/go-csn1@main`.

## Usage

```go
import "github.com/gomaja/go-csn1/ts36331/uecapability"

decoded, err := uecapability.DecodeGERANPS(container)
if err != nil {
    return err
}
encoded, err := uecapability.EncodeGERANPS(decoded.Value)
```

`BitsConsumed` and `Tail` are reported by every decoder. Values contain an explicit `Wire` record for received spare bits, padding, implicit zero spans, truncation, list stop records, the transmitted boundary and tail. Every encode uses the generated bit encoder; received bytes are not retained for replay. An unchanged decoded value re-encodes byte for byte. Edits to receiver-inferred bits or inconsistent wire state are rejected; a caller may clear `Wire` to construct a new value with a new transmitted length.

Every named rest-octet decoder returns `runtime.ErrEmptyValue` for zero input; use `errors.Is` to identify an absent value. This is distinct from a malformed nonempty value. The printed CSN.1 grammars require at least one bit even where the containing message permits an IE of zero octets. Newly constructed IA values use the octet-aligned `0x2B` L/H spare-padding pattern from TS 44.060 V19.0.0 §11 and TS 24.007 V20.0.0 Annex B §B.1.2.2. Decoded values preserve received padding.

The table records whether the *prose* in each TS 44.018 V19.0.0 clause permits a zero-octet IE and the evidence available for each entry point. “Unstated” means the clause supplies no explicit length statement. All return `ErrEmptyValue` at zero input. “Pycrate padding” means a Release-13 decoder accepted a simple padding vector; it is narrower than a structural comparison. Wireshark's RR dissector reaches SI7/8/18/19/20 messages but exposes only their raw message elements, so it cannot serve as a field oracle for those five values.

| Clause | Printed rest-octet entry point | Prose permits zero? | Evidence |
| --- | --- | --- | --- |
| §10.5.2.16 | IA Rest Octets | Yes, 0–11 octets | Pycrate padding vector |
| §10.5.2.17 | IAR Rest Octets | No, 3 octets | Pycrate padding vector |
| §10.5.2.18 | IAX Rest Octets | Yes, 0–4 octets | Pycrate padding vector |
| §10.5.2.22c | NT/N Rest Octets | No, 20 octets | Pycrate padding vector |
| §10.5.2.23 | P1 Rest Octets | Yes, 0–17 octets | Pycrate structural; tshark partial |
| §10.5.2.24 | P2 Rest Octets | No, 1–11 octets | Pycrate padding vector |
| §10.5.2.25 | P3 Rest Octets | No, 3 octets | Pycrate padding vector |
| §10.5.2.32 | SI1 Rest Octets | No, 1 octet | Pycrate padding vector |
| §10.5.2.33 | SI2bis Rest Octets | No, 1 octet | Pycrate padding vector |
| §10.5.2.33a | SI2ter Rest Octets | No, 4 octets | Pycrate padding vector |
| §10.5.2.33b | SI2quater Rest Octets | No, 20 octets | Pycrate padding; tshark partial |
| §10.5.2.33c | SI2n Rest Octets | No, 20 octets | Pycrate padding vector |
| §10.5.2.34 | SI3 Rest Octet | No, 4 octets | Pycrate padding vector |
| §10.5.2.35 | SI4 Rest Octets | Yes, 0–10 octets | Pycrate padding vector |
| §10.5.2.35 | SI4 Rest Octets_O | Yes, SI4 variant | Spec vectors only |
| §10.5.2.35 | SI4 Rest Octets_S | Yes, SI4 variant | Spec vectors only |
| §10.5.2.35a | SI6 rest octets | No, 7 octets | Pycrate padding vector |
| §10.5.2.36 | SI7 Rest Octets | No, 20 octets | Spec vectors, both ACS layouts; tshark raw only |
| §10.5.2.37 | SI8 Rest Octets | No, 20 octets | Spec vectors, both ACS layouts; tshark raw only |
| §10.5.2.37a | SI9 rest octets | No, 17 octets | Pycrate padding vector |
| §10.5.2.37b | SI 13 Rest Octets | No, 20 octets | Pycrate padding vector |
| §10.5.2.37e | SI16 Rest Octets | No, 20 octets | Pycrate padding vector |
| §10.5.2.37f | SI17 Rest Octets | No, 20 octets | Pycrate padding vector |
| §10.5.2.37g | SI 19 Rest Octets | No, 20 octets | Spec group vector; pycrate differs; tshark raw only |
| §10.5.2.37h | SI 18 Rest Octets | No, 20 octets | Pycrate structural on full IE; spec terminal vector; tshark raw only |
| §10.5.2.37i | SI 20 Rest Octets | No, 20 octets | SI18 pycrate oracle by §.37i reuse; spec terminal vector; tshark raw only |
| §10.5.2.37j | SI14 Rest Octets | No, 16 octets | Pycrate padding vector |
| §10.5.2.37k | SI15 Rest Octets | No, 20 octets | Pycrate padding vector |
| §10.5.2.37l | SI 13alt Rest Octets | No, 20 octets | Pycrate padding vector |
| §10.5.2.37m | SI 21 Rest Octets | No, 20 octets | Pycrate padding vector |
| §10.5.2.37n | SI 22 Rest Octets | No, 20 octets | Pycrate padding vector |
| §10.5.2.37o | SI 23 Rest Octets | No, 20 octets | Pycrate padding vector |
| §10.5.2.44 | SI10 rest octets | No, 20 octets | Pycrate padding vector |
| §10.5.2.70 | SI10bis Rest Octets | Unstated | Pycrate padding vector |
| §10.5.2.71 | SI10ter Rest Octets | Unstated | Spec vector only |
| §10.5.2.78 | IPA Rest Octets | No, 19 octets | Pycrate padding vector |

`DecodeSI7RestOctets` and `DecodeSI8RestOctets` return `runtime.ErrContextRequired`, identifiable with `errors.Is` and carrying the specification, clause and printed name. TS 44.018 V19.0.0 §10.5.2.35 selects their layout using ACS in the containing SI4 message. Supply that bit as `runtime.SI4ACSZero` or `runtime.SI4ACSOne` to `DecodeSI7RestOctetsWithContext` / `DecodeSI8RestOctetsWithContext` and the matching `Encode…WithContext` function. `registry.DecodeWithContext` provides the same by printed name. ACS=1 selects `SI4RestOctetsO` plus `SI4RestOctetsS`; ACS=0 selects `SI4RestOctetsS` alone. A mismatched layout is rejected on encode.

`DecodeSI18RestOctets` implements the two stop conditions printed in TS 44.018 V19.0.0 §10.5.2.37h table 10.5.2.37h.1: a zero `NR_OF_CONTAINER_OCTETS` terminal record, or a fully used 20-octet SI message. `DecodeSI20RestOctets` uses the same grammar under §10.5.2.37i. Pycrate 0.7.11 accepts the fully used vector but rejects the printed zero-length terminal record. `DecodeSI19RestOctets` exposes the raw `NR_OF_REMAINING_CELLS` count 0–15, equal to the repeated-cell slice length, and interprets `FREQ_DIFF_LENGTH` as width 1–8. The printed grammar in §10.5.2.37g uses `* (val(NR_OF_REMAINING_CELLS))`; its "Range 1-16" prose conflicts with that repetition and with pycrate `si_19_rest_octets.py` and Wireshark `packet-gsm_rlcmac.c` COMPACT neighbour-cell repetitions. Pycrate 0.7.11 differs on a nonempty group vector. These differences are recorded in the compiler handoff rather than hidden by a compatibility path.

`measurement.DecodeEnhancedMeasurementReport` enforces the five-bit message type `00100` from TS 44.018 V19.0.0 §10.4 table 10.4.2 and §9.1.55. Message type `00101` identifies Measurement Information and is rejected by both the EMR decoder and encoder.

`registry.Lookup(specification, printedName)` succeeds only when the name is unique in that specification. `registry.LookupClause(specification, clause, printedName)` resolves a clause-qualified definition. The canonical key is the specification number, clause and definition name exactly as printed. Each module contains exactly one version per standard, so `Version` stays descriptor metadata and is not a key component. Consumers track `main`, which moves to the current pinned version without carrying historical versions. For example, `A5 bits` occurs in two TS 24.008 clauses with different layouts, so an unqualified lookup returns both candidate clauses in an ambiguity error.

The same printed `PEO IMM Cell Group Details struct` appears in TS 44.018 §§10.5.2.16–18 and .25. Its Go type names receive a clause suffix such as `Clause105216`, while registry keys retain the printed name and exact clause. `restoctets.Lookup` rejects that unqualified name as ambiguous; `restoctets.LookupClause` resolves it.

The compiler turns a printed CSN.1 name into a Go identifier by joining runs of letters and digits, capitalizing each run, preserving existing uppercase runs, and prefixing `N` when the result begins with a digit. Anonymous optionals with one labeled value become pointers named after that value, such as `A5Bits *A5Bits`. An optional with several values becomes a pointer to a content-named `Group` struct. Multi-way choices use a content-named type and alternative fields named from labels unique to each branch; unlabeled branches use their bit pattern, such as `Alt01`. Repetitions contain element structs. Repeated labels in the same parent receive `Variant2`, `Variant3`, and so on; inserting a field with a different label does not rename existing public types. Repeated printed names across clauses receive deterministic clause suffixes in Go; an unresolved identifier collision stops generation.

## Specification coverage

Packages marked **[compiled]** contain generated Go bindings. Planned packages are placeholders for CSN.1 content identified in the cited current specifications.

| Package | Source | Definition or content | Status |
| --- | --- | --- | --- |
| `ts24008/classmark` | TS 24.008 V20.1.0 §10.5.1.7 | Classmark 3 value part | **[compiled]** |
| `ts24008/msrac` | TS 24.008 V20.1.0 §10.5.5.12a | MS RA capability value part | **[compiled]** |
| `ts36331/uecapability` | TS 36.331 V19.4.0, UE-CapabilityRAT-ContainerList field descriptions; TS 24.008 V20.1.0 §10.5.1.6 | `geran-cs` and `geran-ps` containers, including Classmark 2 | **[compiled]** |
| `ts24008/msnetcap` | TS 24.008 V20.1.0 §10.5.5.12 | MS Network Capability value part | planned |
| `ts44018/restoctets` | TS 44.018 V19.0.0 §§10.5.2.16–18, .22c–25, .32–35a, .37a–b, .37e–f, .37j–o, .44, .70–71, .78 | IA, IAR, IAX, NT/N, P1–P3, SI1, SI2bis, SI2ter, SI2quater, SI2n, SI3, SI4, SI6, SI9, SI10, SI10bis, SI10ter, SI13, SI13alt, SI14–17, SI21–23 and IPA with local definitions | **[compiled]** |
| `ts44018/restoctets` | TS 44.018 V19.0.0 §§10.5.2.36–37, .37g–i | SI7 and SI8 with explicit SI4 ACS context; SI18, SI19 and SI20 direct codecs | **[compiled]** |
| `ts44018/measurement` | TS 44.018 V19.0.0 §9.1.55 | Enhanced Measurement Report body and local definitions | **[compiled]** |
| `ts44060/ies` | TS 44.060 V19.0.0 §§12.5.2, 12.8, 12.9a, 12.10a, 12.10d, 12.10f, 12.12, 12.24, 12.33, 12.36–41, 12.57–59, 12.61 | Twenty-seven dependency definitions used by IA, P2, SI13, SI13alt, SI2quater, SI23 and EMR | **[compiled]** |
| `ts44060/rlcmac` | TS 44.060 V19.0.0 §11 | RLC/MAC control messages | planned |

The module has no tags or releases. Consumers track `main` at a pinned commit.

## License

Licensed under the [Apache License, Version 2.0](LICENSE). See [NOTICE](NOTICE) for the
attribution of the 3GPP specifications from which the packages are generated.
