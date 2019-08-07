package message

import (
	"fmt"
	"time"
)

const timestampFormat = "15:04 MST"

// Query is use to query messages from a room.
type Query struct {
	Start      time.Time
	End        time.Time
	RoomName   string
	SenderName string
}

// ChatMessage is the message that a user sends to the room.
type ChatMessage struct {
	Timestamp time.Time `json:"timestamp"`
	Room      string    `json:"room"`
	Sender    string    `json:"sender"`
	Value     string    `json:"value"`
}

// LogMessage formats the message to a log friendly message.
func (message ChatMessage) LogMessage() string {
	return fmt.Sprintf("chat message - [%s %s] %s", message.Room, message.Sender, message.Value)
}

// RoomMessage formats the message to a room friendly message.
func (message ChatMessage) RoomMessage() string {
	return fmt.Sprintf("[%s %s]: %s", message.Timestamp.Format(timestampFormat), message.Sender, message.Value)
}
