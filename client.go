package main

import (
	"bytes"
	"encoding/json"
	"github.com/gorilla/websocket"
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
	sendLog chan []byte
	addr    string
}

var kubeconfigName string

func (c *Client) readPump() {
	defer func() {
		err := c.hub.clients[*c.userIds].Remove(c)
		if err != nil {
			Error.Printf("[%d, %d]: map remove err:%s\n", c.userIds.Uid, c.userIds.Tid, err)
		}
		err = c.conn.Close()
		if err != nil {
			Error.Printf("[%d, %d]: readPump conn close err: %s\n", c.userIds.Uid, c.userIds.Tid, err)
		}
	}()
	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { _ = c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, message, err := c.conn.ReadMessage() // This is a block func, once ws closed, this would be get err
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				Error.Printf("[%d, %d]: readMessage error: %s\n", c.userIds.Uid, c.userIds.Tid, err)
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

					if c.hub.clients[*c.userIds].Head.sm.Content.StatusCode == TRAININGSTOPSUCCESS {
						clientSocket(c, ENDTRAININGSTOPNORMAL)
					} else {
						clientSocket(c, ENDTRAININGSTOPFAIL)
					}

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

func (c *Client) sendGpuMsg() {
	/*defer func() {
		_ = c.conn.Close()
	}()*/
	_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	w, err := c.conn.NextWriter(websocket.TextMessage)
	if err != nil {
		Error.Printf("[%d, %d]: handle log nextWriter error:%s\n", c.userIds.Uid, c.userIds.Tid, err)
		return
	}
	sdmsg, _ := json.Marshal(c.hub.clients[*c.userIds].Head.sm)
	_, err = w.Write(sdmsg)
	if err != nil {
		Error.Printf("[%d, %d]: sendGpuMsg write err: %s\n", c.userIds.Uid, c.userIds.Tid, err)
	}

	if err := w.Close(); err != nil {
		Error.Printf("[%d, %d]: websocket closed error: %s\n", c.userIds.Uid, c.userIds.Tid, err)
		return
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()
	for {
		select {
		case msg, ok := <-c.sendLog: // handle log
			typeCode, _ := strconv.Atoi(string(msg))
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				Error.Printf("[%d, %d]: handle log channel error\n", c.userIds.Uid, c.userIds.Tid)
				return
			}
			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				Error.Printf("[%d, %d]: handle log nextWriter error:%s\n", c.userIds.Uid, c.userIds.Tid, err)
				return
			}

			if typeCode == RSRESPOND {
				//logStatusMsg := strings.Split(c.hub.clients[*c.userIds].Head.sm.Content.Log, " ")
				c.hub.clients[*c.userIds].Head.sm.Type = RSRESPOND
				c.hub.broadcast <- c
			}

			if typeCode == LOGRESPOND {

				logStatusMsg := strings.Split(c.hub.clients[*c.userIds].Head.sm.Content.Log, " ")
				if strings.Contains(logStatusMsg[len(logStatusMsg)-1], TRAINLOGDONE) {
					c.hub.clients[*c.userIds].Head.sm.Type = STATUSRESPOND
					c.hub.clients[*c.userIds].Head.sm.Content.StatusCode = TRAININGSTOPSUCCESS
					c.hub.broadcast <- c
					clientSocket(c, ENDTRAININGSTOPNORMAL)

				} else if strings.Contains(logStatusMsg[len(logStatusMsg)-1], TRAINLOGERR) {
					c.hub.clients[*c.userIds].Head.sm.Type = STATUSRESPOND
					c.hub.clients[*c.userIds].Head.sm.Content.StatusCode = TRAININGSTOPFAILED
					c.hub.broadcast <- c
					clientSocket(c, ENDTRAININGSTOPFAIL)

				} else if strings.Contains(logStatusMsg[len(logStatusMsg)-1], TRAINLOGSTART) {

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
					Error.Printf("[%d, %d]: sendlog chan write err: %s\n", c.userIds.Uid, c.userIds.Tid, err)
				}
			}
			if err := w.Close(); err != nil {
				Error.Printf("[%d, %d]: websocket closed err: %s\n", c.userIds.Uid, c.userIds.Tid, err)
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
				Error.Printf("[%d, %d]: send chan write err: %s\n", c.userIds.Uid, c.userIds.Tid, err)
			}

			if err := w.Close(); err != nil {
				Error.Printf("[%d, %d]: websocket closed error: %s\n", c.userIds.Uid, c.userIds.Tid, err)
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
		hub:     hub,
		conn:    conn,
		userIds: rmtmp.Content.IDs, // initialize is null
		send:    make(chan []byte),
		sendLog: make(chan []byte),
		addr:    conn.RemoteAddr().String(),
	}
	// assemble client to once msg
	msgs.cltmp = client

	//read first connection
	_, message, err := client.conn.ReadMessage()
	if err != nil {
		if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
			Error.Printf("[%d, %d]: handle log channel error: %s\n", client.userIds.Uid, client.userIds.Tid, err)
		}
	}
	message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))

	jsonHandler(message, &rmtmp)

	client.hub.register <- &msgs
	set_gpu_rest(msgs.cltmp)

	/*
		not use  client.hub.broadcast <- msgs.cltmp for broadcast
		because send channel blocked after flash flush(first got new and then exit old)
		so, execute by themself and not broadcast.
		broadcast msg only log msg
	*/
	msgs.cltmp.sendGpuMsg()

	go msgs.cltmp.writePump()
	go msgs.cltmp.readPump()
}
