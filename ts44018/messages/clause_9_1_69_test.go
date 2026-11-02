package messages

import (
	"bytes"
	"encoding/hex"
	"github.com/gomaja/go-csn1/runtime"
	"testing"
)

// TS 44.018 V19.0.0 §9.1.69 tables 9.1.69.1 and 9.1.69.3-5,
// paragraphs 12806-12808 and 12821-12913; §10.4 table 10.4.5.
// The four-bit tag plus 84-bit sequence equals the 88-bit EC-PICH
// information block in TS 45.003 V19.0.0 §5.2b.1.
var ecPagingIndicationVectors = []struct{ wire, sequence string }{
	{"8000000000000000000000", "0000000000000000000000"},
	{"8fffffffffffffffffffff", "fffffffffffffffffffff0"},
	{"811ca614b8041b61b23ef5", "11ca614b8041b61b23ef50"},
	{"82d08ca80e74fb1ebe2fb0", "2d08ca80e74fb1ebe2fb00"},
	{"808755be22383d8a039578", "08755be22383d8a0395780"},
	{"83aa00bca4c96e62e02bc0", "3aa00bca4c96e62e02bc00"},
	{"823508ea727c130a105c54", "23508ea727c130a105c540"},
	{"814fadf8a157ceb0a95381", "14fadf8a157ceb0a953810"},
	{"8257efa0d77e79fe0a00d9", "257efa0d77e79fe0a00d90"},
	{"81a341ef26d95d796ab80c", "1a341ef26d95d796ab80c0"},
	{"808755be22383d8a039578", "08755be22383d8a0395780"},
	{"83aa00bca4c96e62e02bc0", "3aa00bca4c96e62e02bc00"},
	{"800b08eae0a76f2bac58a0", "00b08eae0a76f2bac58a00"},
	{"81bfad2d20746355cede69", "1bfad2d20746355cede690"},
	{"82fd738a1b4868c4fa6c73", "2fd738a1b4868c4fa6c730"},
	{"818c059efd4fb91db1dbff", "18c059efd4fb91db1dbff0"},
	{"800b08eae0a76f2bac58a0", "00b08eae0a76f2bac58a00"},
	{"81bfad2d20746355cede69", "1bfad2d20746355cede690"},
	{"8281251686f65f2ec953e7", "281251686f65f2ec953e70"},
	{"82537a42eb0441c6723491", "2537a42eb0441c67234910"},
	{"80acc4d69d2a661239e44d", "0acc4d69d2a661239e44d0"},
	{"82b61cb849974c216b52c0", "2b61cb849974c216b52c00"},
	{"81379c990a8a495a1f60d4", "1379c990a8a495a1f60d40"},
	{"8061ff13db486d1e955a93", "061ff13db486d1e955a930"},
	{"80a9c8fc109ce886516c51", "0a9c8fc109ce886516c510"},
	{"80981fe5ea480b96b8944f", "0981fe5ea480b96b8944f0"},
	{"80f57dee93d0b8702ba08e", "0f57dee93d0b8702ba08e0"},
	{"81aa99ba5308c538184d56", "1aa99ba5308c538184d560"},
	{"83af0c49aa07050e7dfa6a", "3af0c49aa07050e7dfa6a0"},
	{"81aa99ba5308c538184d56", "1aa99ba5308c538184d560"},
	{"8281251686f65f2ec953e7", "281251686f65f2ec953e70"},
	{"82537a42eb0441c6723491", "2537a42eb0441c67234910"},
	{"827d9680d6354e13f6bfab", "27d9680d6354e13f6bfab0"},
	{"822f25ae2f36f33a662f6d", "22f25ae2f36f33a662f6d0"},
	{"82f0df31208492bda32224", "2f0df31208492bda322240"},
	{"83da29b045c1ee7304ce00", "3da29b045c1ee7304ce000"},
	{"80f79db1f6daf99cee6263", "0f79db1f6daf99cee62630"},
	{"83a1017319ca951a1dc7b7", "3a1017319ca951a1dc7b70"},
	{"8193693bbc1524e90c4421", "193693bbc1524e90c44210"},
	{"83251c483ae535eec90694", "3251c483ae535eec906940"},
	{"83d45cf225879d2d3735ff", "3d45cf225879d2d3735ff0"},
	{"828d6cb25cfd54adb2f6c2", "28d6cb25cfd54adb2f6c20"},
	{"830343568115198a453012", "30343568115198a4530120"},
	{"83df1babdc006f24d0ddf8", "3df1babdc006f24d0ddf80"},
	{"830343568115198a453012", "30343568115198a4530120"},
	{"83df1babdc006f24d0ddf8", "3df1babdc006f24d0ddf80"},
	{"82f0df31208492bda32224", "2f0df31208492bda322240"},
	{"83da29b045c1ee7304ce00", "3da29b045c1ee7304ce000"},
	{"81617faf71065250ba4bb0", "1617faf71065250ba4bb00"},
	{"8385f9d1d11dd975a13b50", "385f9d1d11dd975a13b500"},
}

func TestECPagingIndicationSource(t *testing.T) {
	for _, tc := range ecPagingIndicationVectors {
		wire := ecRRWire(t, tc.wire)
		want := ECPagingIndication{MessageType: 8, PagingIndicationSequence: runtime.BitString{Bytes: ecRRWire(t, tc.sequence), BitLength: 84}}
		got, err := DecodeECPagingIndication(wire)
		if err != nil || !runtime.SemanticallyEqual(got.Value, want) {
			t.Fatalf("typed decode %#v %v", got, err)
		}
		ecRRCheck(t, "9.1.69", "EC Paging Indication", wire, want, 88)
		ecRRBound(t, wire, 88, func(r *runtime.Reader) error { _, err := DecodeECPagingIndicationFrom(r); return err })
		canonical, err := EncodeECPagingIndicationCanonical(want)
		if err != nil || !bytes.Equal(canonical, wire) {
			t.Fatalf("fixed canonical %x %v", canonical, err)
		}
		got.Value.PagingIndicationSequence.Bytes[0] ^= 0x80
		edited, err := EncodeECPagingIndication(got.Value)
		if err != nil || bytes.Equal(edited, wire) {
			t.Fatal("edit lost", err)
		}
		again, err := DecodeECPagingIndication(edited)
		if err != nil || !runtime.SemanticallyEqual(again.Value, got.Value) {
			t.Fatal("edited decode", err)
		}
	}
}
func TestECPagingIndicationInvalidFields(t *testing.T) {
	for _, n := range []int{0, 83, 85} {
		v := ECPagingIndication{MessageType: 8, PagingIndicationSequence: runtime.BitString{Bytes: bytes.Repeat([]byte{255}, 11), BitLength: n}}
		if _, err := EncodeECPagingIndicationCanonical(v); err == nil {
			t.Fatalf("accepted sequence width %d", n)
		}
	}
}
func FuzzECPagingIndicationReplay(f *testing.F) {
	for _, tc := range ecPagingIndicationVectors {
		b, err := hex.DecodeString(tc.wire)
		if err != nil {
			f.Fatal(err)
		}
		f.Add(b)
	}
	f.Fuzz(func(t *testing.T, wire []byte) {
		if len(wire) > 256 {
			return
		}
		d, err := DecodeECPagingIndication(wire)
		if err != nil {
			return
		}
		b, err := EncodeECPagingIndication(d.Value)
		if err != nil || !bytes.Equal(b, wire) {
			t.Fatalf("replay %x %v", b, err)
		}
	})
}
