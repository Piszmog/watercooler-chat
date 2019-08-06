package main

import (
	"log"
	"os"
	"testing"
)

func init() {
	logger = log.New(os.Stdout, "TEST: ", log.LstdFlags|log.LUTC)
}

func TestChatServer_CreateRoomIfMissing(t *testing.T) {
	server := CreateServer()
	server.CreateRoomIfMissing("testRoom")
	if server.rooms["testRoom"] == nil {
		t.Fatal("server does not contain the room 'testRoom'")
	}
}

func TestChatServer_RemoveRoom(t *testing.T) {
	server := CreateServer()
	server.CreateRoomIfMissing("testRoom")
	server.RemoveRoom("testRoom")
	if server.rooms["testRoom"] != nil {
		t.Fatal("server does contain the room 'testRoom' after being removed from the server")
	}
}

func TestChatServer_GetRoom(t *testing.T) {
	server := CreateServer()
	server.CreateRoomIfMissing("testRoom")
	room := server.GetRoom("testRoom")
	if room == nil {
		t.Fatal("server does not contain the room 'testRoom'")
	}
}

func TestChatServer_ListRooms(t *testing.T) {
	server := CreateServer()
	server.CreateRoomIfMissing("testRoom")
	rooms := server.ListRooms()
	if len(rooms) != 1 {
		t.Fatal("List of rooms does not equal expected size '1'")
	} else if rooms[0] != "testRoom" {
		t.Fatal("first element in the list of rooms is not equal to 'testRoom'")
	}
}

func TestChatServer_AddUser(t *testing.T) {
	server := CreateServer()
	server.AddUser(&ChatUser{Name: "tester"})
	if server.users["tester"] == nil {
		t.Fatal("users 'tester' not found in the server")
	}
}

func TestChatServer_GetUser(t *testing.T) {
	server := CreateServer()
	server.AddUser(&ChatUser{Name: "tester"})
	user := server.GetUser("tester")
	if user == nil {
		t.Fatal("user 'tester' is not in the server")
	}
}

func TestChatServer_RemoveUser(t *testing.T) {
	server := CreateServer()
	server.AddUser(&ChatUser{Name: "tester"})
	server.RemoveUser("tester")
	if server.users["tester"] != nil {
		t.Fatal("user 'tester' is still in the server")
	}
}

func TestChatServer_UserExists(t *testing.T) {
	server := CreateServer()
	server.AddUser(&ChatUser{Name: "tester"})
	if !server.UserExists("tester") {
		t.Fatal("user 'tester' does not exist in the server")
	}
}
