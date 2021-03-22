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
	listCount   int
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
		listCount:   0,
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
		one head only allow one node at the same time, except admin
	*/
	if head.next == nil {
		head.next = node
		head.listCount++
	} else {
		if head.listCount == 1 {
			if node.client.Admin == head.next.client.Admin {
				// replace one different
				head.next.client.addr = ""
				close(head.next.client.send)
				close(head.next.client.sendLog)
				head.next = node
			} else {
				head.next.next = node
				head.listCount++
			}
		} else if head.listCount == 2 {
			if node.client.Admin == head.next.client.Admin {
				head.next.client.addr = ""
				close(head.next.client.send)
				close(head.next.client.sendLog)
				head.next = node
			} else if node.client.Admin == head.next.next.client.Admin {
				head.next.next.client.addr = ""
				close(head.next.next.client.send)
				close(head.next.next.client.sendLog)
				head.next.next = node
			}
		} else {
			Error.Printf("[%d, %d]: have >2 clients node\n", head.rm.Content.IDs.Uid, head.rm.Content.IDs.Tid)
			return
		}
	}
}

// delete
func (list *SameIdsLinkList) Remove(client *Client) error {

	empty := list.isEmpty() // have node rather than only head
	if empty {
		return errors.New("this is an empty list")
	}
	head := list.Head // *headNode

	if head.listCount == 1 {
		if head.next.client == client {
			last := head.next.next // node or nil
			client.addr = ""
			close(client.send)
			close(client.sendLog)
			head.next = last
			head.listCount--
			Trace.Printf("[%d, %d](a:%v): %s logged out\n", client.userIds.Uid, client.userIds.Tid, client.Admin, client.addr)
		}
	} else if head.listCount == 2 {
		if head.next.client == client {
			client.addr = ""
			close(client.send)
			close(client.sendLog)
			head.next = head.next.next
			head.listCount--
			Trace.Printf("[%d, %d](a:%v): %s logged out\n", client.userIds.Uid, client.userIds.Tid, client.Admin, client.addr)
		} else if head.next.next.client == client {
			client.addr = ""
			close(client.send)
			close(client.sendLog)
			head.next.next = nil
			head.listCount--
			Trace.Printf("[%d, %d](a:%v): %s logged out\n", client.userIds.Uid, client.userIds.Tid, client.Admin, client.addr)
		}
	} else {
		Trace.Printf("[%d, %d]: %s not in the right list, or >2 client in one head\n", client.userIds.Uid, client.userIds.Tid, client.addr)
	}
	list.PrintList()
	return nil
}

// print
func (list *SameIdsLinkList) PrintList() {
	empty := list.isEmpty()
	if empty {
		Trace.Printf("[%d, %d]:This is an empty list\n", list.Head.rm.Content.IDs.Uid, list.Head.rm.Content.IDs.Tid)
		return
	}
	if list.Head == nil {
		Trace.Printf("no clients in head\n")
		return
	}

	current := list.Head.next
	if list.Head.listCount > 2 {
		Error.Printf("[%d, %d]: already have two client node\n", list.Head.rm.Content.IDs.Uid, list.Head.rm.Content.IDs.Tid)
		return
	}
	Trace.Printf("----------------[%d, %d]----------------\n", list.Head.rm.Content.IDs.Uid, list.Head.rm.Content.IDs.Tid)
	for i := 0; i < list.Head.listCount; i++ {
		Trace.Printf("[%d, %d](a:%v): %s exists\n", current.client.userIds.Uid, current.client.userIds.Tid, current.client.Admin, current.client.addr)
		current = current.next
	}
	Trace.Printf("----------------[%d, %d]----------------\n", list.Head.rm.Content.IDs.Uid, list.Head.rm.Content.IDs.Tid)
}

func (list *SameIdsLinkList) linklistRun() {
	for true {
		select {
		case msgs := <-list.Head.logchan:
			if list.Head.listCount == 1 {
				if !(list.Head.next.client.Admin) {
					list.Head.next.client.sendLog <- []byte(strconv.Itoa(msgs.Type))
				}
			} else if list.Head.listCount == 2 {
				if !(list.Head.next.client.Admin) {
					list.Head.next.client.sendLog <- []byte(strconv.Itoa(msgs.Type))
				} else if !(list.Head.next.next.client.Admin) {
					list.Head.next.next.client.sendLog <- []byte(strconv.Itoa(msgs.Type))
				}
			}
		}
	}
}
