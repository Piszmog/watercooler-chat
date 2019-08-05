package server

import (
	"github.com/piszmog/watercooler-chat/server/room"
	"github.com/piszmog/watercooler-chat/server/user"
	"sync"
)

const DefaultRoom = "main"

type ChatServer struct {
	rooms     map[string]*room.ChatRoom
	roomsLock sync.RWMutex
	users     map[string]user.ChatUser
	usersLock sync.RWMutex
}

func CreateServer() ChatServer {
	return ChatServer{
		rooms:     make(map[string]*room.ChatRoom),
		roomsLock: sync.RWMutex{},
		users:     make(map[string]user.ChatUser),
		usersLock: sync.RWMutex{},
	}
}

func (server *ChatServer) CreateRoomIfMissing(roomName string) *room.ChatRoom {
	//
	// To ensure concurrency safety, use lock
	//
	server.roomsLock.Lock()
	//
	// Check if room exists - another goroutine could have created it
	//
	if server.rooms[roomName] == nil {
		r := room.CreateRoom(roomName)
		server.rooms[roomName] = &r
		//
		// Start the room's message handling
		//
		go r.HandleMessages()
		//main.logger.Printf("Room %s has been created\n", roomName)
	}
	server.roomsLock.Unlock()
	return server.rooms[roomName]
}

func (server *ChatServer) RemoveRoom(roomName string) {
	//
	// To ensure concurrency safety, lock writes to the chatRoom map
	//
	server.roomsLock.Lock()
	server.rooms[roomName].Close()
	delete(server.rooms, roomName)
	server.roomsLock.Unlock()
	//main.logger.Printf("Room %s is empty. Room has been removed\n", roomName)
}

func (server *ChatServer) GetRoom(roomName string) *room.ChatRoom {
	server.roomsLock.RLock()
	selectedRoom := server.rooms[roomName]
	server.roomsLock.RUnlock()
	selectedRoom = server.CreateRoomIfMissing(roomName)
	return selectedRoom
}

func (server *ChatServer) ListRooms() []string {
	server.roomsLock.RLock()
	rooms := server.rooms
	server.roomsLock.RUnlock()
	roomList := make([]string, len(server.rooms))
	index := 0
	for _, exitingRoom := range rooms {
		roomList[index] = exitingRoom.Name
		index++
	}
	return roomList
}

func (server *ChatServer) AddUser(user user.ChatUser) {
	//
	// Ensure concurrency safety
	//
	server.usersLock.Lock()
	server.users[user.Name] = user
	server.usersLock.Unlock()
}

func (server *ChatServer) getUser(userName string) user.ChatUser {
	//
	// Ensure concurrency safety
	//
	server.usersLock.RLock()
	u := server.users[userName]
	server.usersLock.RUnlock()
	return u
}

func (server *ChatServer) RemoveUser(user user.ChatUser) {
	//
	// Ensure concurrency safety
	//
	server.usersLock.Lock()
	delete(server.users, user.Name)
	server.usersLock.Unlock()
	//main.logger.Printf("%s has left the server\n", user.Name)
}

func (server *ChatServer) UserExists(userName string) bool {
	exists := false
	//
	// Ensure concurrency safety
	//
	server.usersLock.RLock()
	if len(server.users[userName].Name) != 0 {
		exists = true
	}
	server.usersLock.RUnlock()
	return exists
}
