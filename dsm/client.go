package dsm

/*
#cgo CFLAGS: -Wall
#include "dsm.h"
*/
import "C"

import (
	"fmt"
	"log"
	"sync/atomic"
	"syscall"
	"time"

	"net"
	"net/rpc"
)

const (
	OK = "OK"
)

var PageSize = syscall.Getpagesize()

const port = ":1234"

type Client struct {
	peers   map[int]string
	central string
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

func (c *Client) HandlePageRequest(args *PageRequestArgs, reply *PageRequestReply) {
	// lock page somehow
	fmt.Println("handling page request on go side")
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
	fmt.Println("handling read on go side")
	ownerReply := &ReadWriteReply{}
	ok := call(c.central, "Central.HandleReadWrite", &ReadWriteArgs{ClientID: c.id, Addr: addr, Access: 1}, ownerReply)
	// err := c.central.Call("Central.handleReadWrite", &ReadWriteArgs{ClientID: c.id, Addr: addr, Access: 1}, ownerReply)
	if !ok {
		return nil
	}
	pageReply := &PageRequestReply{}
	ok = call(ownerReply.Owner, "Client.HandlePageRequest", &PageRequestArgs{Addr: addr, RequestType: 1}, pageReply)
	// err = c.peers[ownerReply.Owner].Call("Client.handlePageRequest", &PageRequestArgs{Addr: addr, RequestType: 1}, pageReply)
	if !ok {
		return nil
	}
	return pageReply.Data
}

//export HandleWrite
func HandleWrite(addr C.uintptr_t) {
	client.handleWrite(uintptr(addr))
}

func (c *Client) handleWrite(addr uintptr) {
	fmt.Println("handling write on go side")
	ownerReply := &ReadWriteReply{}
	ok := call(c.central, "Central.HandleReadWrite", &ReadWriteArgs{ClientID: c.id, Addr: addr, Access: 2}, ownerReply)
	// err := c.central.Call("Central.handleReadWrite", &ReadWriteArgs{ClientID: c.id, Addr: addr, Access: 2}, ownerReply)
	if !ok {
		return
	}
	if ownerReply.Err != OK {
		return
	}
	pageReply := &PageRequestReply{}
	if ownerReply.Owner != "" {
		ok = call(ownerReply.Owner, "Client.HandlePageRequest", &PageRequestArgs{Addr: addr, RequestType: 2}, pageReply)
		// err = c.peers[ownerReply.Owner].Call("Client.handlePageRequest", &PageRequestArgs{Addr: addr, RequestType: 2}, pageReply)
	}
	if !ok {
		return
	}
	C.change_access(C.uintptr_t(addr), 2)
	C.set_page(C.uintptr_t(addr), C.CBytes(pageReply.Data))
}

func (c *Client) ChangeAccess(args *InvalidateArgs, reply *InvalidateReply) {
	// lock page somehow
	C.change_access(C.uintptr_t(args.Addr), C.int(args.NewAccess))
	if args.ReturnPage {
		reply.Data = C.GoBytes(C.get_page(C.uintptr_t(args.Addr)), C.int(PageSize))
	}
}

func call(addr string, rpcname string, args interface{}, reply interface{}) bool {
	client, err := rpc.Dial("tcp", addr+port)
	fmt.Println("dialing", addr+port)
	if err != nil {
		log.Fatal("could not connect to central", err)
	}
	defer client.Close()
	fmt.Println("connected to", addr)
	fmt.Println("calling", rpcname)
	err = client.Call(rpcname, args, reply)
	if err == nil {
		return true
	}
	return false
}

func (c *Client) initialize(peerAddr string, centralAddr string, me int) {
	c.peers = make(map[int]string)
	id := 1 - me
	c.peers[id] = peerAddr
	go c.initializeRPC()
	c.central = centralAddr
	c.id = me
}

// //export MakeClient
// func MakeClient(peers *C.char, centralAddr *C.char, me C.int) {
// 	c := &Client{}
// 	c.initialize(C.GoString(peers), C.GoString(centralAddr), int(me))
// 	client = c
// }

func MakeClient(peers string, centralAddr string, me int) {
	c := &Client{}
	c.initialize(peers, centralAddr, me)
	client = c
}

func (c *Client) initializeRPC() {
	// rpc.Register(c)
	l, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatal("listen error:", err)
	}
	defer l.Close()
	fmt.Println("client server listening on port", port)

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatal("accept error:", err)
		}
		go rpc.ServeConn(conn)
	}
}

func ClientSetup(numpages int, index int, numservers int, peer string, central string) {
	// MakeClient("localhost:8080", "localhost:8081", index)
	MakeClient(peer, central, index)

	C.setup(C.int(numpages), C.int(index), C.int(numservers), true)

	for client.killed() == false {
		time.Sleep(time.Second)
	}
	time.Sleep(time.Second)
}

func CentralSetup(clients map[int]string, numpages int) {
	central := MakeCentral(clients, numpages)

	for central.killed() == false {
		time.Sleep(time.Second)
	}
	time.Sleep(time.Second)
}
