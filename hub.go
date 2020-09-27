package main

import (
	"errors"
	"fmt"
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

// print
func (list *SameIdsLinkList) PrintList() {
	empty := list.isEmpty()
	if empty {
		fmt.Println("This is an empty list")
		return
	}
	current := list.Head.next
	fmt.Println("The elements is:")
	i := 0
	for ; ; i++ {
		if current.next == nil {
			break
		}
		fmt.Println("INode%d,value:%v --> ", i, current.client.addr)
		current = current.next
	}
	fmt.Printf("Node%d value:%v\n", i+1, current.client.addr)
	return
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
			if h.clients[client.userIds] == nil {
				// not exist [uid,tid] key
				headList := NewSocketList()
				headList.Append(newNode(client, nil))
				h.clients[client.userIds] = headList
				headList.PrintList()
			} else {
				headlist := h.clients[client.userIds]
				headlist.Append(newNode(client, nil))
				headlist.PrintList()
			}
		case client := <-h.unregister:
			h.clients[client.userIds].Remove(client)
			if client.send != nil {
				close(client.send)
			}
			/*case message := <- h.broadcast:
			for client := range h.clients[]{

			}*/
		}
	}
}
