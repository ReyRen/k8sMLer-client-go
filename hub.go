package main

import (
	"fmt"
)

type Hub struct {
	// registered clients
	clients    map[TrainingData]*SameIdsLinkList
	register   chan *Client
	unregister chan *Client
}

func newHub() *Hub {
	return &Hub{
		clients:    make(map[TrainingData]*SameIdsLinkList),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (h *Hub) run() {
	for true {
		select {
		case client := <-h.register:
			if h.clients[client.userIds] == nil {
				// not exist [uid,tid] key
				headList := NewSocketList()
				/*
					Here should initialize common training paramaters
				*/
				headList.Append(newNode(client, nil))
				h.clients[client.userIds] = headList
				fmt.Printf("userIds[%d, %d]: -- ", client.userIds.Uid, client.userIds.Tid)
				headList.PrintList()
			} else {
				headlist := h.clients[client.userIds]
				headlist.Append(newNode(client, nil))
				fmt.Printf("userIds[%d, %d]: -- ", client.userIds.Uid, client.userIds.Tid)
				headlist.PrintList()
			}
		case client := <-h.unregister:
			h.clients[client.userIds].Remove(client)
			fmt.Printf("%s is logged out from userIds[%d, %d]\n", client.addr, client.userIds.Uid, client.userIds.Tid)
			if client.send != nil {
				close(client.send)
			}
		}
	}
}
