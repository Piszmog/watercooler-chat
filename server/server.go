package main

import "sync"

type chatServer struct {
	rooms     map[string]*chatRoom
	roomsLock sync.RWMutex
	users     map[string]chatUser
	usersLock sync.RWMutex
}

func (server *chatServer) createRoomIfMissing(roomName string) *chatRoom {
	//
	// To ensure concurrency safety, use lock
	//
	server.roomsLock.Lock()
	//
	// Check if room exists - another goroutine could have created it
	//
	if server.rooms[roomName] == nil {
		server.rooms[roomName] = &chatRoom{
			name:  roomName,
			users: make(map[string]chatUser),
		}
		logger.Printf("Room %s has been created\n", roomName)
	}
	server.roomsLock.Unlock()
	return server.rooms[roomName]
}

func (server *chatServer) removeRoom(roomName string) {
	//
	// To ensure concurrency safety, lock writes to the chatRoom map
	//
	server.roomsLock.Lock()
	delete(server.rooms, roomName)
	server.roomsLock.Unlock()
	logger.Printf("Room %s is empty. Room has been removed\n", roomName)
}

func (server *chatServer) getRoom(roomName string) *chatRoom {
	server.roomsLock.RLock()
	selectedRoom := server.rooms[roomName]
	server.roomsLock.RUnlock()
	selectedRoom = server.createRoomIfMissing(roomName)
	return selectedRoom
}

func (server *chatServer) listRooms() []string {
	server.roomsLock.RLock()
	roomList := make([]string, len(server.rooms))
	index := 0
	for _, exitingRoom := range server.rooms {
		roomList[index] = exitingRoom.name
		index++
	}
	server.roomsLock.RUnlock()
	return roomList
}

func (server *chatServer) addUser(user chatUser) {
	//
	// Ensure concurrency safety
	//
	server.usersLock.Lock()
	server.users[user.name] = user
	server.usersLock.Unlock()
}

func (server *chatServer) getUser(userName string) chatUser {
	//
	// Ensure concurrency safety
	//
	server.usersLock.RLock()
	user := server.users[userName]
	server.usersLock.RUnlock()
	return user
}

func (server *chatServer) removeUser(user chatUser) {
	//
	// Ensure concurrency safety
	//
	server.usersLock.Lock()
	delete(server.users, user.name)
	server.usersLock.Unlock()
	logger.Printf("%s has left the server\n", user.name)
}

func (server *chatServer) userExists(userName string) bool {
	exists := false
	//
	// Ensure concurrency safety
	//
	server.usersLock.RLock()
	if len(server.users[userName].name) != 0 {
		exists = true
	}
	server.usersLock.RUnlock()
	return exists
}
