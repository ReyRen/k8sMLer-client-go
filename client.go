package main

import (
	"bytes"
	"encoding/json"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	hub     *Hub
	conn    *websocket.Conn
	userIds *Ids
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
		_, message, err := c.conn.ReadMessage() // This is a block func, once ws closed, this would be get err
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			//flush websites and close website would caused ReadMessage err and trigger defer func
			return
		}
		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
		//fmt.Printf("userIds[%d, %d] sent messages: %s\n", c.userIds.Uid, c.userIds.Tid, message)
		jsonHandler(message, c.hub.clients[*c.userIds].Head.rm)
		go func() {
			if c.hub.clients[*c.userIds].Head.rm.Type == 2 {
				//1. create namespace - default use "web" as the namespace
				if c.hub.clients[*c.userIds].Head.rm.Content.Command == "START" {
					// assemble sm head type as resourceInfo
					//c.hub.clients[*c.userIds].Head.sm.Type = 2
					resourceOperator(c,
						kubeconfigName,
						"create",
						"pod",
						nameSpace,
						c.hub.clients[*c.userIds].Head.rm.Content.ResourceType,
						c.hub.clients[*c.userIds].Head.rm.Content.ResourceType,
						"10Gi",
						c.hub.clients[*c.userIds].Head.rm.Content.SelectedNodes,
						&c.hub.clients[*c.userIds].Head.rm.realPvcName)

				} else if c.hub.clients[*c.userIds].Head.rm.Content.Command == "STOP" {
					// assemble sm head type as resourceInfo
					//c.hub.clients[*c.userIds].Head.sm.Type = 2
					resourceOperator(c,
						kubeconfigName,
						"delete",
						"pod",
						nameSpace,
						c.hub.clients[*c.userIds].Head.rm.Content.ResourceType,
						c.hub.clients[*c.userIds].Head.rm.Content.ResourceType,
						"10Gi",
						c.hub.clients[*c.userIds].Head.rm.Content.SelectedNodes,
						&c.hub.clients[*c.userIds].Head.rm.realPvcName)
				}
			}
		}()
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

			if typeCode == STATUSRESPOND {
				// Status code
				sdmsg, _ := json.Marshal(c.hub.clients[*c.userIds].Head.sm)
				w.Write(sdmsg)
			} else if typeCode == RESOURCERESPOND {
				// resource msg
				/*sdmsg, _ := json.Marshal(c.hub.clients[*c.userIds].Head.sm)
				w.Write(sdmsg)*/
			} else if typeCode == LOGRESPOND {
				sdmsg, _ := json.Marshal(c.hub.clients[*c.userIds].Head.sm)
				w.Write(sdmsg)

				logStatusMsg := strings.Split(c.hub.clients[*c.userIds].Head.sm.Content.Log, " ")
				if logStatusMsg[len(logStatusMsg)-1] == TRAININGLOGDONE {
					c.hub.clients[*c.userIds].Head.sm.Type = STATUSRESPOND
					c.hub.clients[*c.userIds].Head.sm.Content.StatusCode = TRAININGSTOPSUCCESS
					c.hub.broadcast <- c

				} else if logStatusMsg[len(logStatusMsg)-1] == TRAININGLOGERR {

					c.hub.clients[*c.userIds].Head.sm.Type = STATUSRESPOND
					c.hub.clients[*c.userIds].Head.sm.Content.StatusCode = TRAININGSTOPFAILED
					c.hub.broadcast <- c

				} else if logStatusMsg[len(logStatusMsg)-1] == TRAININGLOGSTART {

					c.hub.clients[*c.userIds].Head.sm.Type = STATUSRESPOND
					c.hub.clients[*c.userIds].Head.sm.Content.StatusCode = TRAININGSTART
					c.hub.broadcast <- c
				}
			} else {
				sdmsg, _ := json.Marshal(c.hub.clients[*c.userIds].Head.sm)
				w.Write(sdmsg)
			}
			if err := w.Close(); err != nil {
				log.Println("websocket closed:", err)
				//log.Fatalln("websocket closed: ", err)
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
	// initialize client, create memory for msg
	var ids Ids
	var msgs msg
	var rmtmp recvMsg
	var smtmp sendMsg
	var recvMsgContenttmp recvMsgContent
	var sendMsgConetenttmp sendMsgContent
	var sendMsgContentGputmp sendMsgContentGpu
	var resourceInfotmp resourceInfo

	recvMsgContenttmp.IDs = &ids
	rmtmp.Content = &recvMsgContenttmp
	smtmp.Content = &sendMsgConetenttmp
	smtmp.Content.GpuInfo = &sendMsgContentGputmp
	smtmp.Content.ResourceInfo = &resourceInfotmp
	msgs.rm = &rmtmp
	msgs.sm = &smtmp

	client := &Client{
		hub:     hub,
		conn:    conn,
		userIds: rmtmp.Content.IDs, // initialize is null
		send:    make(chan []byte),
		addr:    conn.RemoteAddr().String(),
	}
	// assemble client to once msg
	msgs.cltmp = client

	//read first connection
	_, message, err := client.conn.ReadMessage()
	if err != nil {
		if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
			log.Printf("error: %v", err)
		}
	}
	message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))

	//fmt.Println(string(message))

	jsonHandler(message, &rmtmp)

	client.hub.register <- &msgs

	set_gpu_rest(msgs.cltmp)
	//fmt.Printf("%s is logged in userIds[%d, %d]\n", msgs.cltmp.addr, msgs.cltmp.userIds.Uid, msgs.cltmp.userIds.Tid)
	client.hub.broadcast <- msgs.cltmp

	go msgs.cltmp.logDisplay()
	go msgs.cltmp.writePump()
	go msgs.cltmp.readPump()

	// used for !first client and log already in
	if msgs.cltmp.hub.clients[*msgs.cltmp.userIds].Head.logFlag == LOGSTART {
		msgs.cltmp.hub.clients[*msgs.cltmp.userIds].Head.logChan <- msgs.cltmp.hub.clients[*msgs.cltmp.userIds].Head.logFlag
		return
	}
}
