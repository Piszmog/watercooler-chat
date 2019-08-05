package main

import (
	"fmt"
	"sync"
	"time"
)

type chatRoom struct {
	name           string
	userLock       sync.RWMutex
	users          map[string]chatUser
	messageChannel chan chatMessage
	messageLock    sync.RWMutex
	messages       []chatMessage
}

type messageQuery struct {
	start      time.Time
	end        time.Time
	roomName   string
	senderName string
}

func (room *chatRoom) getUsers() []string {
	room.userLock.RLock()
	users := room.users
	room.userLock.RUnlock()
	userList := make([]string, len(users))
	index := 0
	for _, user := range users {
		userList[index] = user.name
		index++
	}
	return userList
}

func (room *chatRoom) addUser(user chatUser) {
	//
	// Update the room
	//
	room.userLock.Lock()
	room.users[user.name] = user
	room.userLock.Unlock()
	logger.Printf("%s entered the room %s\n", user.name, room.name)
	//
	// Notify others that a new user has joined
	//
	room.broadcast(fmt.Sprintf("%s has entered", user.name))
}

func (room *chatRoom) removeUser(currentUser chatUser) {
	room.userLock.Lock()
	delete(room.users, currentUser.name)
	users := room.users
	room.userLock.Unlock()
	for _, user := range users {
		user.unblock(currentUser.name)
	}
	logger.Printf("%s left the room %s\n", currentUser.name, room.name)
}

func (room *chatRoom) handleMessages() {
	for message := range room.messageChannel {
		room.sendMessage(message)
	}
}

func (room *chatRoom) sendMessage(message chatMessage) {
	//
	// Add message to room history
	//
	room.messageLock.Lock()
	room.messages = append(room.messages, message)
	room.messageLock.Unlock()
	//
	// Format the logs with the chatRoom and chatUser
	//
	logger.Println(message.logMessage())
	//
	// Send message to all users in room
	//
	for _, otherUser := range room.users {
		if message.Sender == otherUser.name {
			continue
		} else if otherUser.isBlocked(message.Sender) {
			continue
		}
		//
		// Format the final message with the chatUser and timestamp
		//
		otherUser.receiveMessage(message.roomMessage())
	}
}

func (room *chatRoom) broadcast(message string) {
	room.userLock.RLock()
	users := room.users
	room.userLock.RUnlock()
	for _, user := range users {
		user.receiveMessage(message)
	}
}

func (room *chatRoom) getMessages(query messageQuery) []chatMessage {
	var matchingMessages []chatMessage
	room.messageLock.RLock()
	messages := room.messages
	room.messageLock.RUnlock()
	for _, message := range messages {
		messageMatches := true
		if len(query.roomName) != 0 && query.roomName != message.Room {
			messageMatches = false
		}
		if messageMatches && len(query.senderName) != 0 && query.senderName != message.Sender {
			messageMatches = false
		}
		if messageMatches && !query.start.IsZero() && !message.Timestamp.After(query.start) {
			messageMatches = false
		}
		if messageMatches && !query.end.IsZero() && !message.Timestamp.Before(query.end) {
			messageMatches = false
		}
		if messageMatches {
			matchingMessages = append(matchingMessages, message)
		}
	}
	if len(matchingMessages) == 0 {
		matchingMessages = make([]chatMessage, 0)
	}
	return matchingMessages
}
