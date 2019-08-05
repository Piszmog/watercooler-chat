package room

import (
	"fmt"
	"time"
)

const timestampFormat = "15:04 MST"

type ChatM struct {
}

type ChatMessage struct {
	Timestamp time.Time `json:"timestamp"`
	Room      string    `json:"room"`
	Sender    string    `json:"sender"`
	Value     string    `json:"value"`
}

func (message ChatMessage) logMessage() string {
	return fmt.Sprintf("chat message - [%s %s] %s", message.Room, message.Sender, message.Value)
}

func (message ChatMessage) roomMessage() string {
	return fmt.Sprintf("[%s %s]: %s", message.Timestamp.Format(timestampFormat), message.Sender, message.Value)
}
