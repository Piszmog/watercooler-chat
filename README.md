# Watercooler Chat
[![Build Status](https://travis-ci.org/Piszmog/watercooler-chat.svg?branch=develop)](https://travis-ci.org/Piszmog/watercooler-chat)
[![Coverage Status](https://coveralls.io/repos/github/Piszmog/watercooler-chat/badge.svg?branch=develop)](https://coveralls.io/github/Piszmog/watercooler-chat?branch=develop)
[![Go Report Card](https://goreportcard.com/badge/github.com/Piszmog/watercooler-chat)](https://goreportcard.com/report/github.com/Piszmog/watercooler-chat)
[![GitHub release](https://img.shields.io/github/release/Piszmog/watercooler-chat.svg)](https://github.com/Piszmog/watercooler-chat/releases/latest)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Application for a simple chat server to allow for remote teammates to have their 'watercooler moments'.

## Starting Server
The chat server can be started by running `./watercooler-chat` (or `watercooler-chat.exe` on Windows). A configuration file 
can be provided using the `-c` flag. If a configuration file is not provided, defaults are used.

## Server Configuration
The chat server can be configured with the following JSON file. 

```json
{
  "ipAddress": "${the IP Address to run the server on - defaults to localhost}",
  "telnetPort": "${the port to run the TELNET server on - defaults to 5555}",
  "httpPort": "${the port to run the HTTP server on - defaults to 8080}",
  "logFileLocation": "${the location the log file is written to - defaults to {workingDirectory}/log.txt}"
}
```

###### Example
```json
{
  "ipAddress": "localhost",
  "telnetPort": "5555",
  "httpPort": "8080",
  "logFileLocation": "log.txt"
}
```

### TELNETS (Secure TELNET)
TELNETS (Secure TELNET) can be ran by providing a `certificateFile` and a `keyFile` in the configuration file. If not provided, 
TENET (unsecured) will be started.

#### Client Configuration
In addition to the configuration above, the following also configures TLS for TELNET.

```json
{
  "ipAddress": "${the IP Address to run the server on - defaults to localhost}",
  "telnetPort": "${the port to run the TELNET server on - defaults to 5555}",
  "httpPort": "${the port to run the HTTP server on - defaults to 8080}",
  "logFileLocation": "${the location the log file is written to - defaults to {workingDirectory}/log.txt}",
  "certificateFile": "${the location of the certificate file}",
  "keyFile": "${the location of the key file}"
}
```

###### Example
```json
{
  "ipAddress": "localhost",
  "telnetPort": "5555",
  "httpPort": "8080",
  "logFileLocation": "log.txt",
  "certificateFile": "cert.pem",
  "keyFile": "key.pem"
}
```

## TELNET Commands
After connecting to the server via TELNET, the client will be asked to enter a user name and a room to enter. After connecting 
with and choosing user name/room, a number of commands are available to allow a range of functionality. 

### Commands
```text
-r ${room Name} -- change to the specified room. Creates room if doesn't exist
-b ${user Name} -- to block messages from the specified user
-u ${user Name} -- to Unblock messages from the specified user
-lr             -- to list all existing rooms
-lu             -- to list all users in the current room
-lb             -- to list all users currently blocked
-q              -- to quit the chat
-h              -- to list all available commands
```

## HTTP Endpoints
There are two available endpoints to call on the server. There is a `GET` endpoint to query for messages from a room, and 
there is a `POST` endpoint to send messages to a room.

### Send Messages
Sends a message to the room specified in the URL path.

`POST`  
Path: `/rooms/{room name}`  
Header: `Sender-Name:{name of sender}`  
Body: The message to send to users in the room

#### Response Code
| Code | Description |
|---|---|
| 200 | Message was successfully sent to the room |
| 400 | The request is missing the `Sender-Name` header |
| 500 | The request body could not be read |

##### Example
###### Request
```text
POST /rooms/main HTTP/1.1
Host: localhost:8080
Sender-Name: Tester
cache-control: no-cache
Postman-Token: bcb6a566-b434-471b-abd6-c7843af5076f

Hello from HTTP!
```

###### Response
```text
{
  "statusCode": "200",
  "reason": "Message successfully sent."
}
```

### Retrieve Messages
Messages can be retrieved from a room be providing the room name in the URL. Optional queries `sender`, `start`, and `end` 
can be provided as parameters.

`GET`  
Path: `/rooms/{room name}?sender={sender's name}&start=YYYY-MM-ddTHH:mm:ss.sssZ&end=YYYY-MM-ddTHH:mm:ss.sssZ`  
Body: The message to send to users in the room

Where,
* `sender` - Optional - the name of the sender to retrieve messages for
* `start` - Optional - the start time to retrieve messages after from
* `end` - Optional - the end time to retrieve messages before from

#### Response Code
| Code | Description |
|---|---|
| 200 | Message was successfully sent to the room |
| 400 | Either `start` or `end` were not provided in the expected formats |
| 500 | The response payload could not be sent |

##### Example
###### Request
```text
GET /rooms/main?sender=Tester&amp; start=2019-07-05T19:38:00.000Z HTTP/1.1
Host: localhost:8080
User-Agent: PostmanRuntime/7.15.2
Accept: */*
Cache-Control: no-cache
Postman-Token: 198c685c-3637-47fb-96da-eb5d48214c34,674821ce-1f14-4322-992e-a3a618cf146c
Host: localhost:8080
Accept-Encoding: gzip, deflate
Connection: keep-alive
cache-control: no-cache
```

###### Response
```text
[
  {
    "timestamp": "2019-08-06T17:31:58.1671781-06:00",
    "room": "main",
    "sender": "Tester",
    "value": "Hello from HTTP"
  }
]
```

## Limitations
* The HTTP server is not configurable to be HTTPS
* If the server is cycled (stopped/started), all messages, users, and rooms will be lost
  * Writing to a file or DB can help remedy this
* Multi-line message cannot be sent
* It is not quite clear when a user can begin typing messages
* HTTP messages are capped at 500 characters, but messages via TELNET are not capped
* Benchmarks have not been ran to determine the impact of the usages of `sync.RWMutex`

## Known Bugs
* If a chosen user name is not registered in the server, multiple users are able to pick the same name if the users choose the 
name at the same time. There are locks in place to allow concurrent updates to the users, but there is no final validation at this 
step. This can be remedied by returning the user when registered or a boolean.
* There is no scrubbing of messages being written to the log file. It is vulnerable to [Log Forging/Injection](https://www.owasp.org/index.php/Log_Injection)
* Log file is not being rotated, so if the server runs for too long, the log file can get very large and use all the space 
on a machine.

## Dependencies
Dependencies are managed with __Modules__. Dependencies used in this project can be found in the `go.mod` file.

#### Links
* [Errors](https://github.com/pkg/errors)
* [Go-Telnet](https://github.com/reiver/go-telnet)
* [Gorilla Mux](https://github.com/gorilla/mux) 
