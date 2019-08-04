package main

import (
	"fmt"
	"sync"
)

type chatRoom struct {
	name string
	sync.RWMutex
	users map[string]chatUser
}

func (room *chatRoom) getUsers() []string {
	room.RLock()
	//
	// Get the current user names
	//
	users := room.users
	userList := make([]string, len(users))
	index := 0
	for _, user := range users {
		userList[index] = user.name
		index++
	}
	room.RUnlock()
	return userList
}

func (room *chatRoom) addUser(user chatUser) {
	//
	// Update the room
	//
	room.Lock()
	room.users[user.name] = user
	room.Unlock()
	logger.Printf("%s entered the room %s\n", user.name, room.name)
	//
	// Notify others that a new user has joined
	//
	room.broadcast(fmt.Sprintf("%s has entered", user.name))
}

func (room *chatRoom) removeUser(currentUser chatUser) {
	room.Lock()
	delete(room.users, currentUser.name)
	for _, user := range room.users {
		user.unblock(currentUser.name)
	}
	room.Unlock()
	logger.Printf("%s left the room %s\n", currentUser.name, room.name)
}

func (room *chatRoom) broadcast(message string) {
	room.RLock()
	for _, user := range room.users {
		user.receiveMessage(message)
	}
	room.RUnlock()
}
