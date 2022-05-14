package string

import "testing"

func BenchmarkPlus(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Plus("hello", 1000)
	}
}

func BenchmarkStrbuilder(b *testing.B) {
	for i := 0; i < b.N; i++ {
		StrBuilder("hello", 1000)
	}
}

func BenchmarkByteBuffer(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ByteBuffer("hello", 1000)
	}
}
