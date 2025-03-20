package websocket

import (
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

var (
	clients   = make(map[*Client]bool)
	clientsMu sync.Mutex
	broadcast = make(chan []byte)
)

type Client struct {
	conn *websocket.Conn
	send chan []byte
}

func addClient(client *Client) {
	clientsMu.Lock()
	defer clientsMu.Unlock()
	clients[client] = true
}

func removeClient(client *Client) {
	clientsMu.Lock()
	defer clientsMu.Unlock()

	if _, ok := clients[client]; ok {
		delete(clients, client)
		close(client.send)
	}
}

func HandleMessages() {
	for {
		msg := <-broadcast
		clientsMu.Lock()
		for client := range clients {
			select {
			case client.send <- msg:
			default:
				removeClient(client)
			}
		}
		clientsMu.Unlock()
	}
}

func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Erro ao atualizar para WebSocket:", err)
		return
	}

	client := &Client{conn: conn, send: make(chan []byte, 256)}
	addClient(client)

	go func() {
		defer removeClient(client)
		for {
			select {
			case message, ok := <-client.send:
				if !ok {
					// Canal fechado
					conn.WriteMessage(websocket.CloseMessage, []byte{})
					return
				}
				if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
					log.Println("Erro ao escrever mensagem:", err)
					return
				}
			}
		}
	}()

	// Goroutine para receber mensagens do cliente
	go func() {
		defer removeClient(client)
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Println("Erro ao ler mensagem:", err)
				}
				return
			}
			broadcast <- message
		}
	}()
}
