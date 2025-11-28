package core

import (
	"log"

	"findme/schema"

	"github.com/gorilla/websocket"
)

type Client struct {
	Conn     *websocket.Conn
	UserID   string
	ChatID   string
	SendChan chan *schema.ViewMessage
}

type BroadcastMessage struct {
	ChatID string
	Data   *schema.ViewMessage
}

type ChatHub struct {
	Room       map[string]map[*Client]bool
	Register   chan *Client
	UnRegister chan *Client
	Broadcast  chan *BroadcastMessage
}

func NewClient(conn *websocket.Conn, uid, cid string, data chan *schema.ViewMessage) *Client {
	return &Client{
		Conn:     conn,
		UserID:   uid,
		ChatID:   cid,
		SendChan: data,
	}
}

func NewChatHub(buffersize int) *ChatHub {
	return &ChatHub{
		Room:       make(map[string]map[*Client]bool),
		Register:   make(chan *Client),
		UnRegister: make(chan *Client),
		Broadcast:  make(chan *BroadcastMessage, buffersize),
	}
}

func (c *Client) ReadPump(hub *ChatHub) {
	defer func() {
		hub.UnRegister <- c
		_ = c.Conn.Close()
	}()

	var msg schema.ViewMessage
	for {
		if err := c.Conn.ReadJSON(&msg); err != nil {
			break
		}

		hub.Broadcast <- &BroadcastMessage{
			Data:   &msg,
			ChatID: c.ChatID,
		}
	}
}

func (c *Client) WritePump() {
	defer c.Conn.Close()

	for msg := range c.SendChan {
		if err := c.Conn.WriteJSON(msg); err != nil {
			break
		}
	}
}

func (h *ChatHub) Run() {
	log.Println("[Chat HUB] THe Chat HUB is up and running")
	for {
		select {
		case c := <-h.Register:
			if h.Room[c.ChatID] == nil {
				h.Room[c.ChatID] = make(map[*Client]bool)
			}
			h.Room[c.ChatID][c] = true
		case c := <-h.UnRegister:
			if room := h.Room[c.ChatID]; room != nil {
				delete(room, c)
				close(c.SendChan)
			}
		case msg := <-h.Broadcast:
			room := h.Room[msg.ChatID]
			for c := range room {
				select {
				case c.SendChan <- msg.Data:
				default:
					close(c.SendChan)
					delete(room, c)
				}
			}
		}
	}
}
