package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/piszmog/watercooler-chat/server/message"
	"github.com/pkg/errors"
	"github.com/reiver/go-telnet"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"time"
)

const (
	defaultTelnetPort     = "5555"
	defaultHTTPPort       = "8080"
	defaultLogLocation    = "log.txt"
	queryTimestampFormat  = "2006-01-02T15:04:05.000Z"
	headerContentType     = "Content-Type"
	headerContentTypeJSON = "application/json"
	headerSenderName      = "Sender-Name"
	pathRoom              = "/rooms/{name}"
	pathVariableName      = "name"
	parameterSender       = "sender"
	parameterStart        = "start"
	parameterEnd          = "end"
)

var logger *log.Logger
var server ChatServer

type configuration struct {
	IPAddress   string `json:"ipAddress"`
	TelnetPort  string `json:"telnetPort"`
	HTTPPort    string `json:"httpPort"`
	LogLocation string `json:"logFileLocation"`
}

func main() {
	//
	// Setup flags
	//
	configPath := flag.String("c", "", "Configuration file used to configure the s")
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
	server = CreateServer()
	server.CreateRoomIfMissing(defaultRoom)
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
	logger.Println("Stopping s...")
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
	var userHandler = ChatUser{}
	port := config.TelnetPort
	if len(port) == 0 {
		logger.Printf("No Telnet port provided in the configuration file. Using default Telnet port '%s'\n", defaultTelnetPort)
		port = defaultTelnetPort
	}
	logger.Printf("Starting Telnet s on port '%s'...\n", port)
	//
	// Start server
	//
	err := telnet.ListenAndServe(":"+port, userHandler)
	if nil != err {
		//
		// Fatal will not execute defers, so to ensure we close the log file
		//
		logger.Printf("failed to start Telnet s at address %s: %+v\n", config.TelnetPort, err)
	}
	done <- true
}

func startHTTPServer(config configuration, done chan bool) {
	port := config.TelnetPort
	if len(port) == 0 {
		logger.Printf("No HTTP port provided in the configuration file. Using default HTTP port '%s'\n", defaultHTTPPort)
		port = defaultHTTPPort
	}
	logger.Printf("Starting HTTP s on port '%s'...\n", port)
	//
	// Create and start server
	//
	srv := createHTTPServer(port)
	if err := srv.ListenAndServe(); err != nil {
		logger.Printf("failed to start HTTP s at addredd %s: %+v\n", port, err)
	}
	done <- true
}

func createHTTPServer(port string) *http.Server {
	r := mux.NewRouter()
	//
	// Setup route the the GET and POST for messages to/from a room
	//
	r.HandleFunc(pathRoom, func(writer http.ResponseWriter, request *http.Request) {
		//
		// Always close the request body
		//
		defer closeBody(request.Body)
		writer.Header().Add(headerContentType, headerContentTypeJSON)
		variables := mux.Vars(request)
		roomName := variables[pathVariableName]
		//
		// Room name is required
		//
		if len(roomName) == 0 {
			writer.WriteHeader(http.StatusBadRequest)
			logger.Println("ERROR: HTTP POST request missing room name")
			writeHttpMessage(writer, `{"statusCode":"400", "reason":"Room name cannot be blank"}`)
			return
		}
		if request.Method == http.MethodGet {
			//
			// Query messages from a room
			//
			getMessages(request, writer, roomName)
		} else {
			//
			// Send messages to a room
			//
			sendHTTPMessage(writer, request, roomName)
		}
	}).Methods(http.MethodGet, http.MethodPost)
	srv := &http.Server{
		Addr:         "0.0.0.0:" + port,
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      r,
	}
	return srv
}

func getMessages(request *http.Request, writer http.ResponseWriter, roomName string) {
	logger.Println("Received HTTP request to get messages")
	//
	// Start building the query
	//
	query := message.Query{
		RoomName:   roomName,
		SenderName: request.FormValue(parameterSender),
	}
	//
	// Set start is provided
	//
	start := request.FormValue(parameterStart)
	if len(start) != 0 {
		startTime, err := parseTime(start)
		if err != nil {
			writer.WriteHeader(http.StatusBadRequest)
			logger.Printf("ERROR: HTTP GET request start format incorrect: %+v\n", err)
			writeHttpMessage(writer, `{"statusCode":"400", "reason":"Time format for parameter 'start' is in the incorrect format. Use format 'YYYY-MM-ddTHH:mm:ss.sssZ'"}`)
			return
		}
		query.Start = startTime
	}
	//
	// Set end if provided
	//
	end := request.FormValue(parameterEnd)
	if len(end) != 0 {
		endTime, err := parseTime(end)
		if err != nil {
			writer.WriteHeader(http.StatusBadRequest)
			logger.Printf("ERROR: HTTP GET request end format incorrect: %+v\n", err)
			writeHttpMessage(writer, `{"statusCode":"400", "reason":"Time format for parameter 'end' is in the incorrect format. Use format 'YYYY-MM-ddTHH:mm:ss.sssZ'"}`)
			return
		}
		query.End = endTime
	}
	//
	// Get the room from server
	//
	room := server.GetRoom(roomName)
	if room == nil {
		writer.WriteHeader(http.StatusNotFound)
		writeHttpMessage(writer, `{"statusCode":"404", "reason":"Room `+roomName+` does not exist"}`)
		return
	}
	//
	// Get the messages and send back to the HTTP client
	//
	responseBytes, err := json.MarshalIndent(room.GetMessages(query), "", "  ")
	//
	// Handle error
	//
	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		logger.Printf("ERROR: failed to marshal message response: %+v\n", err)
		writeHttpMessage(writer, `{"statusCode":"500", "reason":"Failed to serialize response data"}`)
		return
	}
	//
	// Ship the OK response
	//
	writer.WriteHeader(http.StatusOK)
	_, err = writer.Write(responseBytes)
	if err != nil {
		logger.Printf("ERROR: failed to write response: %+v\n", err)
	} else {
		logger.Println("Sent room messages to HTTP client")
	}
}

func parseTime(timestamp string) (time.Time, error) {
	parsedTime, err := time.Parse(queryTimestampFormat, timestamp)
	if err != nil {
		return time.Time{}, errors.Wrapf(err, "failed to parse %s to time using format %s", timestamp, queryTimestampFormat)
	}
	return parsedTime, nil
}

func sendHTTPMessage(writer http.ResponseWriter, request *http.Request, roomName string) {
	//
	// Get the required header value
	//
	senderName := request.Header.Get(headerSenderName)
	if len(senderName) == 0 {
		writer.WriteHeader(http.StatusBadRequest)
		logger.Println("ERROR: HTTP POST request missing 'Sender-Name'")
		writeHttpMessage(writer, `{"statusCode":"400", "reason":"Missing Header 'Sender-Name'"}`)
		return
	}
	logger.Println("Received HTTP request to send a message from user " + senderName)
	u := ChatUser{
		Name: senderName,
	}
	//
	// Lookup the room
	//
	room := server.GetRoom(roomName)
	if room == nil {
		writer.WriteHeader(http.StatusNotFound)
		writeHttpMessage(writer, `{"statusCode":"404", "reason":"Room `+roomName+` does not exist"}`)
		return
	}
	//
	// Get the message to send
	//
	body := request.Body
	var buf []byte
	//
	// Prevent too large of messages
	//
	if request.ContentLength > 500 {
		buf = make([]byte, 500)
	} else {
		buf = make([]byte, request.ContentLength)
	}
	_, err := body.Read(buf)
	//
	// Handle error
	//
	if err != nil && err != io.EOF {
		writer.WriteHeader(http.StatusInternalServerError)
		logger.Printf("ERROR: failed to handle HTTP POST request: %+v\n", err)
		writeHttpMessage(writer, `{"statusCode":"500", "reason":"Failed to handle request"}`)
		return
	}
	//
	// Send message to room
	//
	u.SendMessage(string(buf), room)
	if request.ContentLength > 500 {
		writer.WriteHeader(http.StatusOK)
		logger.Println("WARN: HTTP POST request body greater than 500 bytes. Truncating message")
		writeHttpMessage(writer, `{"statusCode":"200", "reason":"Message successfully sent, but was truncated for being larger than 500 characters."}`)
	} else {
		writer.WriteHeader(http.StatusOK)
		writeHttpMessage(writer, `{"statusCode":"200", "reason":"Message successfully sent."}`)
		logger.Println("Sent HTTP message to room " + roomName + " from user " + senderName)
	}
}

func writeHttpMessage(writer http.ResponseWriter, message string) {
	_, err := writer.Write([]byte(message))
	if err != nil {
		logger.Printf("ERROR: failed to send message to HTTP client: %+v\n", err)
	}
}

func closeBody(reader io.ReadCloser) {
	err := reader.Close()
	if err != nil {
		logger.Printf("ERROR: failed to close HTTP body: %+v\n", err)
	}
}
