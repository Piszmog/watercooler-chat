package main

import (
	"github.com/google/uuid"
	"github.com/reiver/go-oi"
	"github.com/reiver/go-telnet"
	"strings"
	"time"
)

var clients = make(map[string]handler)

type handler struct {
	id     string
	writer telnet.Writer
}

func (handler handler) ServeTELNET(ctx telnet.Context, w telnet.Writer, r telnet.Reader) {
	id := uuid.New().String()
	handler.writer = w
	handler.id = id
	clients[id] = handler
	var buffer [1]byte
	p := buffer[:]
	builder := strings.Builder{}
	builder.WriteString(id)
	builder.WriteString(":")
	builder.WriteString(" ")
	var timestamp string
	for {
		n, err := r.Read(p)
		if n > 0 {
			if len(timestamp) == 0 {
				timestamp = time.Now().Format("2006-01-02 15:04:05 MST")
			}
			bytes := p[:n]
			if bytes[0] == '\n' {
				continue
			} else if bytes[0] == '\r' {
				builder.WriteString(" ")
				builder.WriteString("(")
				builder.WriteString(timestamp)
				builder.WriteString(")")
				builder.WriteByte('\n')
				input := builder.String()
				for _, handler := range clients {
					if id == handler.id {
						continue
					}
					oi.LongWriteString(handler.writer, input)
				}
				timestamp = ""
				builder.Reset()
				builder.WriteString(id)
				builder.WriteString(":")
				builder.WriteString(" ")
			} else {
				builder.Write(bytes)
			}
		}
		if nil != err {
			delete(clients, id)
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
