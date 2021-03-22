package main

import "strconv"

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
				headList.Append(newNode(msg.cltmp, nil))
				h.clients[*msg.cltmp.userIds] = headList
				go headList.linklistRun() // used for log control
				headList.PrintList()
			} else {
				headlist := h.clients[*msg.cltmp.userIds]
				headlist.Append(newNode(msg.cltmp, nil))
				go headlist.linklistRun()
				headlist.PrintList()
			}
		case broadcastClient := <-h.broadcast: // only broadcast client msg(small)
			if broadcastClient.hub.clients[*broadcastClient.userIds].Head.listCount == 1 {
				if !(broadcastClient.hub.clients[*broadcastClient.userIds].Head.next.client.Admin) {
					broadcastClient.hub.clients[*broadcastClient.userIds].Head.next.client.send <- []byte(strconv.Itoa(broadcastClient.hub.clients[*broadcastClient.userIds].Head.sm.Type))
				}
			} else if broadcastClient.hub.clients[*broadcastClient.userIds].Head.listCount == 2 {
				if !(broadcastClient.hub.clients[*broadcastClient.userIds].Head.next.client.Admin) {
					broadcastClient.hub.clients[*broadcastClient.userIds].Head.next.client.send <- []byte(strconv.Itoa(broadcastClient.hub.clients[*broadcastClient.userIds].Head.sm.Type))
				} else if !(broadcastClient.hub.clients[*broadcastClient.userIds].Head.next.next.client.Admin) {
					broadcastClient.hub.clients[*broadcastClient.userIds].Head.next.next.client.send <- []byte(strconv.Itoa(broadcastClient.hub.clients[*broadcastClient.userIds].Head.sm.Type))
				}
			}
		}
	}
}
