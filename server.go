package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/reiver/go-telnet"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"sync"
	"time"
)

const (
	defaultTelnetPort  = "5555"
	defaultHTTPPort    = "8080"
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
	TelnetPort  string `json:"telnetPort"`
	HTTPPort    string `json:"httpPort"`
	LogLocation string `json:"logFileLocation"`
}

type chatServer struct {
	rooms     map[string]*chatRoom
	roomsLock sync.RWMutex
	users     map[string]chatUser
	usersLock sync.RWMutex
}

func (server *chatServer) createRoomIfMissing(roomName string) *chatRoom {
	//
	// To ensure concurrency safety, use lock
	//
	server.roomsLock.Lock()
	//
	// Check if room exists - another goroutine could have created it
	//
	if server.rooms[roomName] == nil {
		server.rooms[roomName] = &chatRoom{
			name:  roomName,
			users: make(map[string]chatUser),
		}
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

func (server *chatServer) getRoom(roomName string) *chatRoom {
	server.roomsLock.RLock()
	selectedRoom := server.rooms[roomName]
	server.roomsLock.RUnlock()
	selectedRoom = server.createRoomIfMissing(roomName)
	return selectedRoom
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
	server.createRoomIfMissing(defaultRoom)
	done := make(chan bool)
	//
	// Start the TELNET server
	//
	go startTelnetServer(config, done)
	//
	// Start the HTTP server
	//
	go startHTTPServer(config, done)
	<-done
	logger.Println("Stopping server...")
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

func startTelnetServer(config configuration, done chan bool) {
	var userHandler = chatUser{}
	port := config.TelnetPort
	if len(port) == 0 {
		logger.Printf("No Telnet port provided in the configuration file. Using default Telnet port '%s'\n", defaultTelnetPort)
		port = defaultTelnetPort
	}
	logger.Printf("Starting Telnet server on port '%s'...\n", port)
	err := telnet.ListenAndServe(":"+port, userHandler)
	if nil != err {
		//
		// Fatal will not execute defers, so to ensure we close the log file
		//
		logger.Printf("failed to start Telnet server at address %s: %+v\n", config.TelnetPort, err)
	}
	done <- true
}

func startHTTPServer(config configuration, done chan bool) {
	port := config.TelnetPort
	if len(port) == 0 {
		logger.Printf("No HTTP port provided in the configuration file. Using default HTTP port '%s'\n", defaultHTTPPort)
		port = defaultHTTPPort
	}
	logger.Printf("Starting HTTP server on port '%s'...\n", port)
	r := mux.NewRouter()
	r.HandleFunc("/rooms/{name}", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method == http.MethodGet {
			//
			// Query messages from a room
			//
			fmt.Fprintln(writer, "Hello there... this is not implemented yet...")
		} else {
			//
			// Send messages to a room
			//
			sendMessage(writer, request)
		}
	}).Methods(http.MethodGet, http.MethodPost)
	srv := &http.Server{
		Addr:         "0.0.0.0:" + port,
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      r,
	}
	if err := srv.ListenAndServe(); err != nil {
		logger.Printf("failed to start HTTP server at addredd %s: %+v\n", port, err)
	}
	done <- true
}

func sendMessage(writer http.ResponseWriter, request *http.Request) {
	variables := mux.Vars(request)
	roomName := variables["name"]
	if len(roomName) == 0 {
		// TODO
	}
	senderName := request.Header.Get("Sender-Name")
	if len(senderName) == 0 {
		// TODO
	}
	if request.ContentLength > 500 {
		// TODO
	}
	room := server.getRoom(roomName)
	body := request.Body
	bytes, err := ioutil.ReadAll(body)
	if err != nil {
		// TODO
	}
	// todo log
	room.broadcast(senderName + " " + string(bytes))
	// todo format and with timestamp
}
