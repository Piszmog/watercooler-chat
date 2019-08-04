package main

import (
	"fmt"
	"github.com/reiver/go-oi"
	"github.com/reiver/go-telnet"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

const (
	messageWelcome  = "Welcome to room %s! You may begin chatting with the users."
	messageCommands = "Available commands:\n" +
		"-r ${room name} -- change to the specified room. Creates room if doesn't exist\n" +
		"-b ${user name} -- to block messages from the specified user\n" +
		"-u ${user name} -- to unblock messages from the specified user\n" +
		"-lr             -- to list all existing rooms\n" +
		"-lu             -- to list all users in the current room\n" +
		"-lb             -- to list all users currently blocked\n" +
		"-q              -- to quit the chat\n" +
		"-h              -- to list all available commands\n"
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
			logger.Printf("error occurred in ServeTELNET: %+v: %s\n", r, debug.Stack())
			// todo remove user from room/server
		}
	}()
	//
	// Update attributes on the user
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
	// Let user choose room they want to join
	//
	room := user.selectRoom()
	//
	// Let the user know who else is in the chatRoom
	//
	users := room.getUsers()
	user.receiveMessage(fmt.Sprintf("Users currently in the room:\n%s", strings.Join(users, "\n")))
	//
	// Let user know of commands they can use
	//
	user.receiveMessage(messageCommands)
	//
	// Add user to room
	//
	room.addUser(user)
	user.receiveMessage(fmt.Sprintf(messageWelcome, room.name))
	//
	// Start receiving messages from the user and send them to the others
	//
	user.handleMessages(room)
	//
	// handle when a user leaves the server
	//
	user.leave(room)
	server.removeUser(user)
}

func (user *chatUser) selectName() {
	//
	// loop until the user has chosen an acceptable name
	//
	for len(user.name) == 0 {
		user.receiveMessage("What is your name?")
		userName := user.getInput()
		if len(userName) == 0 { // Do not allow blank names
			user.receiveMessage("A name is required.")
		} else if server.userExists(userName) { // Do not allow a name already taken on the server
			user.receiveMessage(fmt.Sprintf("The name %s already exists on the server. Choose a different name.", userName))
		} else { // An acceptable name has been chosen
			user.name = userName
		}
	}
	//
	// Update the server with the new user
	//
	server.addUser(*user) // todo multiple users with same name can get here if performed at the same exact time
}

func (user chatUser) selectRoom() *chatRoom {
	//
	// Get all current rooms on the sever
	//
	currentRooms := server.listRooms()
	user.receiveMessage(fmt.Sprintf("Existing rooms:\n%s", strings.Join(currentRooms, "\n")))
	//
	// Get which room the user wants to go to/create
	//
	user.receiveMessage("What room would you like to enter (if room is not listed, room will be created)? ")
	roomName := user.getInput()
	//
	// Get chatRoom, or create a new chatRoom
	//
	if len(roomName) == 0 {
		roomName = defaultRoom
	}
	//
	// Retrieve the room
	//
	return server.getRoom(roomName)
}

func (user chatUser) receiveMessage(message string) {
	//
	// Write the message to user
	//
	_, err := oi.LongWriteString(user.writer, message+"\n")
	if err != nil {
		//
		// Something terrible happened - log it
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
		// user disconnected
		//
		if err != nil {
			return ""
		}
	}
	return user.buffer.string()
}

func (user chatUser) handleMessages(room *chatRoom) {
	selectedRoom := room
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
				if strings.HasPrefix(message, "-r") { // change rooms
					selectedRoom = user.changeRoom(selectedRoom, message)
				} else if strings.HasPrefix(message, "-b") { // block a user
					user.blockUser(message)
				} else if strings.HasPrefix(message, "-u") { // unblock as user
					user.unblockUser(message)
				} else if message == "-lr" { // list existing rooms
					user.receiveMessage(fmt.Sprintf("Existing rooms:\n%s", strings.Join(server.listRooms(), "\n")))
				} else if message == "-lu" { // list users in the current room
					user.receiveMessage(fmt.Sprintf("Users currently in the room:\n%s\n", strings.Join(selectedRoom.getUsers(), "\n")))
				} else if message == "-lb" { // list blocked users
					user.receiveMessage(strings.Join(user.getBlocked(), "\n"))
				} else if message == "-q" { // quit the server
					user.receiveMessage("Quiting...")
					break
				} else if message == "-h" || message == "-help" { // print commands
					user.receiveMessage(messageCommands)
				} else { // send message to other users in the room
					user.sendMessage(message, selectedRoom)
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
}

func (user *chatUser) unblockUser(message string) {
	userName := strings.Replace(message, "-u ", "", 1)
	user.unblock(userName)
	user.receiveMessage(fmt.Sprintf("You have unblocked %s", userName))
}

func (user *chatUser) blockUser(message string) {
	userName := strings.Replace(message, "-b ", "", 1)
	user.block(userName)
	user.receiveMessage(fmt.Sprintf("You have blocked %s", userName))
}

func (user chatUser) changeRoom(selectedRoom *chatRoom, message string) *chatRoom {
	user.leave(selectedRoom)
	newRoom := strings.Replace(message, "-r ", "", 1)
	user.receiveMessage("Changed rooms...")
	selectedRoom = server.getRoom(newRoom)
	users := selectedRoom.getUsers()
	user.receiveMessage(fmt.Sprintf("Users currently in the room:\n%s", strings.Join(users, "\n")))
	selectedRoom.addUser(user)
	return selectedRoom
}

func (user chatUser) leave(room *chatRoom) {
	//
	// remove chatUser from chatRoom
	//
	room.removeUser(user)
	//
	// if no one is in the room, remove the room
	//
	if len(room.getUsers()) == 0 && room.name != defaultRoom {
		server.removeRoom(room.name)
	} else {
		//
		// Let other users know that this chatUser left
		//
		room.broadcast(fmt.Sprintf("%s has left", user.name))
	}
}

func (user chatUser) sendMessage(message string, room *chatRoom) {
	//
	// Format the logs with the chatRoom and chatUser
	//
	logger.Println(fmt.Sprintf("chat message - [%s %s] %s", room.name, user.name, message))
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
		otherUser.receiveMessage(fmt.Sprintf("[%s %s]: %s", timestamp, user.name, message))
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

func (user *chatUser) getBlocked() []string {
	user.RLock()
	var blockedUsers []string
	for name, isBlocked := range user.blockedUsers {
		if isBlocked {
			blockedUsers = append(blockedUsers, name)
		}
	}
	user.RUnlock()
	return blockedUsers
}

func (user *chatUser) isBlocked(userName string) bool {
	var blocked bool
	user.RLock()
	blocked = user.blockedUsers[userName]
	user.RUnlock()
	return blocked
}
