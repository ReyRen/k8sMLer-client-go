package main

import (
	"fmt"
	"log"
	"strconv"
)

type Hub struct {
	// registered clients
	clients    map[Ids]*SameIdsLinkList
	register   chan *msg
	broadcast  chan *Client
	unregister chan *Client
}

func newHub() *Hub {
	return &Hub{
		clients:    make(map[Ids]*SameIdsLinkList),
		register:   make(chan *msg),
		broadcast:  make(chan *Client),
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
				go headList.linklistRun() // used for log control
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
		case broadcastClient := <-h.broadcast: // only broadcast client msg(small)
			currentList := broadcastClient.hub.clients[*broadcastClient.userIds].Head.next
			for currentList != nil {
				currentList.client.send <- []byte(strconv.Itoa(broadcastClient.hub.clients[*broadcastClient.userIds].Head.sm.Type))
				currentList = currentList.next
			}
		case client := <-h.unregister:
			err := h.clients[*client.userIds].Remove(client)
			if err != nil {
				log.Println("map remove err:", err)
			}
			fmt.Printf("%s is logged out from userIds[%d, %d]\n", client.addr, client.userIds.Uid, client.userIds.Tid)
		}
	}
}
