package message

import "strings"

// Buffer creates message with a buffer, saving on constant string creation.
type Buffer struct {
	BufferBytes []byte
	Builder     strings.Builder
}

// WriteByte writes the provided byte to the message builder.
func (buffer *Buffer) WriteByte(b byte) {
	buffer.Builder.WriteByte(b)
}

// String converts the written bytes to a string message.
func (buffer *Buffer) String() string {
	message := buffer.Builder.String()
	buffer.Builder.Reset()
	return message
}
