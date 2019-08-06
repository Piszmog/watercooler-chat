package main

import (
	"bytes"
	"testing"
)

func TestChatUser_ServeTELNET(t *testing.T) {
}

func TestChatUser_ReceiveMessage(t *testing.T) {
	var b bytes.Buffer
	user := ChatUser{Name: "tester", writer: &b}
	user.ReceiveMessage("Message to user")
	if b.String() != "Message to user\n" {
		t.Fatal("message received by user is not as expected")
	}
}

func TestChatUser_SendMessage(t *testing.T) {
	user := ChatUser{Name: "tester"}
	room := CreateRoom("testRoom")
	user.SendMessage("Hello from user send test", &room)
	userMessage := <-room.messageChannel
	room.Close()
	if userMessage.Value != "Hello from user send test" {
		t.Fatal("message from user is not of expected value")
	}
}

func TestChatUser_IsBlocked(t *testing.T) {
	user := ChatUser{Name: "tester"}
	user.blockedUsers = make(map[string]bool)
	user.block("tester1")
	if !user.IsBlocked("tester1") {
		t.Fatal("expected user 'tester1' to be blocked")
	}
}
