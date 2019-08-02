package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/reiver/go-oi"
	"github.com/reiver/go-telnet"
	"io"
	"log"
	"os"
	"path"
	"strings"
	"time"
)

const (
	defaultPort        = "5555"
	defaultLogLocation = "log.txt"
	defaultRoom        = "main"
)

var logger *log.Logger
var rooms = make(map[string]room)

type configuration struct {
	IPAddress   string `json:"ipAddress"`
	Port        string `json:"port"`
	LogLocation string `json:"logFileLocation"`
}

type room struct {
	name  string
	users map[string]user
}

type user struct {
	id     string
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
	id := uuid.New().String()
	currentUser.writer = w
	currentUser.id = id
	//
	// Prepare buffer, message builder, and timestamp for incoming messages
	//
	var buffer [1]byte
	p := buffer[:]
	messageBuilder := strings.Builder{}
	//
	// Determine the user's name and room
	//
	for len(currentUser.name) == 0 {
		writeMessage(currentUser, "What is your name? ")
		if getUserInput(r, p, &messageBuilder) {
			return
		}
		currentUser.name = messageBuilder.String()
		messageBuilder.Reset()
		if len(currentUser.name) == 0 {
			writeMessage(currentUser, "A user name is required.\n")
		}
	}
	writeMessage(currentUser, "What room would you like to enter (default to main)? ")
	if getUserInput(r, p, &messageBuilder) {
		return
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
		rooms[roomName] = room{
			name:  roomName,
			users: make(map[string]user),
		}
	}
	//
	// Let the user know who else is in the room
	//
	users := selectedRoom.users
	userList := make([]string, len(users))
	for _, user := range users {
		userList = append(userList, user.name)
	}
	writeMessage(currentUser, fmt.Sprintf("Users currently in the room:%s\n", strings.Join(userList, "\n")))
	//
	// Let the other users know a new user joins them
	//
	sendMessageToOtherUsers(fmt.Sprintf("client %s has entered", currentUser.id), selectedRoom.name, currentUser.id, users)
	users[currentUser.id] = currentUser
	//
	// Start sending messages to the other users
	//
	handleUserMessages(r, p, messageBuilder, selectedRoom, currentUser, users)
}

func writeMessage(user user, message string) {
	_, err := oi.LongWriteString(user.writer, message)
	if err != nil {
		//
		// Something terrible happened
		//
		logger.Printf("ERROR: failed to send message %s to client %s: %+v\n", message, user.id, err)
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
				sendMessageToOtherUsers(message, selectedRoom.name, currentUser.id, users)
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
			// remove user from map
			//
			delete(users, currentUser.id)
			//
			// Let other users know that this user left
			//
			sendMessageToOtherUsers(fmt.Sprintf("client %s has left", currentUser.id), selectedRoom.name, currentUser.id, users)
			break
		}
	}
}

func sendMessageToOtherUsers(message, roomName, senderId string, users map[string]user) {
	//
	// Format the logs with the room and user
	//
	logger.Println(fmt.Sprintf("[%s %s] %s", senderId, roomName, message))
	timestamp := time.Now().Format("15:04 MST")
	for _, user := range users {
		if senderId == user.id {
			continue
		}
		//
		// Format the final message with the user and timestamp
		//
		writeMessage(user, fmt.Sprintf("[%s %s]: %s\n", senderId, timestamp, message))
	}
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
	}
	rooms[mainRoom.name] = mainRoom
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
