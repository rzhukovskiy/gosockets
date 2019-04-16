package daemon

import (
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
)

var clients = make(map[string]*websocket.Conn)
var broadcast = make(chan Message)

var upgrader = websocket.Upgrader{}

type Config struct {
	Listen string
}

type Message struct {
	User string
	Data string
}

func checkUserId(h http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(r.URL.Query().Get("clientId")) == 0 {
			log.Println("User id is missing")
			http.Error(w, "User id is missing", http.StatusBadRequest)
			return
		}
		h.ServeHTTP(w, r)
	})
}

func handleSockets(writer http.ResponseWriter, request *http.Request) {
	clientIds, _ := request.URL.Query()["clientId"]
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }

	ws, err := upgrader.Upgrade(writer, request, nil)
	if err != nil {
		log.Printf("Upgrade error: %v", err)
		return
	}

	defer ws.Close()

	clientId := clientIds[0]
	clients[clientId] = ws
	for {
		var message Message

		err := ws.ReadJSON(&message)

		if err != nil {
			log.Printf("Read error: %v", err)
			delete(clients, clientId)
			break
		}
	}
}

func handlePost(_ http.ResponseWriter, request *http.Request) {
	if err := request.ParseForm(); err != nil {
		fmt.Printf("ParseForm() err: %v", err)
		return
	}

	user, hasUser := request.PostForm["user"]
	if !hasUser {
		log.Println("User id is missing")
		return
	}
	data, hasData := request.PostForm["data"]
	if !hasData {
		log.Println("Data is missing")
		return
	}

	userId := user[0]
	message := Message{User: userId, Data: data[0]}

	broadcast <- message
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
			log.Printf("Message error: %v", err)
			log.Printf("Message: %v", message.Data)
			client.Close()
			delete(clients, clientId)
		}
	}
}

func Run(config *Config) error {
	http.HandleFunc("/ws", checkUserId(handleSockets))
	http.HandleFunc("/socket", handlePost)
	go handleMessages()

	log.Printf("http server started on %v", config.Listen)
	return http.ListenAndServe(config.Listen, nil)
}
