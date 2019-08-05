package room

import (
	"fmt"
	"github.com/piszmog/watercooler-chat/server/user"
	"sync"
	"time"
)

type ChatRoom struct {
	Name           string
	userLock       sync.RWMutex
	users          map[string]user.ChatUser
	messageChannel chan ChatMessage
	messageLock    sync.RWMutex
	messages       []ChatMessage
}

type MessageQuery struct {
	Start      time.Time
	End        time.Time
	RoomName   string
	SenderName string
}

func CreateRoom(name string) ChatRoom {
	return ChatRoom{
		Name:           name,
		userLock:       sync.RWMutex{},
		users:          make(map[string]user.ChatUser),
		messageChannel: make(chan ChatMessage, 100),
		messageLock:    sync.RWMutex{},
	}
}

func (room *ChatRoom) GetUsers() []string {
	room.userLock.RLock()
	users := room.users
	room.userLock.RUnlock()
	userList := make([]string, len(users))
	index := 0
	for _, u := range users {
		userList[index] = u.Name
		index++
	}
	return userList
}

func (room *ChatRoom) AddUser(u user.ChatUser) {
	//
	// Update the room
	//
	room.userLock.Lock()
	room.users[u.Name] = u
	room.userLock.Unlock()
	//main.logger.Printf("%s entered the room %s\n", u.Name, room.name)
	//
	// Notify others that a new user has joined
	//
	room.Broadcast(fmt.Sprintf("%s has entered", u.Name))
}

func (room *ChatRoom) RemoveUser(currentUser user.ChatUser) {
	room.userLock.Lock()
	delete(room.users, currentUser.Name)
	users := room.users
	room.userLock.Unlock()
	for _, u := range users {
		u.Unblock(currentUser.Name)
	}
	//main.logger.Printf("%s left the room %s\n", currentUser.Name, room.name)
}

func (room *ChatRoom) SendMessage(message ChatMessage) {
	room.messageChannel <- message
}

func (room *ChatRoom) HandleMessages() {
	for message := range room.messageChannel {
		room.sendUserMessage(message)
	}
}

func (room *ChatRoom) sendUserMessage(message ChatMessage) {
	//
	// Add message to room history
	//
	room.messageLock.Lock()
	room.messages = append(room.messages, message)
	room.messageLock.Unlock()
	//
	// Format the logs with the chatRoom and ChatUser
	//
	//main.logger.Println(message.logMessage())
	//
	// Send message to all users in room
	//
	for _, otherUser := range room.users {
		if message.Sender == otherUser.Name {
			continue
		} else if otherUser.IsBlocked(message.Sender) {
			continue
		}
		//
		// Format the final message with the ChatUser and timestamp
		//
		otherUser.ReceiveMessage(message.roomMessage())
	}
}

func (room *ChatRoom) Broadcast(message string) {
	room.userLock.RLock()
	users := room.users
	room.userLock.RUnlock()
	for _, u := range users {
		u.ReceiveMessage(message)
	}
}

func (room *ChatRoom) Close() {
	close(room.messageChannel)
}

func (room *ChatRoom) GetMessages(query MessageQuery) []ChatMessage {
	var matchingMessages []ChatMessage
	room.messageLock.RLock()
	messages := room.messages
	room.messageLock.RUnlock()
	for _, message := range messages {
		messageMatches := true
		if len(query.RoomName) != 0 && query.RoomName != message.Room {
			messageMatches = false
		}
		if messageMatches && len(query.SenderName) != 0 && query.SenderName != message.Sender {
			messageMatches = false
		}
		if messageMatches && !query.Start.IsZero() && !message.Timestamp.After(query.Start) {
			messageMatches = false
		}
		if messageMatches && !query.End.IsZero() && !message.Timestamp.Before(query.End) {
			messageMatches = false
		}
		if messageMatches {
			matchingMessages = append(matchingMessages, message)
		}
	}
	if len(matchingMessages) == 0 {
		matchingMessages = make([]ChatMessage, 0)
	}
	return matchingMessages
}
