package main

import (
	"fmt"
	"github.com/reiver/go-oi"
	"github.com/reiver/go-telnet"
	"strings"
	"sync"
	"time"
)

const (
	messageWelcome  = "Welcome to room %s! You may begin chatting with the users.\n"
	messageCommands = "Available commands:\n" +
		"-r ${room name} -- change to the specified room. Creates room if doesn't exist\n" +
		"-b ${user name} -- to block messages from the specified user\n" +
		"-u ${user name} -- to unblock messages from the specified user\n" +
		"-lr             -- to list all existing rooms\n" +
		"-lu             -- to list all users in the current room\n" +
		"-lb             -- to list all users currently blocked\n" +
		"-h              -- to list all available commands\n\n"
	timestampFormat = "15:04 MST"
)

type chatUser struct {
	name   string
	writer telnet.Writer
	reader telnet.Reader
	buffer messageBuffer
	sync.RWMutex
	blockedUsers map[string]bool
}

func (user chatUser) ServeTELNET(ctx telnet.Context, w telnet.Writer, r telnet.Reader) {
	//
	// Setup recover to handle any unexpected errors
	//
	defer func() {
		if r := recover(); r != nil {
			logger.Println("error occurred in ServeTELNET:", r)
		}
	}()
	//
	// Update attributes on the chatUser
	//
	user.writer = w
	user.reader = r
	user.buffer = messageBuffer{
		bufferBytes: make([]byte, 1, 1),
		builder:     strings.Builder{},
	}
	user.blockedUsers = make(map[string]bool)
	//
	// Determine the chatUser's name
	//
	user.selectName()
	//
	// Let chatUser choose chatRoom they want to join
	//
	room := user.selectRoom()
	//
	// Let the chatUser know who else is in the chatRoom
	//
	users := room.listUsers()
	user.writeMessage(fmt.Sprintf("Users currently in the room:\n%s\n", strings.Join(users, "\n")))
	//
	// Let chatUser know of commands they can use
	//
	user.writeMessage(messageCommands)
	//
	// Let the other users know a new chatUser joins them
	//
	room.addUser(user)
	user.sendMessage(fmt.Sprintf("%s has entered", user.name), room)
	//
	// Start sending messages to the other users
	//
	user.writeMessage(fmt.Sprintf(messageWelcome, room.name))
	user.handleMessage(room)
	//
	// user has left the server
	//
	server.removeUser(user)
}

func (user *chatUser) selectName() {
	for len(user.name) == 0 {
		user.writeMessage("What is your name? ")
		userName := user.getInput()
		if len(userName) == 0 {
			user.writeMessage("A name is required.\n")
		} else if server.userExists(userName) {
			user.writeMessage(fmt.Sprintf("The name %s already exists on the server. Choose a different name.\n", userName))
		} else {
			user.name = userName
		}
	}
	server.addUser(*user)
}

func (user chatUser) selectRoom() *chatRoom {
	currentRooms := server.listRooms()
	user.writeMessage(fmt.Sprintf("Existing rooms:\n%s\n", strings.Join(currentRooms, "\n")))
	user.writeMessage("What room would you like to enter (if room is not listed, room will be created)? ")
	roomName := user.getInput()
	//
	// Get chatRoom, or create a new chatRoom
	//
	if len(roomName) == 0 {
		roomName = defaultRoom
	}
	selectedRoom := server.rooms[roomName]
	if selectedRoom == nil {
		selectedRoom = server.createRoom(roomName)
	}
	return selectedRoom
}

func (user chatUser) writeMessage(message string) {
	_, err := oi.LongWriteString(user.writer, message)
	if err != nil {
		//
		// Something terrible happened
		//
		logger.Printf("ERROR: failed to send message %s to client %s: %+v\n", message, user.name, err)
	}
}

func (user chatUser) getInput() string {
	for {
		n, err := user.reader.Read(user.buffer.bufferBytes)
		if n > 0 {
			b := user.buffer.bufferBytes[0]
			if b == '\n' {
				continue
			} else if b == '\r' {
				//
				// Break from loop
				//
				break
			} else {
				user.buffer.writeByte(b)
			}
		}
		//
		// chatUser disconnected
		//
		if err != nil {
			return ""
		}
	}
	return user.buffer.string()
}

func (user chatUser) handleMessage(room *chatRoom) {
	for {
		n, err := user.reader.Read(user.buffer.bufferBytes)
		if n > 0 {
			b := user.buffer.bufferBytes[0]
			if b == '\n' {
				continue
			} else if b == '\r' {
				//
				// Send message to all other users
				//
				message := user.buffer.string()
				//
				// Check if message is a command
				//
				if strings.HasPrefix(message, "-r") {
					// todo
				} else if strings.HasPrefix(message, "-b") {
					userName := strings.Replace(message, "-b ", "", 1)
					user.block(userName)
					user.writeMessage(fmt.Sprintf("You have blocked %s\n", userName))
				} else if strings.HasPrefix(message, "-u") {
					userName := strings.Replace(message, "-u ", "", 1)
					user.unblock(userName)
					user.writeMessage(fmt.Sprintf("You have unblocked %s\n", userName))
				} else if message == "-lr" {
					currentRooms := server.listRooms()
					user.writeMessage(fmt.Sprintf("Existing rooms:\n%s\n", strings.Join(currentRooms, "\n")))
				} else if message == "-lu" {
					users := room.listUsers()
					user.writeMessage(fmt.Sprintf("Users currently in the room:\n%s\n", strings.Join(users, "\n")))
				} else if message == "-lb" {
					//todo
				} else if message == "-h" || message == "-help" {
					user.writeMessage(messageCommands)
				} else {
					//
					// if not a command, send message to other users
					//
					user.sendMessage(message, room)
				}
			} else {
				user.buffer.writeByte(b)
			}
		}
		//
		// handle error case - chatUser left for some reason
		//
		if err != nil {
			break
		}
	}
	user.leave(room)
}

func (user chatUser) leave(room *chatRoom) {
	//
	// remove chatUser from chatRoom
	//
	room.removeUser(user)
	//
	// Let other users know that this chatUser left
	//
	user.sendMessage(fmt.Sprintf("%s has left", user.name), room)
}

func (user chatUser) sendMessage(message string, room *chatRoom) {
	//
	// Format the logs with the chatRoom and chatUser
	//
	logger.Println(fmt.Sprintf("chat message - [%s %s] %s", user.name, room.name, message))
	timestamp := time.Now().Format(timestampFormat)
	for _, otherUser := range room.users {
		if user.name == otherUser.name {
			continue
		} else if otherUser.isBlocked(user.name) {
			continue
		}
		//
		// Format the final message with the chatUser and timestamp
		//
		otherUser.writeMessage(fmt.Sprintf("[%s %s]: %s\n", user.name, timestamp, message))
	}
}

func (user *chatUser) block(userName string) {
	user.Lock()
	user.blockedUsers[userName] = true
	user.Unlock()
}

func (user *chatUser) unblock(userName string) {
	user.Lock()
	user.blockedUsers[userName] = false
	user.Unlock()
}

func (user *chatUser) isBlocked(userName string) bool {
	var blocked bool
	user.RLock()
	blocked = user.blockedUsers[userName]
	user.RUnlock()
	return blocked
}
