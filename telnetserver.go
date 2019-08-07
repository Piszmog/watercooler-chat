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
	ipAddress := config.IPAddress
	if len(ipAddress) == 0 {
		logger.Printf("No IP Address provided in the configuration file. Using default IP Address '%s'\n", defaultIPAddress)
		ipAddress = defaultIPAddress
	}
	logger.Printf("Starting Telnet server on '%s'...\n", ipAddress+":"+port)
	//
	// Start server
	//
	var err error
	if len(config.CertificateFile) != 0 && len(config.KeyFile) != 0 {
		err = telnet.ListenAndServeTLS(ipAddress+":"+port, config.CertificateFile, config.KeyFile, userHandler)
	} else {
		logger.Println("A certificate and key file were not provided. Telnet server will start in unsecured mode.")
		err = telnet.ListenAndServe(ipAddress+":"+port, userHandler)
	}
	if nil != err {
		//
		// Fatal will not execute defers, so to ensure we close the log file
		//
		logger.Printf("failed to start Telnet server at address %s: %+v\n", config.TelnetPort, err)
	}
	done <- true
}
