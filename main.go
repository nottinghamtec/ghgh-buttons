package main

// Button for the GHGH judges to relay winners to TEC.

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"log"
)

type Message struct {
	Src      string `json:"src"`
	Dest     string `json:"dest"`
	MsgType  string `json:"type"`
	MsgValue string `json:"value"`
}

type Client struct {
	name string
	conn *websocket.Conn
	send chan Message
}

var (
	port = ":8000"

	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	clients = map[string]*Client{}
)

func (c Client) writePump() {
	for {
		select {
		case msg := <-c.send:
			c.conn.WriteJSON(msg)
			log.Printf("[%s][%s][send] OK", msg.Src, msg.Dest)
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

	client := &Client{name, conn, make(chan Message)}
	clients[name] = client

	go client.writePump()

	for {
		jsonMsg := Message{}
		err := conn.ReadJSON(&jsonMsg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				log.Printf("error: %v", err)
			}
			break
		}

		if jsonMsg.Src == "" {
			jsonMsg.Src = name
		}

		log.Printf("[%s][%s][read] %s: %s", jsonMsg.Src, jsonMsg.Dest, jsonMsg.MsgType, jsonMsg.MsgValue)

		if jsonMsg.Dest == "" {
			for _, c := range clients {
				if c.name != jsonMsg.Src {
					jsonMsg.Dest = c.name
					c.send <- jsonMsg
				}
			}
		}

		dest := clients[jsonMsg.Dest]
		if dest != nil {
			clients[jsonMsg.Dest].send <- jsonMsg
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
