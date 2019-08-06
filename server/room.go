package main

import (
	"fmt"
	"github.com/piszmog/watercooler-chat/server/message"
	"sync"
)

// ChatRoom that represents a possible room for users to chat within.
type ChatRoom struct {
	Name           string
	userLock       sync.RWMutex
	users          []string
	messageChannel chan message.ChatMessage
	messageLock    sync.RWMutex
	messages       []message.ChatMessage
}

// CreateRoom creates a room with the provided name.
func CreateRoom(name string) ChatRoom {
	return ChatRoom{
		Name:           name,
		userLock:       sync.RWMutex{},
		messageChannel: make(chan message.ChatMessage, 100),
		messageLock:    sync.RWMutex{},
	}
}

// GetUsers retrieves the list of user names that are in the room.
func (room *ChatRoom) GetUsers() []string {
	room.userLock.RLock()
	users := room.users
	room.userLock.RUnlock()
	return users
}

// AddUser adds the user to the room.
func (room *ChatRoom) AddUser(userName string) {
	//
	// Update the room
	//
	room.userLock.Lock()
	room.users = append(room.users, userName)
	room.userLock.Unlock()
	logger.Printf("%s entered the room %s\n", userName, room.Name)
	//
	// Notify others that a new user has joined
	//
	room.Broadcast(fmt.Sprintf("%s has entered", userName))
}

// RemoveUser removes the user from the room.
func (room *ChatRoom) RemoveUser(userName string) {
	room.userLock.Lock()
	for index, name := range room.users {
		if userName == name {
			room.users = append(room.users[:index], room.users[index+1:]...)
			break
		}
	}
	room.userLock.Unlock()
	logger.Printf("%s left the room %s\n", userName, room.Name)
}

// SendMessage sends the message to the room message channel. Allows for message to be async sent to all users.
func (room *ChatRoom) SendMessage(message message.ChatMessage) {
	room.messageChannel <- message
}

// HandleMessage handles messages from the message channel. Messages received will be sent to users in the room.
func (room *ChatRoom) HandleMessages() {
	for chatMessage := range room.messageChannel {
		room.sendUserMessage(chatMessage)
	}
}

func (room *ChatRoom) sendUserMessage(message message.ChatMessage) {
	//
	// Add message to room history
	//
	room.messageLock.Lock()
	room.messages = append(room.messages, message)
	room.messageLock.Unlock()
	//
	// Format the logs with the chatRoom and ChatUser
	//
	logger.Println(message.LogMessage())
	//
	// Send message to all users in room
	//
	for _, name := range room.users {
		if message.Sender == name {
			continue
		} else {
			otherUser := server.GetUser(name)
			if otherUser.IsBlocked(message.Sender) {
				continue
			}
			//
			// Format the final message with the ChatUser and timestamp
			//
			otherUser.ReceiveMessage(message.RoomMessage())
		}
	}
}

// Broadcast sends a message to all users in the room, regardless of the user that caused the message to occur.
func (room *ChatRoom) Broadcast(message string) {
	room.userLock.RLock()
	users := room.users
	room.userLock.RUnlock()
	for _, name := range users {
		server.GetUser(name).ReceiveMessage(message)
	}
}

// Close closes the message channel.
func (room *ChatRoom) Close() {
	close(room.messageChannel)
}

// GetMessages retrieves messages based on the provided query.
func (room *ChatRoom) GetMessages(query message.Query) []message.ChatMessage {
	var matchingMessages []message.ChatMessage
	room.messageLock.RLock()
	messages := room.messages
	room.messageLock.RUnlock()
	for _, chatMessage := range messages {
		messageMatches := true
		//
		// If the sender name matches the message sender name, query matches
		//
		if len(query.SenderName) != 0 && query.SenderName != chatMessage.Sender {
			messageMatches = false
		}
		//
		// If message falls after the start date, it matches
		//
		if messageMatches && !query.Start.IsZero() && !chatMessage.Timestamp.After(query.Start) {
			messageMatches = false
		}
		//
		// If message falls before the end date, then it matches
		//
		if messageMatches && !query.End.IsZero() && !chatMessage.Timestamp.Before(query.End) {
			messageMatches = false
		}
		//
		// If message matches all criteria, add to slice
		//
		if messageMatches {
			matchingMessages = append(matchingMessages, chatMessage)
		}
	}
	if len(matchingMessages) == 0 {
		matchingMessages = make([]message.ChatMessage, 0)
	}
	return matchingMessages
}
