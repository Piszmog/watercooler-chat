package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/pkg/errors"
	"github.com/reiver/go-telnet"
	"io"
	"log"
	"os"
	"path"
	"sync"
)

const (
	defaultPort        = "5555"
	defaultLogLocation = "log.txt"
	defaultRoom        = "main"
)

var logger *log.Logger
var server = chatServer{
	rooms:     make(map[string]*chatRoom),
	roomsLock: sync.RWMutex{},
	users:     make(map[string]chatUser),
	usersLock: sync.RWMutex{},
}

type configuration struct {
	IPAddress   string `json:"ipAddress"`
	Port        string `json:"port"`
	LogLocation string `json:"logFileLocation"`
}

type chatServer struct {
	rooms     map[string]*chatRoom
	roomsLock sync.RWMutex
	users     map[string]chatUser
	usersLock sync.RWMutex
}

func (server *chatServer) createRoom(roomName string) *chatRoom {
	//
	// To ensure concurrency safety, lock writes to the chatRoom map
	//
	server.roomsLock.Lock()
	server.rooms[roomName] = &chatRoom{
		name:  roomName,
		users: make(map[string]chatUser),
	}
	server.roomsLock.Unlock()
	logger.Printf("Room %s has been created\n", roomName)
	return server.rooms[roomName]
}

func (server *chatServer) removeRoom(roomName string) {
	//
	// To ensure concurrency safety, lock writes to the chatRoom map
	//
	server.roomsLock.Lock()
	delete(server.rooms, roomName)
	server.roomsLock.Unlock()
	logger.Printf("Room %s is empty. Room has been removed\n", roomName)
}

func (server *chatServer) listRooms() []string {
	server.roomsLock.RLock()
	roomList := make([]string, len(server.rooms))
	index := 0
	for _, exitingRoom := range server.rooms {
		roomList[index] = exitingRoom.name
		index++
	}
	server.roomsLock.RUnlock()
	return roomList
}

func (server *chatServer) addUser(user chatUser) {
	//
	// Ensure concurrency safety
	//
	server.usersLock.Lock()
	server.users[user.name] = user
	server.usersLock.Unlock()
}

func (server *chatServer) removeUser(user chatUser) {
	//
	// Ensure concurrency safety
	//
	server.usersLock.Lock()
	delete(server.users, user.name)
	server.usersLock.Unlock()
	logger.Printf("%s has left the server\n", user.name)
}

func (server *chatServer) userExists(userName string) bool {
	exists := false
	//
	// Ensure concurrency safety
	//
	server.usersLock.RLock()
	if len(server.users[userName].name) != 0 {
		exists = true
	}
	server.usersLock.RUnlock()
	return exists
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
	// Setup chat server
	//
	server.createRoom(defaultRoom)
	// Start the TELNET server
	//
	var userHandler = chatUser{}
	port := config.Port
	if len(port) == 0 {
		logger.Printf("No port provided in the configuration file. Using default port '%s'\n", defaultPort)
		port = defaultPort
	}
	logger.Printf("Starting server on port '%s'...\n", port)
	err = telnet.ListenAndServe(":"+port, userHandler)
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
