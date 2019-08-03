package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/pkg/errors"
	"github.com/reiver/go-oi"
	"github.com/reiver/go-telnet"
	"io"
	"log"
	"os"
	"path"
	"strings"
	"sync"
	"time"
)

const (
	defaultPort        = "5555"
	defaultLogLocation = "log.txt"
	defaultRoom        = "main"
)

var logger *log.Logger
var roomsMutex = sync.Mutex{}
var rooms = make(map[string]room)

type configuration struct {
	IPAddress   string `json:"ipAddress"`
	Port        string `json:"port"`
	LogLocation string `json:"logFileLocation"`
}

type room struct {
	name  string
	mutex sync.Mutex
	users map[string]user
}

type user struct {
	name   string
	writer telnet.Writer
}

func (currentUser user) ServeTELNET(ctx telnet.Context, w telnet.Writer, r telnet.Reader) {
	//
	// Setup recover to handle any unexpected errors
	//
	defer func() {
		if r := recover(); r != nil {
			logger.Println("error occurred in ServeTELNET:", r)
		}
	}()
	//
	// Update attributes on the user
	//
	currentUser.writer = w
	//
	// Prepare buffer and message builder
	//
	var buffer [1]byte
	p := buffer[:]
	messageBuilder := strings.Builder{}
	//
	// Let user choose room they want to join
	//
	selectedRoom := selectRoom(currentUser, r, p, messageBuilder)
	//
	// Determine the user's name
	//
	selectName(&currentUser, r, p, messageBuilder, selectedRoom)
	//
	// Let user know of commands they can use
	//
	writeMessage("Available commands:\n"+
		"-r ${room name} -- change to the specified room. Creates room if doesn't exist\n"+
		"-b ${user name} -- to block messages from the specified user\n"+
		"-u ${user name} -- to unblock messages from the specified user\n"+
		"-lr             -- to list all existing rooms\n"+
		"-lu             -- to list all users in the room\n"+
		"-lb             -- to list all users currently blocked\n"+
		"-h              -- to list all available commands\n\n", currentUser)
	//
	// Let the user know who else is in the room
	//
	listUsersInRoom(selectedRoom, currentUser)
	//
	// Let the other users know a new user joins them
	//
	sendMessageToOtherUsers(fmt.Sprintf("%s has entered", currentUser.name), selectedRoom.name, currentUser, selectedRoom.users)
	addUser(currentUser, selectedRoom)
	//
	// Start sending messages to the other users
	//
	handleUserMessages(r, p, messageBuilder, selectedRoom, currentUser, selectedRoom.users)
}

func listUsersInRoom(selectedRoom room, currentUser user) {
	users := selectedRoom.users
	userList := make([]string, len(users))
	index := 0
	for _, user := range users {
		userList[index] = user.name
		index++
	}
	userNames := "None"
	if len(userList) != 0 {
		userNames = strings.Join(userList, "\n")
	}
	writeMessage(fmt.Sprintf("Users currently in the room:\n%s\n", userNames), currentUser)
}

func selectName(currentUser *user, r telnet.Reader, p []byte, messageBuilder strings.Builder, selectedRoom room) {
	for len(currentUser.name) == 0 {
		writeMessage("What is your name? ", *currentUser)
		getUserInput(r, p, &messageBuilder)
		userName := messageBuilder.String()
		messageBuilder.Reset()
		if len(userName) == 0 {
			writeMessage("A user name is required.\n", *currentUser)
		} else if len(selectedRoom.users[userName].name) != 0 {
			writeMessage(fmt.Sprintf("The user name %s is already taken. Choose a different name.", userName), *currentUser)
		} else {
			currentUser.name = userName
		}
	}
}

func selectRoom(currentUser user, reader telnet.Reader, bytes []byte, messageBuilder strings.Builder) room {
	listExistingRooms(currentUser)
	writeMessage("What room would you like to enter (if room is not listed, room will be created)? ", currentUser)
	if getUserInput(reader, bytes, &messageBuilder) {
		return room{}
	}
	roomName := messageBuilder.String()
	messageBuilder.Reset()
	//
	// Get room, or create a new room
	//
	if len(roomName) == 0 {
		roomName = defaultRoom
	}
	selectedRoom := rooms[roomName]
	if len(selectedRoom.name) == 0 {
		selectedRoom = createRoom(roomName)
	}
	return selectedRoom
}

func listExistingRooms(currentUser user) {
	roomList := make([]string, len(rooms))
	index := 0
	for _, exitingRoom := range rooms {
		roomList[index] = exitingRoom.name
		index++
	}
	writeMessage(fmt.Sprintf("Existing rooms:\n%s\n", strings.Join(roomList, "\n")), currentUser)
}

func writeMessage(message string, user user) {
	_, err := oi.LongWriteString(user.writer, message)
	if err != nil {
		//
		// Something terrible happened
		//
		logger.Printf("ERROR: failed to send message %s to client %s: %+v\n", message, user.name, err)
	}
}

func getUserInput(reader telnet.Reader, bytes []byte, messageBuilder *strings.Builder) bool {
	for {
		n, err := reader.Read(bytes)
		if n > 0 {
			bytes := bytes[:n]
			if bytes[0] == '\n' {
				continue
			} else if bytes[0] == '\r' {
				//
				// Break from loop
				//
				break
			} else {
				messageBuilder.Write(bytes)
			}
		}
		//
		// user disconnected
		//
		if err != nil {
			return true
		}
	}
	return false
}

func createRoom(roomName string) room {
	//
	// To ensure concurrency safety, lock writes to the room map
	//
	roomsMutex.Lock()
	createdRoom := room{
		name:  roomName,
		users: make(map[string]user),
		mutex: sync.Mutex{},
	}
	rooms[roomName] = createdRoom
	logger.Printf("Room %s has been created\n", roomName)
	roomsMutex.Unlock()
	return createdRoom
}

func sendMessageToOtherUsers(message, roomName string, senderUser user, users map[string]user) {
	//
	// Format the logs with the room and user
	//
	logger.Println(fmt.Sprintf("broadcast - [%s %s] %s", senderUser.name, roomName, message))
	timestamp := time.Now().Format("15:04 MST")
	for _, user := range users {
		if senderUser.name == user.name {
			continue
		}
		//
		// Format the final message with the user and timestamp
		//
		writeMessage(fmt.Sprintf("[%s %s]: %s\n", senderUser.name, timestamp, message), user)
	}
}

func addUser(currentUser user, room room) {
	room.mutex.Lock()
	room.users[currentUser.name] = currentUser
	logger.Printf("%s entered the room %s\n", currentUser.name, room.name)
	room.mutex.Unlock()
}

func handleUserMessages(reader telnet.Reader, bytes []byte, messageBuilder strings.Builder, selectedRoom room, currentUser user, users map[string]user) {
	for {
		n, err := reader.Read(bytes)
		if n > 0 {
			bytes := bytes[:n]
			if bytes[0] == '\n' {
				continue
			} else if bytes[0] == '\r' {
				//
				// Send message to all other users
				//
				message := messageBuilder.String()
				//
				// Check if message is a command
				//
				if strings.HasPrefix(message, "-r") {
				} else if strings.HasPrefix("-b", "") {
				} else if strings.HasPrefix("-u", "") {
				} else if message == "-lr" {
					listExistingRooms(currentUser)
				} else if message == "-lu" {
					listUsersInRoom(selectedRoom, currentUser)
				} else if message == "-lb" {
				} else if message == "-h" || message == "-help" {
				} else {
					//
					// if not a command, send message to other users
					//
					sendMessageToOtherUsers(message, selectedRoom.name, currentUser, users)
				}
				//
				// Reset everything
				//
				messageBuilder.Reset()
			} else {
				messageBuilder.Write(bytes)
			}
		}
		//
		// handle error case - user left for some reason
		//
		if err != nil {
			//
			// remove user from room
			//
			removeUser(currentUser, selectedRoom)
			//
			// Let other users know that this user left
			//
			sendMessageToOtherUsers(fmt.Sprintf("client %s has left", currentUser.name), selectedRoom.name, currentUser, users)
			break
		}
	}
}

func removeUser(currentUser user, room room) {
	room.mutex.Lock()
	delete(room.users, currentUser.name)
	logger.Printf("User %s left the room %s\n", currentUser.name, room.name)
	room.mutex.Unlock()
}

func main() {
	//
	// Setup flags
	//
	configPath := flag.String("c", "", "Configuration file used to configure the server")
	flag.Parse()
	//
	// Determine if using defaults
	//
	var config configuration
	var err error
	if len(*configPath) == 0 {
		fmt.Println("No configuration file specified with flag '-c'. Using defaults.")
	} else {
		//
		// Open and read the configuration file
		//
		config, err = readConfigurationFile(*configPath)
		if err != nil {
			log.Fatalln(err)
		}
	}
	//
	// Setup log file
	//
	logFile, err := getLogFile(config)
	if err != nil {
		log.Fatalln(err)
	}
	defer closeFile(logFile)
	//
	// write to file and console -- TODO turn off console
	//
	multiWriter := io.MultiWriter(os.Stdout, logFile)
	logger = log.New(multiWriter, "", log.LstdFlags|log.LUTC)
	//
	// Setup main room
	//
	mainRoom := room{
		name:  defaultRoom,
		users: make(map[string]user),
		mutex: sync.Mutex{},
	}
	rooms[mainRoom.name] = mainRoom
	//
	// Do not let stale rooms float around. Clean them up after a certain amount of time
	//
	go cleanupEmptyRooms()
	//
	// Start the TELNET server
	//
	var handler = user{}
	port := config.Port
	if len(port) == 0 {
		logger.Printf("No port provided in the configuration file. Using default port '%s'\n", defaultPort)
		port = defaultPort
	}
	logger.Printf("Starting server on port '%s'...\n", port)
	err = telnet.ListenAndServe(":"+port, handler)
	if nil != err {
		//
		// Fatal will not execute defers, so to ensure we close the log file
		//
		logger.Printf("failed to start server at address %s: %+v\n", config.Port, err)
		closeFile(logFile)
		return
	}
}

func cleanupEmptyRooms() {
	for {
		time.Sleep(15 * time.Second)
		for id, room := range rooms {
			if id == defaultRoom {
				continue
			}
			room.mutex.Lock()
			if len(room.users) == 0 {
				delete(rooms, id)
				logger.Printf("Removed empty room %s\n", id)
			}
			room.mutex.Unlock()
		}
	}
}

func readConfigurationFile(configPath string) (configuration, error) {
	configFile, err := os.Open(configPath)
	if err != nil {
		return configuration{}, errors.Wrapf(err, "failed to open configuration file at location %s", configPath)
	}
	defer closeFile(configFile)
	var config configuration
	err = json.NewDecoder(configFile).Decode(&config)
	if err != nil {
		return config, errors.Wrapf(err, "failed to read the configuration file %s", configPath)
	}
	return config, nil
}

func getLogFile(config configuration) (*os.File, error) {
	logLocation := config.LogLocation
	//
	// If no log file location is provided, use default location
	//
	if len(config.LogLocation) == 0 {
		currentDirectory, err := os.Getwd()
		if err != nil {
			return nil, errors.Wrapf(err, "no log file location provided: could not get the current working directory for the default location")
		}
		fmt.Printf("No log file location provided in the configuration file. Using default log location '%s'\n", path.Join(currentDirectory, defaultLogLocation))
		logLocation = defaultLogLocation
	}
	logFile, err := os.OpenFile(logLocation, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open log file %s", config.LogLocation)
	}
	return logFile, nil
}

func closeFile(file *os.File) {
	err := file.Close()
	if err != nil {
		fmt.Printf("failed to close file: %+v", err)
	}
}
