# Watercooler Chat
[![Build Status](https://travis-ci.org/Piszmog/watercooler-chat.svg?branch=develop)](https://travis-ci.org/Piszmog/watercooler-chat)
[![Coverage Status](https://coveralls.io/repos/github/Piszmog/watercooler-chat/badge.svg?branch=develop)](https://coveralls.io/github/Piszmog/watercooler-chat?branch=develop)
[![Go Report Card](https://goreportcard.com/badge/github.com/Piszmog/watercooler-chat)](https://goreportcard.com/report/github.com/Piszmog/watercooler-chat)
[![GitHub release](https://img.shields.io/github/release/Piszmog/watercooler-chat.svg)](https://github.com/Piszmog/watercooler-chat/releases/latest)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Application for a simple chat server to allow for remote teammates to have their 'watercooler moments'.

## Server Configuration

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

#### Client Configuration

## Starting Server

## TELNET Commands

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

## Dependencies
Dependencies used in this project can be found in the `go.mod` file.

* [Errors](https://github.com/pkg/errors)
* [Go-Telnet](https://github.com/reiver/go-telnet)
* [Gorilla Mux](https://github.com/gorilla/mux) 
