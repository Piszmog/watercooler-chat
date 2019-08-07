package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/pkg/errors"
	"io"
	"log"
	"os"
	"path"
)

const (
	defaultLogLocation = "log.txt"
	defaultIPAddress   = "localhost"
)

var logger *log.Logger
var server ChatServer

type configuration struct {
	IPAddress       string `json:"ipAddress"`
	TelnetPort      string `json:"telnetPort"`
	HTTPPort        string `json:"httpPort"`
	LogLocation     string `json:"logFileLocation"`
	CertificateFile string `json:"certificateFile"`
	KeyFile         string `json:"keyFile"`
}

func main() {
	//
	// Setup flags
	//
	configPath := flag.String("c", "", "Configuration file used to configure the server")
	debugMode := flag.Bool("d", false, "Enables debug mode - logs are also written to the console")
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
	// write to file and/or console if debug mode is enabled
	//
	if *debugMode {
		multiWriter := io.MultiWriter(os.Stdout, logFile)
		logger = log.New(multiWriter, "", log.LstdFlags|log.LUTC)
	} else {
		logger = log.New(logFile, "", log.LstdFlags|log.LUTC)
	}
	//
	// Setup chat server
	//
	server = CreateServer()
	server.CreateRoomIfMissing(defaultRoom)
	done := make(chan bool)
	//
	// Start the TELNET server
	//
	go StartTelnetServer(config, done)
	//
	// Start the HTTP server
	//
	go StartHTTPServer(config, done)
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
