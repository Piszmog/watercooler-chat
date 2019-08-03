package main

import (
	"sync"
)

type chatRoom struct {
	name string
	sync.RWMutex
	users map[string]chatUser
}

func (room *chatRoom) getUsers() []string {
	room.RLock()
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

func (room *chatRoom) addUser(currentUser chatUser) {
	room.Lock()
	room.users[currentUser.name] = currentUser
	room.Unlock()
	logger.Printf("%s entered the room %s\n", currentUser.name, room.name)
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
		user.sendMessage(message, room)
	}
	room.RUnlock()
}
