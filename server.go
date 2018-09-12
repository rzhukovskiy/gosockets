package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"strconv"
)

var clients = make(map[int64]*websocket.Conn)
var broadcast = make(chan Message)

var upgrader = websocket.Upgrader{}

type Message struct {
	User int64  "user"
	Data string "data"
}

func handleConnections(writer http.ResponseWriter, request *http.Request) {
	clientIds, hasClient := request.URL.Query()["clientId"]

	if hasClient {
		upgrader.CheckOrigin = func(r *http.Request) bool { return true }
		ws, err := upgrader.Upgrade(writer, request, nil)
		if err != nil {
			log.Printf("error: %v", err)
			return
		}

		defer ws.Close()

		clientId, _ := strconv.ParseInt(clientIds[0], 10, 64)
		clients[clientId] = ws
		for {
			var message Message

			err := ws.ReadJSON(&message)

			if err != nil {
				log.Printf("error: %v", err)
				if hasClient {
					delete(clients, clientId)
				}
				break
			}

			broadcast <- message
		}
	} else {
		if err := request.ParseForm(); err != nil {
			fmt.Printf("ParseForm() err: %v", err)
			return
		}

		user, hasUser := request.PostForm["user"]
		if !hasUser {
			log.Println("User id required")
			return
		}
		userId, _ := strconv.ParseInt(user[0], 10, 64)
		data, hasData := request.PostForm["data"]
		if !hasData {
			log.Println("Data required")
			return
		}

		message := Message{User: userId, Data: data[0]}

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

		err := client.WriteMessage(websocket.TextMessage, []byte(message.Data))
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
	log.Fatal("ListenAndServe: ", http.ListenAndServe(":8000", nil))
}
