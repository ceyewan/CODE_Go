package string

import (
	"bytes"
	"strings"
)

func Plus(str string, n int) string {
	s := ""
	for i := 0; i < n; i++ {
		s += str
	}
	return s
}

func StrBuilder(str string, n int) string {
	var build strings.Builder
	for i := 0; i < n; i++ {
		build.WriteString(str)
	}
	return build.String()
}

func ByteBuffer(str string, n int) string {
	buf := new(bytes.Buffer)
	for i := 0; i < n; i++ {
		buf.WriteString(str)
	}
	return buf.String()
}
