package main

import (
	"bytes"
	"github.com/reiver/go-telnet"
	"strings"
	"testing"
)

func TestChatUser_ServeTELNET(t *testing.T) {
	server = CreateServer()
	//
	// reset
	//
	defer func() {
		server = CreateServer()
	}()
	user := ChatUser{}
	reader := strings.NewReader("Tester\n\rTest Room\n\r-q\n\r")
	var b bytes.Buffer
	user.ServeTELNET(telnet.NewContext(), &b, reader)
	expectedServerMessages := `What is your Name?
Existing rooms:

What room would you like to enter (if room is not listed, room will be created)? 
Users currently in the room:

Available commands:
-r ${room Name} -- change to the specified room. Creates room if doesn't exist
-b ${user Name} -- to block messages from the specified user
-u ${user Name} -- to Unblock messages from the specified user
-lr             -- to list all existing rooms
-lr             -- to list all users in the current room
-lr             -- to list all users currently blocked
-q              -- to quit the chat
-h              -- to list all available commands

Tester has entered
Welcome to room Test Room! You may begin chatting with the users.
Quiting...
`
	if b.String() != expectedServerMessages {
		t.Fatal("server did not write the expected format to the user")
	}
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
