package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/reiver/go-oi"
	"github.com/reiver/go-telnet"
	"log"
	"os"
	"path"
	"strings"
	"time"
)

const (
	defaultPort        = "5555"
	defaultLogLocation = "log.txt"
)

var logger *log.Logger
var clients = make(map[string]handler)

type configuration struct {
	IPAddress   string `json:"ipAddress"`
	Port        string `json:"port"`
	LogLocation string `json:"logFileLocation"`
}

type handler struct {
	id     string
	writer telnet.Writer
}

func (handler handler) ServeTELNET(ctx telnet.Context, w telnet.Writer, r telnet.Reader) {
	id := uuid.New().String()
	handler.writer = w
	handler.id = id
	for _, handler := range clients {
		oi.LongWriteString(handler.writer, "client "+id+" has entered\n")
	}
	clients[id] = handler
	var buffer [1]byte
	p := buffer[:]
	builder := strings.Builder{}
	builder.WriteString(id)
	builder.WriteString(":")
	builder.WriteString(" ")
	isTimestampSet := false
	for {
		n, err := r.Read(p)
		if n > 0 {
			if !isTimestampSet {
				builder.WriteString("[")
				builder.WriteString(time.Now().Format("15:04 MST"))
				builder.WriteString("]")
				builder.WriteString(" ")
				isTimestampSet = true
			}
			bytes := p[:n]
			if bytes[0] == '\n' {
				continue
			} else if bytes[0] == '\r' {
				builder.WriteByte('\n')
				input := builder.String()
				logger.Print(input)
				for _, handler := range clients {
					if id == handler.id {
						continue
					}
					oi.LongWriteString(handler.writer, input)
				}
				isTimestampSet = false
				builder.Reset()
				builder.WriteString(id)
				builder.WriteString(":")
				builder.WriteString(" ")
			} else {
				builder.Write(bytes)
			}
		}
		if nil != err {
			delete(clients, id)
			for _, handler := range clients {
				oi.LongWriteString(handler.writer, "client "+id+" has left\n")
			}
			break
		}
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
	logger = log.New(logFile, "", log.LstdFlags|log.LUTC)
	//
	// Start the TELNET server
	//
	var handler = handler{}
	port := config.Port
	if len(port) == 0 {
		fmt.Printf("No port provided in the configuration file. Using default port '%s'\n", defaultPort)
		port = defaultPort
	}
	fmt.Printf("Starting server on port '%s'...\n", port)
	err = telnet.ListenAndServe(":"+port, handler)
	if nil != err {
		//
		// Fatal will not execute defers, so to ensure we close the log file
		//
		closeFile(logFile)
		log.Fatalf("failed to start server at address %s: %+v\n", config.Port, err)
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
