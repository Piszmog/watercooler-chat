package main

import (
	"fmt"
	"time"
)

const timestampFormat = "15:04 MST"

type chatMessage struct {
	Timestamp time.Time `json:"timestamp"`
	Room      string    `json:"room"`
	Sender    string    `json:"sender"`
	Value     string    `json:"value"`
}

func (message chatMessage) logMessage() string {
	return fmt.Sprintf("chat message - [%s %s] %s", message.Room, message.Sender, message.Value)
}

func (message chatMessage) roomMessage() string {
	return fmt.Sprintf("[%s %s]: %s", message.Timestamp.Format(timestampFormat), message.Sender, message.Value)
}
