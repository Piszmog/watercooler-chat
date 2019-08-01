package main

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/reiver/go-oi"
	"github.com/reiver/go-telnet"
	"log"
	"os"
	"strings"
	"time"
)

var logger *log.Logger
var clients = make(map[string]handler)

type handler struct {
	id     string
	writer telnet.Writer
}

func (handler handler) ServeTELNET(ctx telnet.Context, w telnet.Writer, r telnet.Reader) {
	id := uuid.New().String()
	handler.writer = w
	handler.id = id
	for _, handler := range clients {
		oi.LongWriteString(handler.writer, "client "+id+" has entered\n")
	}
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
				logger.Print(input)
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
			for _, handler := range clients {
				oi.LongWriteString(handler.writer, "client "+id+" has left\n")
			}
			break
		}
	}
}

func main() {
	const logFile = "log.txt"
	const address = ":5555"
	file, err := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("failed to open log file %s: %+v", logFile, err)
	}
	defer closeFile(file)
	logger = log.New(file, "", 0)
	var handler = handler{}
	err = telnet.ListenAndServe(address, handler)
	if nil != err {
		closeFile(file)
		log.Fatalf("failed to start server at address %s: %+v", address, err)
	}
}

func closeFile(file *os.File) {
	err := file.Close()
	if err != nil {
		fmt.Printf("failed to close file: %+v", err)
	}
}
