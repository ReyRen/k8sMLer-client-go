package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"net/http"
	"os"
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
		err := c.conn.Close()
		if err != nil {
			Error.Printf("[%d, %d]: readPump conn close err: %s\n", c.userIds.Uid, c.userIds.Tid, err)
		}
		err = c.hub.clients[*c.userIds].Remove(c)
		if err != nil {
			Error.Printf("[%d, %d]: map remove err:%s\n", c.userIds.Uid, c.userIds.Tid, err)
		}
	}()
	/*c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { _ = c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })*/
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
		fmt.Printf("userIds[%d, %d] sent messages: %s\n", c.userIds.Uid, c.userIds.Tid, message)
		jsonHandler(message, c.hub.clients[*c.userIds].Head.rm)
		go func() {
			if c.hub.clients[*c.userIds].Head.rm.Type == 2 {
				//1. create namespace - default use "web" as the namespace
				if c.hub.clients[*c.userIds].Head.rm.Content.Command == "START" {
					QUEUELIST = append(QUEUELIST, c.hub.clients[*c.userIds].Head)
					Trace.Println("len(QUEUELIST):", len(QUEUELIST))
					//handle socket with the frontend
					clientSocket(c, WAITINGRESOURCE)
					for {
						if QUEUELIST[0] == nil {
							Trace.Println("Waiting list is empty...quit")
							break
						} else if QUEUELIST[0] == c.hub.clients[*c.userIds].Head {
							if c.hub.clients[*c.userIds].Head.ScheduleMap == BEFORECREATE {
								// start creating
								c.hub.clients[*c.userIds].Head.ScheduleMap = CREATING
								resourceOperator(c,
									kubeconfigName,
									"create",
									"pod",
									nameSpace,
									c.hub.clients[*c.userIds].Head.rm.Content.ResourceType,
									c.hub.clients[*c.userIds].Head.rm.Content.ResourceType,
									"10Gi",
									c.hub.clients[*c.userIds].Head.rm.Content.SelectedNodes,
									c.hub.clients[*c.userIds].Head.rm.RandomName)
								break
							} else if c.hub.clients[*c.userIds].Head.ScheduleMap == CREATING {
								Trace.Println("Your task is creating...")
								QUEUELIST = QUEUELIST[1:]
								break
							} else if c.hub.clients[*c.userIds].Head.ScheduleMap == POSTCREATE {
								Trace.Println("Your task already created...")
								QUEUELIST = QUEUELIST[1:]
								break
							}
						} else if QUEUELIST[0] != c.hub.clients[*c.userIds].Head {
							if QUEUELIST[0].ScheduleMap == CREATING || QUEUELIST[0].ScheduleMap == BEFORECREATE {
								//	Trace.Printf("[%d, %d] is creating, please wait for a while...\n", QUEUELIST[0].next.client.userIds.Uid, QUEUELIST[0].next.client.userIds.Tid)
								time.Sleep(time.Second * 3)
							} else if QUEUELIST[0].ScheduleMap == POSTCREATE {
								QUEUELIST = QUEUELIST[1:]
								continue
							}
						}
					}
				} else if c.hub.clients[*c.userIds].Head.rm.Content.Command == "STOP" {

					//if c.hub.clients[*c.userIds].Head.sm.Content.StatusCode == TRAININGSTOPSUCCESS {
					if c.hub.clients[*c.userIds].Head.sm.Type == TRAININGSTOPSUCCESS {
						clientSocket(c, ENDTRAININGSTOPNORMAL)
					} else {
						clientSocket(c, ENDTRAININGSTOPFAIL)
					}
					c.hub.clients[*c.userIds].Head.ScheduleMap = POSTCREATE
					resourceOperator(c,
						kubeconfigName,
						"delete",
						"pod",
						nameSpace,
						c.hub.clients[*c.userIds].Head.rm.Content.ResourceType,
						c.hub.clients[*c.userIds].Head.rm.Content.ResourceType,
						"10Gi",
						c.hub.clients[*c.userIds].Head.rm.Content.SelectedNodes,
						c.hub.clients[*c.userIds].Head.rm.RandomName)
				} else if c.hub.clients[*c.userIds].Head.rm.Content.Command == "RESTART" {
					lock.Lock()
					c.hub.clients[*c.userIds].Head.ScheduleMap = BEFORECREATE
					c.hub.clients[*c.userIds].Head.ips = ""
					lock.Unlock()
					resourceOperator(c,
						kubeconfigName,
						"delete",
						"pod",
						nameSpace,
						c.hub.clients[*c.userIds].Head.rm.Content.ResourceType,
						c.hub.clients[*c.userIds].Head.rm.Content.ResourceType,
						"10Gi",
						c.hub.clients[*c.userIds].Head.rm.Content.SelectedNodes,
						c.hub.clients[*c.userIds].Head.rm.RandomName)
				} else if c.hub.clients[*c.userIds].Head.rm.Content.Command == "RESET" {
					c.hub.clients[*c.userIds].Head.sm.Type = TRAININGRESET
				}
			}
		}()
	}
}

func (c *Client) sendGpuMsg() {
	_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	w, err := c.conn.NextWriter(websocket.TextMessage)
	if err != nil {
		Error.Printf("[%d, %d]: handle log nextWriter error:%s\n", c.userIds.Uid, c.userIds.Tid, err)
		return
	}
	/*if strings.Contains(c.hub.clients[*c.userIds].Head.sm.Content.Log, TRAINLOGSTART) {
		c.hub.clients[*c.userIds].Head.sm.Type = 0
		c.hub.clients[*c.userIds].Head.signalChan <- []byte("?")
	}*/
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
				Error.Printf("[%d, %d]: c.sendLog channel error\n", c.userIds.Uid, c.userIds.Tid)
				return
			}
			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				Error.Printf("[%d, %d]: handle log nextWriter error:%s\n", c.userIds.Uid, c.userIds.Tid, err)
				return
			}

			if typeCode == RSRESPOND {
				//logStatusMsg := strings.Split(c.hub.clients[*c.userIds].Head.sm.Content.Log, " ")
				c.hub.clients[*c.userIds].Head.sm.Type = typeCode
				c.hub.broadcast <- c
			}

			if typeCode == LOGRESPOND {
				//logStatusMsg := strings.Split(c.hub.clients[*c.userIds].Head.sm.Content.Log, " ")
				if strings.Contains(c.hub.clients[*c.userIds].Head.sm.Content.Log, TRAINLOGDONE) {
					/*c.hub.clients[*c.userIds].Head.sm.Type = STATUSRESPOND
					c.hub.clients[*c.userIds].Head.sm.Content.StatusCode = TRAININGSTOPSUCCESS*/
					c.hub.clients[*c.userIds].Head.sm.Type = TRAININGSTOPSUCCESS
					c.hub.broadcast <- c
					break
					// block

				} else if strings.Contains(c.hub.clients[*c.userIds].Head.sm.Content.Log, TRAINLOGERR) {
					/*c.hub.clients[*c.userIds].Head.sm.Type = STATUSRESPOND
					c.hub.clients[*c.userIds].Head.sm.Content.StatusCode = TRAININGSTOPFAILED*/
					if c.hub.clients[*c.userIds].Head.mideng == 0 {
						c.hub.clients[*c.userIds].Head.sm.Type = TRAININGSTOPFAILED
						c.hub.broadcast <- c
						lock.Lock()
						c.hub.clients[*c.userIds].Head.mideng = 1
						lock.Unlock()
					}
					break
					// block

				} else if strings.Contains(c.hub.clients[*c.userIds].Head.sm.Content.Log, TRAINLOGSTART) {
					/*c.hub.clients[*c.userIds].Head.sm.Type = STATUSRESPOND
					c.hub.clients[*c.userIds].Head.sm.Content.StatusCode = TRAININGSTART*/
					c.hub.clients[*c.userIds].Head.sm.Type = TRAININGSTART
					c.hub.broadcast <- c
					// block
					//c.hub.clients[*c.userIds].Head.signalChan <- []byte("?")
				} else {
					sdmsg, _ := json.Marshal(c.hub.clients[*c.userIds].Head.sm)
					_, err := w.Write(sdmsg)
					if err != nil {
						Error.Printf("[%d, %d]: sendlog chan write err: %s\n", c.userIds.Uid, c.userIds.Tid, err)
					}
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
				//Error.Printf("[%d, %d]: c.send channel error\n", c.userIds.Uid, c.userIds.Tid)
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

func serveWs(hub *Hub, w http.ResponseWriter, r *http.Request, mod string) {
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
	var resourceInfotmp resourceInfo
	var selectNodeSlice []selectNodes

	recvMsgContenttmp.IDs = &ids
	rmtmp.Content = &recvMsgContenttmp
	rmtmp.Content.SelectedNodes = &selectNodeSlice
	smtmp.Content = &sendMsgConetenttmp
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
	get_node_info(msgs.cltmp)

	/*
		not use  client.hub.broadcast <- msgs.cltmp for broadcast
		because send channel blocked after flash flush(first got new and then exit old)
		so, execute by themself and not broadcast.
		broadcast msg only log msg                       c
	*/
	msgs.cltmp.sendGpuMsg()

	go msgs.cltmp.writePump()
	go msgs.cltmp.readPump()

	/* used to update */
	if mod == MOD_UPDATE {

		Trace.Println("entry into the update mode")

		tmpbyte := make([]byte, 4096)

		mapKey := strconv.Itoa(client.userIds.Uid) + "-" + strconv.Itoa(client.userIds.Tid)

		//if _, ok := UPDATEMAP
		file, error := os.OpenFile(".update", os.O_RDONLY, 0766)
		if error != nil {
			fmt.Println(error)
		}

		total, err := file.Read(tmpbyte)
		if err != nil {
			Error.Println(err)
		}

		err = json.Unmarshal(tmpbyte[:total], &UPDATEMAP) // tmpbyte[:total] for error invalid character '\x00' after top-level value
		if err != nil {
			Error.Println(err)
		}

		//Trace.Println(UPDATEMAP[mapKey])
		client.hub.clients[*client.userIds].Head.rm.RandomName = UPDATEMAP[mapKey][0]
		client.hub.clients[*client.userIds].Head.rm.Type, _ = strconv.Atoi(UPDATEMAP[mapKey][1])
		client.hub.clients[*client.userIds].Head.rm.Content.ResourceType = UPDATEMAP[mapKey][2]
		//handle selectednodes
		var i int
		i = 0
		for _, v := range strings.Split(UPDATEMAP[mapKey][3], ",") {
			(*(client.hub.clients[*client.userIds].Head.rm.Content.SelectedNodes))[i].NodeNames = strings.Split(v, "-")[0]
			(*(client.hub.clients[*client.userIds].Head.rm.Content.SelectedNodes))[i].GPUNum, _ = strconv.Atoi(strings.Split(v, "-")[1])
			i++
		}
		statusCode, _ := strconv.Atoi(UPDATEMAP[mapKey][4])

		if statusCode >= RESOURCECOMPLETE {
			// active log
			log_back_to_frontend(client, kubeconfigName, nameSpace,
				client.hub.clients[*client.userIds].Head.rm.Content.ResourceType,
				client.hub.clients[*client.userIds].Head.rm.RandomName,
				len(*(client.hub.clients[*client.userIds].Head.rm.Content.SelectedNodes)),
				(*(client.hub.clients[*client.userIds].Head.rm.Content.SelectedNodes))[0].GPUNum)
		}
		/* used to update */
	}
}
