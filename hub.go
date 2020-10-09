package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"time"
)

type Hub struct {
	// registered clients
	clients    map[Ids]*SameIdsLinkList
	register   chan *msg
	unregister chan *Client
}

func newHub() *Hub {
	return &Hub{
		clients:    make(map[Ids]*SameIdsLinkList),
		register:   make(chan *msg),
		unregister: make(chan *Client),
	}
}

func (h *Hub) run() {
	for true {
		select {
		case msg := <-h.register:
			if h.clients[*msg.cltmp.userIds] == nil {
				// not exist [uid,tid] key
				headList := NewSocketList(msg)
				headList.Append(newNode(msg.cltmp, nil))
				h.clients[*msg.cltmp.userIds] = headList
				fmt.Printf("userIds[%d, %d]: -- ", msg.cltmp.userIds.Uid, msg.cltmp.userIds.Tid)
				headList.PrintList()
			} else {
				headlist := h.clients[*msg.cltmp.userIds]
				headlist.Append(newNode(msg.cltmp, nil))
				fmt.Printf("userIds[%d, %d]: -- ", msg.cltmp.userIds.Uid, msg.cltmp.userIds.Tid)
				headlist.PrintList()
			}
		case client := <-h.unregister:
			h.clients[*client.userIds].Remove(client)
			fmt.Printf("%s is logged out from userIds[%d, %d]\n", client.addr, client.userIds.Uid, client.userIds.Tid)
			if client.send != nil {
				close(client.send)
			}
		}
	}
}

func (c *Client) handle_broadcast() {
	for true {
		select {
		case sm, ok := <-c.hub.clients[*c.userIds].Head.broadcast:
			//typeCode, _ := strconv.Atoi(string(msg))
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

			gpuSend, err := json.Marshal(sm)
			if err != nil {
				log.Fatalln("json.Marshal err ", err)
			}
			w.Write(gpuSend)

			if err := w.Close(); err != nil {
				log.Fatalln("websocket closed: ", err)
				return
			}
		}
	}
}
