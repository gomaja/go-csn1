package runtime

import "testing"

var benchmarkReader *Reader

func BenchmarkReaderFork(b *testing.B) {
	r := NewReader([]byte{0x2b, 0x2b})
	r.Set("N_E-UTRAN", 2)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		benchmarkReader = r.Fork()
	}
}

func BenchmarkReaderPath(b *testing.B) {
	r := NewReader([]byte{0x2b})
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if i%1000 == 0 {
			r = NewReader([]byte{0x2b})
		}
		_ = r.Enter("First")
		_ = r.Enter("Second")
		_ = r.Enter("Third")
		r.Leave()
		r.Leave()
		r.Leave()
	}
}
