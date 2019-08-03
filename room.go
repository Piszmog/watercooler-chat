package main

import (
	"sync"
)

var emptyList = []string{"None"}

type chatRoom struct {
	name string
	sync.RWMutex
	users map[string]chatUser
}

func (room *chatRoom) listUsers() []string {
	room.RLock()
	users := room.users
	userList := make([]string, len(users))
	index := 0
	if len(users) == 0 {
		userList = emptyList
	} else {
		for _, user := range users {
			userList[index] = user.name
			index++
		}
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
	logger.Printf("User %s left the room %s\n", currentUser.name, room.name)
}
