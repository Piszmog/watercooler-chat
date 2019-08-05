package user

import (
	"fmt"
	"github.com/piszmog/watercooler-chat/server/room"
	"github.com/piszmog/watercooler-chat/server/server"
	"github.com/reiver/go-oi"
	"github.com/reiver/go-telnet"
	"strings"
	"sync"
	"time"
)

const (
	messageWelcome  = "Welcome to room %s! You may begin chatting with the users."
	messageCommands = "Available commands:\n" +
		"-r ${room Name} -- change to the specified room. Creates room if doesn't exist\n" +
		"-b ${user Name} -- to block messages from the specified user\n" +
		"-u ${user Name} -- to Unblock messages from the specified user\n" +
		"-lr             -- to list all existing rooms\n" +
		"-lu             -- to list all users in the current room\n" +
		"-lb             -- to list all users currently blocked\n" +
		"-q              -- to quit the chat\n" +
		"-h              -- to list all available commands\n"
)

type ChatUser struct {
	Name   string
	writer telnet.Writer
	reader telnet.Reader
	buffer messageBuffer
	sync.RWMutex
	blockedUsers map[string]bool
}

func (user ChatUser) ServeTELNET(ctx telnet.Context, writer telnet.Writer, reader telnet.Reader) {
	//
	// Setup recover to handle any unexpected errors
	//
	defer func() {
		if r := recover(); r != nil {
			//main.logger.Printf("error occurred in ServeTELNET: %+v: %s\n", reader, debug.Stack())
			// todo remove user from room/server
		}
	}()
	//
	// Update attributes on the user
	//
	user.writer = writer
	user.reader = reader
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
	r := user.selectRoom()
	//
	// Let the user know who else is in the chatRoom
	//
	users := r.GetUsers()
	user.ReceiveMessage(fmt.Sprintf("Users currently in the r:\n%s", strings.Join(users, "\n")))
	//
	// Let user know of commands they can use
	//
	user.ReceiveMessage(messageCommands)
	//
	// Add user to room
	//
	r.AddUser(user)
	user.ReceiveMessage(fmt.Sprintf(messageWelcome, r.Name))
	//
	// Start receiving messages from the user and send them to the others
	//
	user.handleMessages(r)
	//
	// handle when a user leaves the server
	//
	user.leave(r)
	user.Server.RemoveUser(user)
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
		} else if user.Server.UserExists(userName) { // Do not allow a name already taken on the server
			user.ReceiveMessage(fmt.Sprintf("The Name %s already exists on the server. Choose a different Name.", userName))
		} else { // An acceptable name has been chosen
			user.Name = userName
		}
	}
	//
	// Update the server with the new user
	//
	user.Server.AddUser(*user) // todo multiple users with same name can get here if performed at the same exact time
}

func (user ChatUser) selectRoom() *room.ChatRoom {
	//
	// Get all current rooms on the sever
	//
	currentRooms := user.Server.ListRooms()
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
		roomName = server.DefaultRoom
	}
	//
	// Retrieve the room
	//
	return user.Server.GetRoom(roomName)
}

func (user ChatUser) ReceiveMessage(message string) {
	//
	// Write the message to user
	//
	_, err := oi.LongWriteString(user.writer, message+"\n")
	if err != nil {
		//
		// Something terrible happened - log it
		//
		//main.logger.Printf("ERROR: failed to send message %s to client %s: %+v\n", message, user.Name, err)
	}
}

func (user ChatUser) getInput() string {
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

func (user ChatUser) handleMessages(room *room.ChatRoom) {
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
				msg := user.buffer.string()
				//
				// Check if message is a command
				//
				if strings.HasPrefix(msg, "-r") { // change rooms
					selectedRoom = user.changeRoom(selectedRoom, msg)
				} else if strings.HasPrefix(msg, "-b") { // block a user
					user.blockUser(msg)
				} else if strings.HasPrefix(msg, "-u") { // unblock as user
					user.unblockUser(msg)
				} else if msg == "-lr" { // list existing rooms
					user.ReceiveMessage(fmt.Sprintf("Existing rooms:\n%s", strings.Join(user.Server.ListRooms(), "\n")))
				} else if msg == "-lu" { // list users in the current room
					user.ReceiveMessage(fmt.Sprintf("Users currently in the room:\n%s\n", strings.Join(selectedRoom.GetUsers(), "\n")))
				} else if msg == "-lb" { // list blocked users
					user.ReceiveMessage(strings.Join(user.getBlocked(), "\n"))
				} else if msg == "-q" { // quit the server
					user.ReceiveMessage("Quiting...")
					break
				} else if msg == "-h" || msg == "-help" { // print commands
					user.ReceiveMessage(messageCommands)
				} else { // send message to other users in the room
					user.SendMessage(msg, room)
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

func (user ChatUser) SendMessage(msg string, r *room.ChatRoom) {
	r.SendMessage(room.ChatMessage{
		Timestamp: time.Now(),
		Room:      r.Name,
		Sender:    user.Name,
		Value:     msg,
	})
}

func (user *ChatUser) unblockUser(message string) {
	userName := strings.Replace(message, "-u ", "", 1)
	user.Unblock(userName)
	user.ReceiveMessage(fmt.Sprintf("You have unblocked %s", userName))
}

func (user *ChatUser) blockUser(message string) {
	userName := strings.Replace(message, "-b ", "", 1)
	user.block(userName)
	user.ReceiveMessage(fmt.Sprintf("You have blocked %s", userName))
}

func (user ChatUser) changeRoom(selectedRoom *room.ChatRoom, message string) *room.ChatRoom {
	user.leave(selectedRoom)
	newRoom := strings.Replace(message, "-r ", "", 1)
	user.ReceiveMessage("Changed rooms...")
	selectedRoom = user.Server.GetRoom(newRoom)
	users := selectedRoom.GetUsers()
	user.ReceiveMessage(fmt.Sprintf("Users currently in the room:\n%s", strings.Join(users, "\n")))
	selectedRoom.AddUser(user)
	return selectedRoom
}

func (user ChatUser) leave(room *room.ChatRoom) {
	//
	// remove chatUser from chatRoom
	//
	room.RemoveUser(user)
	//
	// if no one is in the room, remove the room
	//
	if len(room.GetUsers()) == 0 && room.Name != server.DefaultRoom {
		user.Server.RemoveRoom(room.Name)
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

func (user *ChatUser) Unblock(userName string) {
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

func (user *ChatUser) IsBlocked(userName string) bool {
	var blocked bool
	user.RLock()
	blocked = user.blockedUsers[userName]
	user.RUnlock()
	return blocked
}
