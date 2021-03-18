package main

import (
	"errors"
	"strconv"
)

// first the linklist
type Node struct {
	client *Client
	next   *Node
}
type headNode struct {
	sm          *sendMsg
	rm          *recvMsg
	logchan     chan *sendMsg
	signalChan  chan []byte
	ips         string
	mideng      int // forbiden multiple 111Err111
	next        *Node
	ScheduleMap int
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
		sm:          msg.sm,
		rm:          msg.rm,
		logchan:     make(chan *sendMsg),
		signalChan:  make(chan []byte),
		ips:         "",
		next:        nil,
		ScheduleMap: BEFORECREATE,
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
	/*
		one head only allow one node at the same time
	*/
	if head.next == nil {
		head.next = node
	} else {
		legacy := head.next
		legacy.client.addr = ""
		close(legacy.client.send)
		close(legacy.client.sendLog)

		head.next = node
	}
	/*if head.next == nil {
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
	}*/
}

// delete
func (list *SameIdsLinkList) Remove(client *Client) error {

	empty := list.isEmpty() // have node rather than only head
	if empty {
		return errors.New("this is an empty list")
	}
	head := list.Head // *headNode
	if head.next.client == client {
		head.next = nil
		Trace.Printf("[%d, %d]: %s logged out\n", client.userIds.Uid, client.userIds.Tid, client.addr)
		client.addr = ""
		close(client.send)
		close(client.sendLog)
	} else {
		Trace.Printf("[%d, %d]: %s not in the right list, or >2 client in one head\n", client.userIds.Uid, client.userIds.Tid, client.addr)
	}
	return nil
	/*if head.next.client == client {
		head.next = head.next.next
		//if client.hub.clients[*client.userIds].Head.sm.Content.StatusCode != TRAININGSTOPFAILED {
		if client.hub.clients[*client.userIds].Head.sm.Type != TRAININGSTOPFAILED {
			//close(client.send)
			//close(client.sendLog)
		}
		Trace.Printf("[%d, %d]: %s logged out\n", client.userIds.Uid, client.userIds.Tid, client.addr)
		client.addr = ""
		return nil
	} else {
		current := head.next
		for current.next != nil {
			if current.next.client == client {
				current.next = current.next.next
				//if client.hub.clients[*client.userIds].Head.sm.Content.StatusCode != TRAININGSTOPFAILED {
				if client.hub.clients[*client.userIds].Head.sm.Type != TRAININGSTOPFAILED {
					//close(client.send)
					//close(client.sendLog)
				}
				Trace.Printf("[%d, %d]: %s logged out\n", client.userIds.Uid, client.userIds.Tid, client.addr)
				client.addr = ""
				return nil
			}
			current = current.next
		}
	}*/
}

// print
func (list *SameIdsLinkList) PrintList() {
	empty := list.isEmpty()
	if empty {
		Trace.Printf("This is an empty list\n")
		return
	}
	if list.Head == nil {
		Trace.Printf("no clients in head\n")
		return
	}

	current := list.Head.next
	i := 0
	for ; ; i++ {
		if current.next == nil {
			break
		}
		Trace.Printf("[%d, %d] Client%d value:%v\n", current.client.userIds.Uid, current.client.userIds.Tid, i+1, current.client.addr)
		current = current.next
	}
	Trace.Printf("[%d, %d] Client%d value:%v\n", current.client.userIds.Uid, current.client.userIds.Tid, i+1, current.client.addr)
	return
}

func (list *SameIdsLinkList) linklistRun() {
	for true {
		select {
		case msgs := <-list.Head.logchan:
			if list.Head.next != nil {
				list.Head.next.client.sendLog <- []byte(strconv.Itoa(msgs.Type))
			}
			/*currentList := list.Head.next
			for currentList != nil {
				// sendlog cannot close or won't send to next client
				currentList.client.sendLog <- []byte(strconv.Itoa(msgs.Type))
				/*if strings.Contains(msgs.Content.Log, TRAINLOGERR) {
					close(currentList.client.sendLog)
				}
				currentList = currentList.next
			}*/
		}
	}
}
