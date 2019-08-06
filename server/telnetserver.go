package main

import "github.com/reiver/go-telnet"

const (
	defaultTelnetPort = "5555"
)

// StartTelnetServer start a TELNET server.
func StartTelnetServer(config configuration, done chan bool) {
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
