package main

import (
	"sync"
)

const defaultRoom = "main"

// ChatServer is the server that is keeping rooms and users in-sync.
type ChatServer struct {
	rooms     map[string]*ChatRoom
	roomsLock sync.RWMutex
	users     map[string]*ChatUser
	usersLock sync.RWMutex
}

// CreateServer creates the server.
func CreateServer() ChatServer {
	return ChatServer{
		rooms:     make(map[string]*ChatRoom),
		roomsLock: sync.RWMutex{},
		users:     make(map[string]*ChatUser),
		usersLock: sync.RWMutex{},
	}
}

// CreateRoomIfMissing creates the room with the specified name if it does not exist. If the room exists, the room matching
// the name is returned.
func (server *ChatServer) CreateRoomIfMissing(roomName string) *ChatRoom {
	//
	// To ensure concurrency safety, use lock
	//
	server.roomsLock.Lock()
	//
	// Check if room exists - another goroutine could have created it
	//
	if server.rooms[roomName] == nil {
		r := CreateRoom(roomName)
		server.rooms[roomName] = &r
		//
		// Start the room's message handling
		//
		go r.HandleMessages()
		logger.Printf("Room %s has been created\n", roomName)
	}
	server.roomsLock.Unlock()
	return server.rooms[roomName]
}

// RemoveRoom removes the room from the server.
func (server *ChatServer) RemoveRoom(roomName string) {
	//
	// To ensure concurrency safety, lock writes to the chatRoom map
	//
	server.roomsLock.Lock()
	server.rooms[roomName].Close()
	delete(server.rooms, roomName)
	server.roomsLock.Unlock()
	logger.Printf("Room %s is empty. Room has been removed\n", roomName)
}

// GetRoom retrieves the room matching the specified room name.
func (server *ChatServer) GetRoom(roomName string) *ChatRoom {
	server.roomsLock.RLock()
	selectedRoom := server.rooms[roomName]
	server.roomsLock.RUnlock()
	selectedRoom = server.CreateRoomIfMissing(roomName)
	return selectedRoom
}

// ListRooms returns a list of all room names in the server.
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

// AddUser adds the user to the server.
func (server *ChatServer) AddUser(user *ChatUser) {
	//
	// Ensure concurrency safety
	//
	server.usersLock.Lock()
	server.users[user.Name] = user
	server.usersLock.Unlock()
}

// GetUser retrieves the user matching the specified user name.
func (server ChatServer) GetUser(userName string) *ChatUser {
	//
	// Ensure concurrency safety
	//
	server.usersLock.RLock()
	u := server.users[userName]
	server.usersLock.RUnlock()
	return u
}

// RemoveUser removes the user.
func (server *ChatServer) RemoveUser(userName string) {
	//
	// Ensure concurrency safety
	//
	server.usersLock.Lock()
	delete(server.users, userName)
	server.usersLock.Unlock()
	logger.Printf("%s has left the server\n", userName)
}

// UserExists checks if the user exists in the server.
func (server *ChatServer) UserExists(userName string) bool {
	exists := false
	//
	// Ensure concurrency safety
	//
	server.usersLock.RLock()
	if server.users[userName] != nil {
		exists = true
	}
	server.usersLock.RUnlock()
	return exists
}
