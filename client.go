package main

import (
	"bytes"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"time"
)

type Client struct {
	hub     *Hub
	conn    *websocket.Conn
	userIds Ids
	send    chan []byte
	addr    string
}

var kubeconfigName string

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			//flush websites and close website would caused ReadMessage err and trigger defer func
			break
		}
		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
		fmt.Printf("userIds[%d, %d] sent messages: %s\n", c.userIds.Uid, c.userIds.Tid, message)
		jsonHandler(message, c.hub.clients[c.userIds].Head.td)

		currentList := c.hub.clients[c.userIds].Head.next
		for currentList != nil {
			currentList.client.send <- []byte("start")
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
			// send to client from server
			//fmt.Printf("%s received msg: %s\n", c.addr, message)
			fmt.Println(string(message))
			w.Write(message)

			/*execute rs creation*/
			//1. create namespace - default use "web" as the namespace
			if c.hub.clients[c.userIds].Head.td.Command == "START" {
				resourceOperator(kubeconfigName,
					"create",
					"pod",
					nameSpace,
					c.hub.clients[c.userIds].Head.td.ResourceType,
					c.hub.clients[c.userIds].Head.td.ResourceType,
					"10Gi",
					2, // TODO
					&c.hub.clients[c.userIds].Head.td.realPvcName)
			} else if c.hub.clients[c.userIds].Head.td.Command == "STOP" {
				resourceOperator(kubeconfigName,
					"delete",
					"pod",
					nameSpace,
					c.hub.clients[c.userIds].Head.td.ResourceType,
					c.hub.clients[c.userIds].Head.td.ResourceType,
					"10Gi",
					2, // TODO
					&c.hub.clients[c.userIds].Head.td.realPvcName)
			}

			/*// Add queued chat messages to the current websocket message.
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write(newline)
				w.Write(<-c.send)
			}*/

			if err := w.Close(); err != nil {
				log.Fatalln("websocket closed: ", err)
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
	// initialize client
	var ids Ids
	client := &Client{
		hub:     hub,
		conn:    conn,
		userIds: ids, // initialize is null
		send:    make(chan []byte),
		addr:    conn.RemoteAddr().String(),
	}

	//read first connected
	_, message, err := client.conn.ReadMessage()
	if err != nil {
		if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
			log.Printf("error: %v", err)
		}
	}
	message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
	// first initialize: get uid and tid only
	jsonHandler(message, &client.userIds)

	client.hub.register <- client

	fmt.Printf("%s is logged in userIds[%d, %d]\n", client.addr, client.userIds.Uid, client.userIds.Tid)
	go client.writePump()
	go client.readPump()
}
