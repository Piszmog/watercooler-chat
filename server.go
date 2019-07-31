package main

import (
	"github.com/google/uuid"
	"github.com/reiver/go-oi"
	"github.com/reiver/go-telnet"
	"strings"
)

var clients = make(map[string]handler)

type handler struct {
	writer telnet.Writer
}

func (handler handler) ServeTELNET(ctx telnet.Context, w telnet.Writer, r telnet.Reader) {
	handler.writer = w
	var buffer [1]byte
	p := buffer[:]
	id := uuid.New().String()
	clients[id] = handler
	builder := strings.Builder{}
	builder.WriteString(id)
	builder.WriteString(":")
	builder.WriteString(" ")
	for {
		n, err := r.Read(p)
		if n > 0 {
			bytes := p[:n]
			if bytes[0] == '\n' {
				continue
			} else if bytes[0] == '\r' {
				builder.WriteByte('\n')
				input := builder.String()
				for _, handler := range clients {
					oi.LongWriteString(handler.writer, input)
				}
				builder.Reset()
				builder.WriteString(id)
				builder.WriteString(":")
				builder.WriteString(" ")
			} else {
				builder.Write(bytes)
			}
		}
		if nil != err {
			break
		}
	}
}

func main() {
	var handler = handler{}
	err := telnet.ListenAndServe(":5555", handler)
	if nil != err {
		//@TODO: Handle this error better.
		panic(err)
	}
}
