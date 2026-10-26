# Carrying-message integration tests

This separate module checks the local CSN.1 packages with the ASN.1 decoder of
their LTE carrying messages. Its go-asn1 dependency and Go 1.26.9 minimum apply
only to these tests; the main library module remains dependency-free.

From this directory, run:

```sh
GOWORK=off go mod tidy -diff
GO_CSN1_LTE_RRC_UU_JSON=/path/to/local/uu.json GOWORK=off go test -tags=integration -count=1 -v ./...
```

`replace github.com/gomaja/go-csn1 => ..` tests the adjacent library checkout.
The environment variable selects private local trace data. With it unset or the
file absent, the test skips. No trace bytes or decoded values belong in this
module.
