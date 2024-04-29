package main

/*
#cgo CFLAGS: -Wall
#include "dsm.h"
*/
import "C"

import (
	"log"
	"sync/atomic"
	"syscall"

	"net"
	"net/rpc"
)

const (
	OK = "OK"
)

var PageSize = syscall.Getpagesize()

type Err string

type Client struct {
	peers   map[int]*rpc.Client
	central *rpc.Client
	id      int
	dead    int32 // for testing
}

func (c *Client) Kill() {
	atomic.StoreInt32(&c.dead, 1)
	// Your code here, if desired.
}

func (c *Client) killed() bool {
	z := atomic.LoadInt32(&c.dead)
	return z == 1
}

var client *Client

func (c *Client) handlePageRequest(args *PageRequestArgs, reply *PageRequestReply) {
	// lock page somehow
	if args.RequestType == 1 {
		C.change_access(C.uintptr_t(args.Addr), 1)
	} else if args.RequestType == 2 {
		C.change_access(C.uintptr_t(args.Addr), 0)
	}
	reply.Data = C.GoBytes(C.get_page(C.uintptr_t(args.Addr)), C.int(PageSize))
}

//export HandleRead
func HandleRead(addr C.uintptr_t) *C.char {
	result := client.handleRead(uintptr(addr))
	if result == nil {
		return nil
	}
	return (*C.char)(C.CBytes(result))
}

func (c *Client) handleRead(addr uintptr) []byte {
	ownerReply := &ReadWriteReply{}
	err := c.central.Call("Central.handleReadWrite", &ReadWriteArgs{ClientID: c.id, Addr: addr, Access: 1}, ownerReply)
	if err != nil {
		return nil
	}
	pageReply := &PageRequestReply{}
	err = c.peers[ownerReply.Owner].Call("Client.handlePageRequest", &PageRequestArgs{Addr: addr, RequestType: 1}, pageReply)
	if err != nil {
		return nil
	}
	return pageReply.Data
}

//export HandleWrite
func HandleWrite(addr C.uintptr_t) {
	client.handleWrite(uintptr(addr))
}

func (c *Client) handleWrite(addr uintptr) {
	ownerReply := &ReadWriteReply{}
	err := c.central.Call("Central.handleReadWrite", &ReadWriteArgs{ClientID: c.id, Addr: addr, Access: 2}, ownerReply)
	if err != nil {
		return
	}
	if ownerReply.Err != OK {
		return
	}
	pageReply := &PageRequestReply{}
	if ownerReply.Owner != -1 {
		err = c.peers[ownerReply.Owner].Call("Client.handlePageRequest", &PageRequestArgs{Addr: addr, RequestType: 2}, pageReply)
	}
	if err != nil {
		return
	}
	C.change_access(C.uintptr_t(addr), 2)
	C.set_page(C.uintptr_t(addr), C.CBytes(pageReply.Data))
}

func (c *Client) changeAccess(args *InvalidateArgs, reply *InvalidateReply) {
	// lock page somehow
	C.change_access(C.uintptr_t(args.Addr), C.int(args.NewAccess))
	if args.ReturnPage {
		reply.Data = C.GoBytes(C.get_page(C.uintptr_t(args.Addr)), C.int(PageSize))
	}
}

func (c *Client) initialize(peerAddr string, centralAddr string, me int) {
	c.peers = make(map[int]*rpc.Client)
	id := 0
	peer, err := rpc.Dial("tcp", peerAddr)
	if err != nil {
		log.Fatal("could not connect to peer", err)
	}
	c.peers[id] = peer
	central, err := rpc.Dial("tcp", centralAddr)
	if err != nil {
		log.Fatal("could not connect to central", err)
	}
	c.central = central
	c.id = me
	go c.initializeRPC()
}

//export MakeClient
func MakeClient(peers *C.char, centralAddr *C.char, me C.int) {
	c := &Client{}
	c.initialize(C.GoString(peers), C.GoString(centralAddr), int(me))
	client = c
}

func (c *Client) initializeRPC() {
	rpc.Register(c)
	l, err := net.Listen("tcp", ":1234")
	if err != nil {
		log.Fatal("listen error:", err)
	}
	defer l.Close()

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatal("accept error:", err)
		}
		go rpc.ServeConn(conn)
	}
}

func main() {}
