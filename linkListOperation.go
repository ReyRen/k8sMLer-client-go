package main

import (
	"errors"
	"fmt"
	"strconv"
)

// first the linklist
type Node struct {
	client *Client
	next   *Node
}
type headNode struct {
	sm         *sendMsg
	rm         *recvMsg
	logchan    chan *sendMsg
	singlechan chan []byte
	next       *Node
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
		sm:         msg.sm,
		rm:         msg.rm,
		logchan:    make(chan *sendMsg),
		singlechan: make(chan []byte),
		next:       nil,
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
		if client.send != nil {
			close(client.send)
		}
		client.addr = ""
		//client.goroutineClose <- []byte("1")
		return nil
	} else {
		current := head.next
		for current.next != nil {
			if current.next.client == client {
				current.next = current.next.next
				if client.send != nil {
					close(client.send)
				}
				client.addr = ""
				//client.goroutineClose <- []byte("1")
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

func (head *SameIdsLinkList) linklistRun() {
	for true {
		select {
		case msgs := <-head.Head.logchan:
			currentList := head.Head.next
			for currentList != nil {
				currentList.client.sendLog <- []byte(strconv.Itoa(msgs.Type)) // sendlog cannot close or won't send to next client
				currentList = currentList.next
			}
		}
	}
}
