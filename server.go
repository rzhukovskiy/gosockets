package main

import (
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"strconv"
)

var clients = make(map[int64]*websocket.Conn)
var broadcast = make(chan Message)

var upgrader = websocket.Upgrader{}

type Message struct {
	Dialog  int64  `json:"dialog"`
	User    int64  `json:"user"`
	Token   string `json:"token"`
	Message string `json:"message"`
}

func handleConnections(writer http.ResponseWriter, request *http.Request) {
	clientIds, ok := request.URL.Query()["clientId"]
	if !ok {
		log.Println("Client ID is missing")
		return
	}
	clientId, _ := strconv.ParseInt(clientIds[0], 10, 64)

	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	ws, err := upgrader.Upgrade(writer, request, nil)
	if err != nil {
		log.Fatal(err)
	}

	defer ws.Close()

	clients[clientId] = ws

	for {
		var message Message

		err := ws.ReadJSON(&message)

		if err != nil {
			log.Printf("error: %v", err)
			delete(clients, clientId)
			break
		}

		broadcast <- message
	}
}

func handleMessages() {
	for {
		message := <-broadcast

		clientId := message.User
		client, ok := clients[clientId]
		if !ok {
			continue
		}

		err := client.WriteJSON(message)
		if err != nil {
			log.Printf("error: %v", err)
			client.Close()
			delete(clients, clientId)
		}
	}
}

func main() {
	http.HandleFunc("/", handleConnections)
	go handleMessages()

	log.Println("http server started on :8000")
	err := http.ListenAndServe(":8000", nil)

	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
