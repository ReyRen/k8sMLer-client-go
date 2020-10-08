package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"strconv"
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
		jsonHandler(message, c.hub.clients[c.userIds].Head.rm)

		currentList := c.hub.clients[c.userIds].Head.next
		for currentList != nil {
			currentList.client.send <- []byte(strconv.Itoa(c.hub.clients[c.userIds].Head.rm.Type))
			currentList = currentList.next
		}
	}
}

/*func (c *Client) writePumpInit() {
	// This shouldn't close conn
	/*defer func() {
		c.conn.Close()
	}()
	c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	w, err := c.conn.NextWriter(websocket.TextMessage)
	if err != nil {
		log.Fatalln("c.conn.NextWriter: ", err)
	}
	capacity, used := get_gpu_rest()
	fmt.Printf("capacity GPU : %s", string(capacity))
	fmt.Printf("used GPU : %s", string(used))

	m := map[string]string{
		"gpuCapacity": string(capacity),
		"gpuUsed":     string(used),
	}
	gpuSend, _ := json.Marshal(m)
	w.Write(gpuSend)

	if err := w.Close(); err != nil {
		log.Fatalln("websocket closed: ", err)
	}
}*/

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case msg, ok := <-c.send:
			typeCode, _ := strconv.Atoi(string(msg))
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

			if typeCode == 1 {
				// initialize information
				gpuSend, _ := json.Marshal(c.hub.clients[c.userIds].Head.sm)
				w.Write(gpuSend)

			} else if typeCode == 2 {
				// start training and stop training msg
				//1. create namespace - default use "web" as the namespace
				if c.hub.clients[c.userIds].Head.rm.Content.Command == "START" {
					resourceOperator(kubeconfigName,
						"create",
						"pod",
						nameSpace,
						c.hub.clients[c.userIds].Head.rm.Content.ResourceType,
						c.hub.clients[c.userIds].Head.rm.Content.ResourceType,
						"10Gi",
						c.hub.clients[c.userIds].Head.rm.Content.SelectedNodes,
						&c.hub.clients[c.userIds].Head.rm.realPvcName)
				} else if c.hub.clients[c.userIds].Head.rm.Content.Command == "STOP" {
					resourceOperator(kubeconfigName,
						"delete",
						"pod",
						nameSpace,
						c.hub.clients[c.userIds].Head.rm.Content.ResourceType,
						c.hub.clients[c.userIds].Head.rm.Content.ResourceType,
						"10Gi",
						c.hub.clients[c.userIds].Head.rm.Content.SelectedNodes,
						&c.hub.clients[c.userIds].Head.rm.realPvcName)
				}
			}
			// send to client from server
			//fmt.Printf("%s received msg: %s\n", c.addr, message)

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
	//jsonHandler(message, &client.userIds)
	var rmtmp recvMsg
	var recvMsgContenttmp recvMsgContent
	rmtmp.Content = recvMsgContenttmp
	fmt.Println(string(message))
	jsonHandler(message, &rmtmp)
	client.userIds.Uid = rmtmp.Content.Uid
	client.userIds.Tid = rmtmp.Content.Tid
	//jsonHandler(message, &client.userIds)

	client.hub.register <- client
	set_gpu_rest(client)

	fmt.Printf("%s is logged in userIds[%d, %d]\n", client.addr, client.userIds.Uid, client.userIds.Tid)

	go client.writePump()
	go client.readPump()
}
