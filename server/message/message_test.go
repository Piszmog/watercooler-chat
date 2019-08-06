package message

import (
	"testing"
	"time"
)

func TestChatMessage_LogMessage(t *testing.T) {
	chatMessage := ChatMessage{
		Timestamp: time.Date(2019, 1, 1, 1, 1, 1, 0, time.UTC),
		Room:      "test",
		Sender:    "tester",
		Value:     "Hello from tester!",
	}
	logMessage := chatMessage.LogMessage()
	if logMessage != "chat message - [test tester] Hello from tester!" {
		t.Fatalf("Log message not match expected value. Actual value: %s", logMessage)
	}
}

func TestChatMessage_RoomMessage(t *testing.T) {
	chatMessage := ChatMessage{
		Timestamp: time.Date(2019, 1, 1, 1, 1, 1, 0, time.UTC),
		Room:      "test",
		Sender:    "tester",
		Value:     "Hello from tester!",
	}
	logMessage := chatMessage.RoomMessage()
	if logMessage != "[01:01 UTC tester]: Hello from tester!" {
		t.Fatalf("Log message not match expected value. Actual value: %s", logMessage)
	}
}
