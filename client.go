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
	hub            *Hub
	conn           *websocket.Conn
	userIds        *Ids
	send           chan []byte
	sendLog        chan []byte
	goroutineClose chan []byte
	addr           string
}

var kubeconfigName string

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		err := c.conn.Close()
		if err != nil {
			log.Println("readPump conn close err: ", err)
		}
	}()
	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { _ = c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
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
					//handle socket with the frontend
					clientSocket(c, WAITINGRESOURCE)
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
		err := c.conn.Close()
		if err != nil {
			log.Println("writePump conn close err: ", err)
		}
	}()
	for {
		select {
		case msg, ok := <-c.sendLog: // handle log
			typeCode, _ := strconv.Atoi(string(msg))
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}

			if typeCode == LOGRESPOND {

				logStatusMsg := strings.Split(c.hub.clients[*c.userIds].Head.sm.Content.Log, " ")

				if logStatusMsg[len(logStatusMsg)-1] == TRAININGLOGDONE {
					c.hub.clients[*c.userIds].Head.sm.Type = STATUSRESPOND
					c.hub.clients[*c.userIds].Head.sm.Content.StatusCode = TRAININGSTOPSUCCESS
					c.hub.broadcast <- c
					clientSocket(c, ENDTRAININGSTOPNORMAL)

				} else if logStatusMsg[len(logStatusMsg)-1] == TRAININGLOGERR {

					c.hub.clients[*c.userIds].Head.sm.Type = STATUSRESPOND
					c.hub.clients[*c.userIds].Head.sm.Content.StatusCode = TRAININGSTOPFAILED
					c.hub.broadcast <- c
					clientSocket(c, ENDTRAININGSTOPFAIL)

				} else if logStatusMsg[len(logStatusMsg)-1] == TRAININGLOGSTART {

					c.hub.clients[*c.userIds].Head.sm.Type = STATUSRESPOND
					c.hub.clients[*c.userIds].Head.sm.Content.StatusCode = TRAININGSTART
					c.hub.broadcast <- c
					clientSocket(c, ENDTRAININGSTART)
					// block
					c.hub.clients[*c.userIds].Head.signalChan <- []byte("?")
				}
				sdmsg, _ := json.Marshal(c.hub.clients[*c.userIds].Head.sm)
				_, err := w.Write(sdmsg)
				if err != nil {
					log.Println("sendlog chan write err: ", err)
				}
			}
			if err := w.Close(); err != nil {
				log.Println("websocket closed:", err)
				return
			}
		case _, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}

			sdmsg, _ := json.Marshal(c.hub.clients[*c.userIds].Head.sm) // STATUSRESPOND or GPU
			_, err = w.Write(sdmsg)
			if err != nil {
				log.Println("send chan write err: ", err)
			}

			if err := w.Close(); err != nil {
				log.Println("websocket closed:", err)
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func serveWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool {
		return true
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		// mute : websocket: the client is not using the websocket protocol: 'upgrade' token not found in 'Connection' header
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
		hub:            hub,
		conn:           conn,
		userIds:        rmtmp.Content.IDs, // initialize is null
		send:           make(chan []byte),
		sendLog:        make(chan []byte),
		goroutineClose: make(chan []byte),
		addr:           conn.RemoteAddr().String(),
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

	jsonHandler(message, &rmtmp)

	client.hub.register <- &msgs

	set_gpu_rest(msgs.cltmp)

	client.hub.broadcast <- msgs.cltmp

	go msgs.cltmp.writePump()
	go msgs.cltmp.readPump()
}
