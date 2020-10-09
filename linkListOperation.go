package main

import (
	"errors"
	"fmt"
)

// first the linklist
type Node struct {
	client *Client
	next   *Node
}
type headNode struct {
	sm        *sendMsg
	rm        *recvMsg
	broadcast chan *sendMsg
	next      *Node
}
type SameIdsLinkList struct {
	Head *headNode
}

func newNode(client *Client, next *Node) *Node {
	return &Node{
		client: client,
		next:   next,
	}
}

func NewSocketList(msg *msg) *SameIdsLinkList {
	head := &headNode{
		sm:        msg.sm,
		rm:        msg.rm,
		broadcast: make(chan *sendMsg),
		next:      nil,
	}
	return &SameIdsLinkList{head}
}

// check whether the link is empty
func (list *SameIdsLinkList) isEmpty() bool {
	return list.Head.next == nil
}

// append list
func (list *SameIdsLinkList) Append(node *Node) {
	head := list.Head // *headNode
	if head.next == nil {
		head.next = node
	} else {
		current := head.next // *Node
		for {
			if current.next == nil {
				break
			}
			current = current.next
		}
		current.next = node
	}
}

// delete
func (list *SameIdsLinkList) Remove(client *Client) error {
	empty := list.isEmpty() // have node rather than only head
	if empty {
		return errors.New("This is an empty list")
	}
	head := list.Head // *headNode
	if head.next.client == client {
		head.next = head.next.next
		return nil
	} else {
		current := head.next
		for current.next != nil {
			if current.next.client == client {
				current.next = current.next.next
				return nil
			}
			current = current.next
		}
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
		fmt.Printf("Client%d value:%v\n", i+1, current.client.addr)
		current = current.next
	}
	fmt.Printf("Client%d value:%v\n", i+1, current.client.addr)
	return
}
