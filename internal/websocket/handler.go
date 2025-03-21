package websocket

import (
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

var (
	clients   = make(map[*Cliente]bool)
	clientsMu sync.Mutex
	broadcast = make(chan []byte)
)

type Cliente struct {
	conn     *websocket.Conn
	send     chan []byte
	username string
}

func addClient(client *Cliente) {
	clientsMu.Lock()
	defer clientsMu.Unlock()
	clients[client] = true
}

func removeClient(client *Cliente) {
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
			// Verifica se a mensagem é privada
			if len(msg) > 0 && msg[0] == '@' {
				// Extrai o nome do destinatário
				partes := strings.SplitN(string(msg), " ", 2)
				if len(partes) < 2 {
					continue
				}
				destinatario := partes[0][1:] // Remove o '@'
				mensagem := partes[1]

				// Envia a mensagem apenas para o destinatário
				if client.username == destinatario {
					select {
					case client.send <- []byte(mensagem):
					default:
						removeClient(client)
					}
				}
			} else {
				// Transmite a mensagem para todos os clientes
				select {
				case client.send <- msg:
				default:
					removeClient(client)
				}
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

	// Recebe o nome do usuário
	_, usernameBytes, err := conn.ReadMessage()
	if err != nil {
		log.Println("Erro ao ler nome do usuário:", err)
		conn.Close()
		return
	}
	username := string(usernameBytes)

	client := &Cliente{conn: conn, send: make(chan []byte, 256), username: username}
	addClient(client)

	log.Printf("Novo cliente conectado: %s\n", username)

	go func() {
		defer removeClient(client)

		for message := range client.send {
			if err := client.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Println("Erro ao escrever mensagem:", err)
				return
			}
		}
		// If the loop exits, close the WebSocket gracefully
		client.conn.WriteMessage(websocket.CloseMessage, []byte{})
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
