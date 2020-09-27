package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"time"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second
	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second
	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10
	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

type Client struct {
	hub     *Hub
	conn    *websocket.Conn
	userIds Ids
	send    chan []byte
	addr    string
}

/*// define registered map y
type clientMapKey struct {
	ids Ids
	broadcast chan []byte
}*/

type Ids struct {
	Uid int `json:"uid"`
	Tid int `json:"tid"`
}

func (c *Client) readPump() {
	defer func() {
		fmt.Println("111111111111")
		c.hub.unregister <- c
		c.conn.Close()
	}()
	fmt.Println("222222222")
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
		fmt.Printf("messages: %s\n", message)
		//fmt.Printf("the received msg when connected is : %s \n", u)
		//fmt.Println(c.uim.Tid)*/
		//c.hub.broadcast <- message
		currentList := c.hub.clients[c.userIds].Head.next
		for currentList != nil {
			currentList.client.send <- message
			currentList = currentList.next
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)
			fmt.Printf("%s received msg: %s\n", c.addr, message)

			// Add queued chat messages to the current websocket message.
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write(newline)
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func serveWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	// conn, err := upgrader.Upgrade(w, r, nil)
	upgrader.CheckOrigin = func(r *http.Request) bool {
		return true
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		// mute : websocket: the client is not using the websocket protocol: 'upgrade' token not found in 'Connection' header
		//log.Printf("upgrade err: %v\n", err)
		return
	}
	// initialize clie
	var ids Ids
	client := &Client{
		hub:     hub,
		conn:    conn,
		userIds: ids, // initialize is null
		send:    nil,
		addr:    conn.RemoteAddr().String(),
	}
	_, message, err := client.conn.ReadMessage()
	if err != nil {
		if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
			log.Printf("error: %v", err)
		}
	}
	message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
	fmt.Printf("messages: %s\n", message)
	// first initialize
	errJson := json.Unmarshal(message, &client.userIds)
	if errJson != nil {
		log.Fatalln("json err: ", errJson)
	}

	client.hub.register <- client

	fmt.Printf("%s is logged in and registered, UID:%d, TID:%d\n", client.addr, client.userIds.Uid, client.userIds.Tid)
	go client.readPump()
	go client.writePump()
}
