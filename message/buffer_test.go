package message

import (
	"strings"
	"testing"
)

func TestBuffer_String(t *testing.T) {
	buffer := Buffer{
		BufferBytes: make([]byte, 1, 1),
		Builder:     strings.Builder{},
	}
	buffer.WriteByte('h')
	buffer.WriteByte('e')
	buffer.WriteByte('l')
	buffer.WriteByte('l')
	buffer.WriteByte('o')
	message := buffer.String()
	if message != "hello" {
		t.Fatalf("buffer message does not match expected value. Actual Value %s", message)
	}
}
