package main

import (
	"errors"
	//"github.com/gorilla/websocket"
)

// fine the linklist
type Node struct {
	client *Client
	next   *Node
}
type SameIdsLinkList struct {
	Head *Node
}

func newNode(client *Client, next *Node) *Node {
	return &Node{
		client: client,
		next:   next,
	}
}
func NewSocketList() *SameIdsLinkList {
	head := &Node{
		client: nil,
		next:   nil,
	}
	return &SameIdsLinkList{head}
}

// check whether the link is empty
func (list *SameIdsLinkList) isEmpty() bool {
	return list.Head.next == nil
}

// append list
func (list *SameIdsLinkList) Append(node *Node) {
	current := list.Head
	for {
		if current.next == nil {
			break
		}
		current = current.next
	}
	current.next = node
}

// delete
func (list *SameIdsLinkList) Remove(client *Client) error {
	empty := list.isEmpty()
	if empty {
		return errors.New("This is an empty list")
	}
	current := list.Head
	for current.next != nil {
		if current.next.client == client {
			current.next = current.next.next
			return nil
		}
		current = current.next
	}
	return nil
}

type Hub struct {
	// registered clies
	clients map[Ids]*SameIdsLinkList
	//broadcast chan []byte
	register   chan *Client
	unregister chan *Client
}

func newHub() *Hub {
	return &Hub{
		clients: make(map[Ids]*SameIdsLinkList),
		//broadcast: make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (h *Hub) run() {
	for true {
		select {
		case client := <-h.register:
			if h.clients[client.userIds].isEmpty() {
				// not exist [uid,tid] key
				headList := NewSocketList()
				headList.Append(newNode(client, nil))
				h.clients[client.userIds] = headList
			} else {
				h.clients[client.userIds].Append(newNode(client, nil))
			}
		case client := <-h.unregister:
			h.clients[client.userIds].Remove(client)
			close(client.send)
			/*case message := <- h.broadcast:
			for client := range h.clients[]{

			}*/
		}
	}
}
