package main

import "strings"

type messageBuffer struct {
	bufferBytes []byte
	builder     strings.Builder
}

func (buffer *messageBuffer) writeByte(b byte) {
	buffer.builder.WriteByte(b)
}

func (buffer *messageBuffer) string() string {
	message := buffer.builder.String()
	buffer.builder.Reset()
	return message
}
