package main

import (
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"net/url"
	"strings"
)

var clients = make(map[string]*websocket.Conn)
var broadcast = make(chan Message)

var upgrader = websocket.Upgrader{}

type Message struct {
	Dialog  string `json:"dialog"`
	User    string `json:"user"`
	Token   string `json:"token"`
	Message string `json:"message"`
}

func handleConnections(writer http.ResponseWriter, request *http.Request) {
	clientIds, ok := request.URL.Query()["clientId"]
	if !ok {
		log.Println("Client ID is missing")
		return
	}
	clientId := clientIds[0]

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

		log.Println(message)

		dialogUrl := "https://api.pm.iwad.ru/dialog/" + message.Dialog
		bearer := "Bearer " + message.Token
		form := url.Values{}
		form.Add("message", message.Message)
		req, err := http.NewRequest("POST", dialogUrl, strings.NewReader(form.Encode()))
		req.Header.Add("Authorization", bearer)
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		httpClient := &http.Client{}
		res, err := httpClient.Do(req)
		if err != nil {
			log.Println("Error on response.\n[ERRO] -", err)
		}
		defer res.Body.Close()

		clientId := message.User
		client, ok := clients[clientId]
		if !ok {
			continue
		}

		err = client.WriteJSON(message)
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
