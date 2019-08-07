package main

import (
	"fmt"
	"github.com/piszmog/watercooler-chat/message"
	"github.com/reiver/go-oi"
	"github.com/reiver/go-telnet"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

const (
	messageWelcome          = "Welcome to room %s! You may begin chatting with the users."
	commandChangeRoom       = "-r"
	commandBlockUser        = "-b"
	commandUnblockUser      = "-u"
	commandListRooms        = "-lr"
	commandListUsersInRoom  = "-lu"
	commandListUsersBlocked = "-lb"
	commandQuit             = "-q"
	commandHelp             = "-h"
	commandHelpLong         = "-help"
	messageCommands         = "Available commands:\n" +
		commandChangeRoom + " ${room Name} -- change to the specified room. Creates room if doesn't exist\n" +
		commandBlockUser + " ${user Name} -- to block messages from the specified user\n" +
		commandUnblockUser + " ${user Name} -- to Unblock messages from the specified user\n" +
		commandListRooms + "             -- to list all existing rooms\n" +
		commandListUsersInRoom + "             -- to list all users in the current room\n" +
		commandListUsersBlocked + "             -- to list all users currently blocked\n" +
		commandQuit + "              -- to quit the chat\n" +
		commandHelp + "              -- to list all available commands\n"
	charNewLine        = '\n'
	charCarriageReturn = '\r'
)

// ChatUser is a user that can enter rooms and send messages to other users.
type ChatUser struct {
	Name   string
	writer telnet.Writer
	reader telnet.Reader
	buffer message.Buffer
	sync.RWMutex
	blockedUsers map[string]bool
}

// ServeTELNET is called when a new client connects over TELNET. When connected, the new client selects a user name and a
// a room to join.
func (user ChatUser) ServeTELNET(ctx telnet.Context, writer telnet.Writer, reader telnet.Reader) {
	//
	// Setup recover to handle any unexpected errors
	//
	defer func() {
		if r := recover(); r != nil {
			logger.Printf("error occurred in ServeTELNET: %+v: %s\n", reader, debug.Stack())
			// todo remove user from room/server
		}
	}()
	//
	// Update attributes on the user
	//
	user.writer = writer
	user.reader = reader
	user.buffer = message.Buffer{
		BufferBytes: make([]byte, 1, 1),
		Builder:     strings.Builder{},
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
	users := room.GetUsers()
	user.ReceiveMessage(fmt.Sprintf("Users currently in the room:\n%s", strings.Join(users, "\n")))
	//
	// Let user know of commands they can use
	//
	user.ReceiveMessage(messageCommands)
	//
	// Add user to room
	//
	room.AddUser(user.Name)
	user.ReceiveMessage(fmt.Sprintf(messageWelcome, room.Name))
	//
	// Start receiving messages from the user and send them to the others
	//
	user.handleMessages(room)
	//
	// handle when a user leaves the server
	//
	user.leave(room)
	server.RemoveUser(user.Name)
}

func (user *ChatUser) selectName() {
	//
	// loop until the user has chosen an acceptable name
	//
	for len(user.Name) == 0 {
		user.ReceiveMessage("What is your Name?")
		userName := user.getInput()
		if len(userName) == 0 { // Do not allow blank names
			user.ReceiveMessage("A Name is required.")
		} else if server.UserExists(userName) { // Do not allow a name already taken on the server
			user.ReceiveMessage(fmt.Sprintf("The Name %s already exists on the server. Choose a different Name.", userName))
		} else { // An acceptable name has been chosen
			user.Name = userName
		}
	}
	//
	// Update the server with the new user
	//
	server.AddUser(user) // todo multiple users with same name can get here if performed at the same exact time
}

func (user ChatUser) selectRoom() *ChatRoom {
	//
	// Get all current rooms on the sever
	//
	currentRooms := server.ListRooms()
	user.ReceiveMessage(fmt.Sprintf("Existing rooms:\n%s", strings.Join(currentRooms, "\n")))
	//
	// Get which room the user wants to go to/create
	//
	user.ReceiveMessage("What room would you like to enter (if room is not listed, room will be created)? ")
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
	return server.GetRoom(roomName)
}

// ReceiveMessage writes the provided message to the client.
func (user ChatUser) ReceiveMessage(message string) {
	//
	// Write the message to user
	//
	_, err := oi.LongWriteString(user.writer, message+"\n")
	if err != nil {
		//
		// Something terrible happened - log it
		//
		logger.Printf("ERROR: failed to send message %s to client %s: %+v\n", message, user.Name, err)
	}
}

func (user ChatUser) getInput() string {
	for {
		n, err := user.reader.Read(user.buffer.BufferBytes)
		if n > 0 {
			b := user.buffer.BufferBytes[0]
			if b == charNewLine {
				continue
			} else if b == charCarriageReturn {
				//
				// Break from loop
				//
				break
			} else {
				user.buffer.WriteByte(b)
			}
		}
		//
		// user disconnected
		//
		if err != nil {
			return ""
		}
	}
	return user.buffer.String()
}

func (user ChatUser) handleMessages(room *ChatRoom) {
	selectedRoom := room
	for {
		n, err := user.reader.Read(user.buffer.BufferBytes)
		if n > 0 {
			b := user.buffer.BufferBytes[0]
			if b == charNewLine {
				continue
			} else if b == charCarriageReturn {
				//
				// Send message to all other users
				//
				msg := user.buffer.String()
				//
				// Check if message is a command
				//
				if strings.HasPrefix(msg, commandChangeRoom) { // change rooms
					selectedRoom = user.changeRoom(selectedRoom, msg)
				} else if strings.HasPrefix(msg, commandBlockUser) { // block a user
					user.blockUser(msg)
				} else if strings.HasPrefix(msg, commandUnblockUser) { // unblock as user
					user.unblockUser(msg)
				} else if msg == commandListRooms { // list existing rooms
					user.ReceiveMessage(fmt.Sprintf("Existing rooms:\n%s", strings.Join(server.ListRooms(), "\n")))
				} else if msg == commandListUsersInRoom { // list users in the current room
					user.ReceiveMessage(fmt.Sprintf("Users currently in the room:\n%s\n", strings.Join(selectedRoom.GetUsers(), "\n")))
				} else if msg == commandListUsersBlocked { // list blocked users
					user.ReceiveMessage(strings.Join(user.getBlocked(), "\n"))
				} else if msg == commandQuit { // quit the server
					user.ReceiveMessage("Quiting...")
					break
				} else if msg == commandHelp || msg == commandHelpLong { // print commands
					user.ReceiveMessage(messageCommands)
				} else { // send message to other users in the room
					user.SendMessage(msg, selectedRoom)
				}
			} else {
				user.buffer.WriteByte(b)
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

// SendMessage sends the message from the user to the room.
func (user ChatUser) SendMessage(msg string, room *ChatRoom) {
	go room.SendMessage(message.ChatMessage{
		Timestamp: time.Now(),
		Room:      room.Name,
		Sender:    user.Name,
		Value:     msg,
	})
}

func (user ChatUser) changeRoom(previousRoom *ChatRoom, message string) *ChatRoom {
	user.leave(previousRoom)
	newRoomName := strings.Replace(message, commandChangeRoom+" ", "", 1)
	user.ReceiveMessage("Changed rooms...")
	newRoom := server.GetRoom(newRoomName)
	users := newRoom.GetUsers()
	user.ReceiveMessage(fmt.Sprintf("Users currently in the room:\n%s", strings.Join(users, "\n")))
	newRoom.AddUser(user.Name)
	return newRoom
}

func (user *ChatUser) blockUser(message string) {
	userName := strings.Replace(message, commandBlockUser+" ", "", 1)
	user.block(userName)
	user.ReceiveMessage(fmt.Sprintf("You have blocked %s", userName))
}

func (user *ChatUser) unblockUser(message string) {
	userName := strings.Replace(message, commandUnblockUser+" ", "", 1)
	user.unblock(userName)
	user.ReceiveMessage(fmt.Sprintf("You have unblocked %s", userName))
}

func (user ChatUser) leave(room *ChatRoom) {
	//
	// remove chatUser from chatRoom
	//
	room.RemoveUser(user.Name)
	//
	// if no one is in the room, remove the room
	//
	if len(room.GetUsers()) == 0 && room.Name != defaultRoom {
		server.RemoveRoom(room.Name)
	} else {
		//
		// Let other users know that this chatUser left
		//
		room.Broadcast(fmt.Sprintf("%s has left", user.Name))
	}
}

func (user *ChatUser) block(userName string) {
	user.Lock()
	user.blockedUsers[userName] = true
	user.Unlock()
}

func (user *ChatUser) unblock(userName string) {
	user.Lock()
	user.blockedUsers[userName] = false
	user.Unlock()
}

func (user *ChatUser) getBlocked() []string {
	user.RLock()
	users := user.blockedUsers
	user.RUnlock()
	var blockedUsers []string
	for name, isBlocked := range users {
		if isBlocked {
			blockedUsers = append(blockedUsers, name)
		}
	}
	return blockedUsers
}

// IsBlocked determines if the specified user is blocked the client.
func (user *ChatUser) IsBlocked(userName string) bool {
	var blocked bool
	user.RLock()
	blocked = user.blockedUsers[userName]
	user.RUnlock()
	return blocked
}
