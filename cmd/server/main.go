package main

import (
	"log"
	"net/http"

	"github.com/Arilson1/go-chat/internal/websocket"
	"github.com/gorilla/mux"
)

func main() {
	r := mux.NewRouter()

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("servidor de chat em tempo real!"))
	})

	r.HandleFunc("/ws", websocket.HandleWebSocket)

	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	// Inicia a goroutine para transmitir mensagens
	go websocket.HandleMessages()

	log.Println("Servidor rodando em :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
