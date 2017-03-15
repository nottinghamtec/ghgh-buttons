package main

// Button for the GHGH judges to relay winners to TEC.

import (
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/gorilla/mux"
	"log"
)

type Client struct {
	conn *websocket.Conn
	send chan []byte
}

var (
	port = ":8000"

	upgrader = websocket.Upgrader{}

	clients[] *Client
)

func (c Client) writePump() {
	for {
		select {
		case msg := <- c.send:
			c.conn.WriteMessage(websocket.TextMessage, msg)
			log.Printf("[%s][send] OK", c.conn.RemoteAddr())
		}
	}
}


func wsServe(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade: ", err)
		http.Error(w, "upgrade error", 500)
	}

	defer conn.Close()

	vars := mux.Vars(r)
	name := vars["name"]

	log.Println("Connecting client", name)

	client := &Client{conn, make(chan []byte)}
	clients = append(clients, client)

	go client.writePump()

	for {
		mt, msg, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				log.Printf("error: %v", err)
			}
			break
		}

		if mt != websocket.TextMessage {
			log.Printf("[%s][warn] Unkown message type", conn.RemoteAddr())
			continue
		}

		log.Printf("[%s][read] %s", conn.RemoteAddr(), msg)
		for _, c := range clients {
			if c != client {
				c.send <- msg
			}
		}
	}
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/ws/{name}", wsServe)
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("./static/")))

	log.Println("Listening on", port)
	log.Fatal(http.ListenAndServe(port, r))
}
