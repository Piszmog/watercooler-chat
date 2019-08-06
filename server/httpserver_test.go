package main

import (
	"bytes"
	"github.com/gorilla/mux"
	"github.com/piszmog/watercooler-chat/server/message"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func init() {
	logger = log.New(os.Stdout, "TEST: ", log.LstdFlags|log.LUTC)
}

func TestHandleRoomRequest_GetAllMessages(t *testing.T) {
	//
	// Setup server
	//
	server = CreateServer()
	defer func() {
		server = CreateServer()
	}()
	server.CreateRoomIfMissing("main")
	room := server.GetRoom("main")
	server.AddUser(&ChatUser{
		Name:   "tester",
		writer: &bytes.Buffer{},
	})
	room.SendMessage(message.ChatMessage{
		Timestamp: time.Date(2019, 2, 1, 1, 1, 1, 0, time.UTC),
		Room:      "main",
		Sender:    "tester",
		Value:     "Hello from HTTP test",
	})
	room.Close()
	room.HandleMessages()
	//
	// Setup HTTP test
	//
	req, err := http.NewRequest(http.MethodGet, "/rooms/main?sender=tester&start=2019-01-01T01:01:00.000Z&end=2020-01-01T01:01:00.000Z", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/rooms/{name}", RoomRequestHandler)
	router.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
	expected := `[
  {
    "timestamp": "2019-02-01T01:01:01Z",
    "room": "main",
    "sender": "tester",
    "value": "Hello from HTTP test"
  }
]`
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
	}
}

func TestHandleRoomRequest_GetMessages_BadStart(t *testing.T) {
	//
	// Setup server
	//
	server = CreateServer()
	defer func() {
		server = CreateServer()
	}()
	server.CreateRoomIfMissing("main")
	room := server.GetRoom("main")
	server.AddUser(&ChatUser{
		Name:   "tester",
		writer: &bytes.Buffer{},
	})
	room.SendMessage(message.ChatMessage{
		Timestamp: time.Date(2019, 2, 1, 1, 1, 1, 0, time.UTC),
		Room:      "main",
		Sender:    "tester",
		Value:     "Hello from HTTP test",
	})
	room.Close()
	room.HandleMessages()
	//
	// Setup HTTP test
	//
	req, err := http.NewRequest(http.MethodGet, "/rooms/main?sender=tester&start=2019-01-01", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/rooms/{name}", RoomRequestHandler)
	router.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}
	expected := `{"statusCode":"400", "reason":"Time format for parameter 'start' is in the incorrect format. Use format 'YYYY-MM-ddTHH:mm:ss.sssZ'"}`
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
	}
}

func TestHandleRoomRequest_GetMessages_BadEnd(t *testing.T) {
	//
	// Setup server
	//
	server = CreateServer()
	defer func() {
		server = CreateServer()
	}()
	server.CreateRoomIfMissing("main")
	room := server.GetRoom("main")
	server.AddUser(&ChatUser{
		Name:   "tester",
		writer: &bytes.Buffer{},
	})
	room.SendMessage(message.ChatMessage{
		Timestamp: time.Date(2019, 2, 1, 1, 1, 1, 0, time.UTC),
		Room:      "main",
		Sender:    "tester",
		Value:     "Hello from HTTP test",
	})
	room.Close()
	room.HandleMessages()
	//
	// Setup HTTP test
	//
	req, err := http.NewRequest(http.MethodGet, "/rooms/main?sender=tester&end=2020-01-01", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/rooms/{name}", RoomRequestHandler)
	router.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}
	expected := `{"statusCode":"400", "reason":"Time format for parameter 'end' is in the incorrect format. Use format 'YYYY-MM-ddTHH:mm:ss.sssZ'"}`
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
	}
}

func TestHandleRoomRequest_PostMessage(t *testing.T) {
	//
	// Setup server
	//
	server = CreateServer()
	defer func() {
		server = CreateServer()
	}()
	//
	// Setup HTTP test
	//
	req, err := http.NewRequest(http.MethodPost, "/rooms/main", bytes.NewBufferString("Message posted from test"))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("Sender-Name", "tester")
	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/rooms/{name}", RoomRequestHandler)
	router.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
	expected := `{"statusCode":"200", "reason":"Message successfully sent."}`
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
	}
}

func TestHandleRoomRequest_PostMessage_NoSenderName(t *testing.T) {
	//
	// Setup server
	//
	server = CreateServer()
	defer func() {
		server = CreateServer()
	}()
	//
	// Setup HTTP test
	//
	req, err := http.NewRequest(http.MethodPost, "/rooms/main", bytes.NewBufferString("Message posted from test"))
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/rooms/{name}", RoomRequestHandler)
	router.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}
	expected := `{"statusCode":"400", "reason":"Missing Header 'Sender-Name'"}`
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
	}
}
