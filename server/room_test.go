package main

import (
	"bytes"
	"github.com/piszmog/watercooler-chat/server/message"
	"log"
	"os"
	"sync"
	"testing"
	"time"
)

func init() {
	logger = log.New(os.Stdout, "TEST: ", log.LstdFlags|log.LUTC)
}

func TestChatRoom_GetUsers(t *testing.T) {
	room := CreateRoom("testRoom")
	defer room.Close()
	room.users = append(room.users, "tester")
	users := room.GetUsers()
	if len(users) != 1 {
		t.Fatal("users in room is not of size '1'")
	} else if users[0] != "tester" {
		t.Fatal("first element in users is not 'tester'")
	}
}

func TestChatRoom_AddUser(t *testing.T) {
	server = CreateServer()
	//
	// reset
	//
	defer func() {
		server = CreateServer()
	}()
	var b bytes.Buffer
	room := CreateRoom("testRoom")
	defer room.Close()
	server.AddUser(&ChatUser{
		Name:    "tester",
		writer:  &b,
		reader:  nil,
		buffer:  message.Buffer{},
		RWMutex: sync.RWMutex{},
	})
	room.AddUser("tester")
	if room.users[0] != "tester" {
		t.Fatal("room does not contain user 'tester'")
	}
}

func TestChatRoom_RemoveUser(t *testing.T) {
	server = CreateServer()
	//
	// reset
	//
	defer func() {
		server = CreateServer()
	}()
	var b bytes.Buffer
	room := CreateRoom("testRoom")
	defer room.Close()
	server.AddUser(&ChatUser{
		Name:    "tester",
		writer:  &b,
		reader:  nil,
		buffer:  message.Buffer{},
		RWMutex: sync.RWMutex{},
	})
	room.RemoveUser("tester")
	if len(room.users) > 0 {
		t.Fatal("room does contain users")
	}
}

func TestChatRoom_SendMessage(t *testing.T) {
	room := CreateRoom("testRoom")
	defer room.Close()
	room.SendMessage(message.ChatMessage{
		Timestamp: time.Now(),
		Room:      "testRoom",
		Sender:    "tester",
		Value:     "Hello from a test",
	})
	chatMessage := <-room.messageChannel
	if chatMessage.Value != "Hello from a test" {
		t.Fatal("message from room message channel does not contain the expected message")
	}
}

func TestChatRoom_HandleMessages_sameUser(t *testing.T) {
	server = CreateServer()
	//
	// reset
	//
	defer func() {
		server = CreateServer()
	}()
	var b bytes.Buffer
	server.AddUser(&ChatUser{
		Name:    "tester",
		writer:  &b,
		reader:  nil,
		buffer:  message.Buffer{},
		RWMutex: sync.RWMutex{},
	})
	room := CreateRoom("testRoom")
	room.AddUser("tester")
	room.SendMessage(message.ChatMessage{
		Timestamp: time.Now(),
		Room:      "testRoom",
		Sender:    "tester",
		Value:     "Hello from a test",
	})
	room.Close()
	room.HandleMessages()
	if len(room.messages) != 1 {
		t.Fatal("message was not added to the room's messages")
	}
}

func TestChatRoom_HandleMessages_differentUser(t *testing.T) {
	server = CreateServer()
	//
	// reset
	//
	defer func() {
		server = CreateServer()
	}()
	var b bytes.Buffer
	server.AddUser(&ChatUser{
		Name:    "tester",
		writer:  &b,
		reader:  nil,
		buffer:  message.Buffer{},
		RWMutex: sync.RWMutex{},
	})
	server.AddUser(&ChatUser{
		Name:    "tester1",
		writer:  &b,
		reader:  nil,
		buffer:  message.Buffer{},
		RWMutex: sync.RWMutex{},
	})
	room := CreateRoom("testRoom")
	room.AddUser("tester1")
	room.SendMessage(message.ChatMessage{
		Timestamp: time.Now(),
		Room:      "testRoom",
		Sender:    "tester",
		Value:     "Hello from a test",
	})
	room.Close()
	room.HandleMessages()
	if len(room.messages) != 1 {
		t.Fatal("message was not added to the room's messages")
	}
}

func TestChatRoom_Broadcast(t *testing.T) {
	server = CreateServer()
	//
	// reset
	//
	defer func() {
		server = CreateServer()
	}()
	var b bytes.Buffer
	server.AddUser(&ChatUser{
		Name:    "tester",
		writer:  &b,
		reader:  nil,
		buffer:  message.Buffer{},
		RWMutex: sync.RWMutex{},
	})
	room := CreateRoom("testRoom")
	room.Broadcast("Hello from the broadcast test")
}

func TestChatRoom_GetMessages_all(t *testing.T) {
	server = CreateServer()
	//
	// reset
	//
	defer func() {
		server = CreateServer()
	}()
	var b bytes.Buffer
	server.AddUser(&ChatUser{
		Name:    "tester",
		writer:  &b,
		reader:  nil,
		buffer:  message.Buffer{},
		RWMutex: sync.RWMutex{},
	})
	room := CreateRoom("testRoom")
	room.SendMessage(message.ChatMessage{
		Timestamp: time.Now(),
		Room:      "testRoom",
		Sender:    "tester",
		Value:     "Hello from a test",
	})
	room.Close()
	room.HandleMessages()
	if len(room.GetMessages(message.Query{})) == 0 {
		t.Fatal("failed to query any messages")
	}
}

func TestChatRoom_GetMessages_notBySender(t *testing.T) {
	server = CreateServer()
	//
	// reset
	//
	defer func() {
		server = CreateServer()
	}()
	var b bytes.Buffer
	server.AddUser(&ChatUser{
		Name:    "tester",
		writer:  &b,
		reader:  nil,
		buffer:  message.Buffer{},
		RWMutex: sync.RWMutex{},
	})
	room := CreateRoom("testRoom")
	room.SendMessage(message.ChatMessage{
		Timestamp: time.Now(),
		Room:      "testRoom",
		Sender:    "tester",
		Value:     "Hello from a test",
	})
	room.Close()
	room.HandleMessages()
	if len(room.GetMessages(message.Query{SenderName: "tester1"})) != 0 {
		t.Fatal("expected to not find any messages")
	}
}

func TestChatRoom_GetMessages_NotBeforeEnd(t *testing.T) {
	start := time.Now()
	server = CreateServer()
	//
	// reset
	//
	defer func() {
		server = CreateServer()
	}()
	var b bytes.Buffer
	server.AddUser(&ChatUser{
		Name:    "tester",
		writer:  &b,
		reader:  nil,
		buffer:  message.Buffer{},
		RWMutex: sync.RWMutex{},
	})
	room := CreateRoom("testRoom")
	room.SendMessage(message.ChatMessage{
		Timestamp: time.Now().Add(5 * time.Second),
		Room:      "testRoom",
		Sender:    "tester",
		Value:     "Hello from a test",
	})
	room.Close()
	room.HandleMessages()
	if len(room.GetMessages(message.Query{End: start})) != 0 {
		t.Fatal("expected to not find any messages")
	}
}

func TestChatRoom_GetMessages_notAfterStart(t *testing.T) {
	server = CreateServer()
	//
	// reset
	//
	defer func() {
		server = CreateServer()
	}()
	var b bytes.Buffer
	server.AddUser(&ChatUser{
		Name:    "tester",
		writer:  &b,
		reader:  nil,
		buffer:  message.Buffer{},
		RWMutex: sync.RWMutex{},
	})
	room := CreateRoom("testRoom")
	room.SendMessage(message.ChatMessage{
		Timestamp: time.Now(),
		Room:      "testRoom",
		Sender:    "tester",
		Value:     "Hello from a test",
	})
	room.Close()
	room.HandleMessages()
	if len(room.GetMessages(message.Query{Start: time.Now().Add(5 * time.Second)})) != 0 {
		t.Fatal("expected no messages to match")
	}
}
