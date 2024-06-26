package dsm

/*
#cgo CFLAGS: -Wall
#include "dsm.h"
*/
import "C"

import (
	"fmt"
	"log"
	"sync"
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
	central string
	id      int
	dead    int32 // for testing
	mu      sync.Mutex
	ready   bool
}

func (c *Client) Kill() {
	atomic.StoreInt32(&c.dead, 1)
	// Your code here, if desired.
}

func (c *Client) killed() bool {
	z := atomic.LoadInt32(&c.dead)
	return z == 1
}

func (c *Client) AllClientsRegistered(args *Args, reply *Reply) error {
	log.Println("all clients registered")
	c.ready = true
	return nil
}

var client *Client

func (c *Client) HandlePageRequest(args *PageRequestArgs, reply *PageRequestReply) error {
	log.Println("handling page request on go side", args.Addr)
	reply.Data = C.GoBytes(C.get_page(C.uintptr_t(args.Addr)), C.int(PageSize))
	return nil
}

//export HandleRead
func HandleRead(addr C.uintptr_t) {
	client.handleRead(uintptr(addr))
}

func (c *Client) handleRead(addr uintptr) {
	log.Println("handling read on go side", addr)
	ownerReply := &ReadWriteReply{}
	// get owner of page
	ok := call(c.central, "Central.HandleReadWrite", &ReadWriteArgs{ClientID: c.id, Addr: addr, Access: 1}, ownerReply)
	if !ok {
		log.Println("error could not get owner of page")
	}
	if ownerReply.HadOwner {
		pageReply := &PageRequestReply{}
		// get page data
		ok = call(ownerReply.Owner, "Client.HandlePageRequest", &PageRequestArgs{Addr: addr, RequestType: 1}, pageReply)
		if !ok {
			log.Println("error could not get page data")
		}
		// write to page
		C.set_page(C.uintptr_t(addr), C.CBytes(pageReply.Data))
	} else {
		var empty_page []byte
		C.set_page(C.uintptr_t(addr), C.CBytes(empty_page))
	}
	C.change_access(C.uintptr_t(addr), 1)

	ok = call(c.central, "Central.HandleConfirmation", &ConfirmationArgs{ClientID: c.id, Addr: addr}, &Reply{})
}

//export HandleWrite
func HandleWrite(addr C.uintptr_t) {
	client.handleWrite(uintptr(addr))
}

func (c *Client) handleWrite(addr uintptr) {
	log.Println("handling write on go side", addr)
	ownerReply := &ReadWriteReply{}
	// invalidate caches and load page
	ok := call(c.central, "Central.HandleReadWrite", &ReadWriteArgs{ClientID: c.id, Addr: addr, Access: 2}, ownerReply)
	if !ok {
		return
	}
	if ownerReply.Err != OK {
		return
	}
	// write to page
	C.set_page(C.uintptr_t(addr), C.CBytes(ownerReply.Data))
	ok = call(c.central, "Central.HandleConfirmation", &ConfirmationArgs{ClientID: c.id, Addr: addr}, &Reply{})
}

func (c *Client) ChangeAccess(args *InvalidateArgs, reply *InvalidateReply) error {
	if args.ReturnPage {
		log.Println("changing access on go side and returning page first", args.Addr)
		reply.Data = C.GoBytes(C.get_page(C.uintptr_t(args.Addr)), C.int(PageSize))
	}
	C.change_access(C.uintptr_t(args.Addr), C.int(args.NewAccess))
	return nil
}

func call(addr string, rpcname string, args interface{}, reply interface{}) bool {
	if addr == "" {
		log.Println("invalid address")
		for {
		}
	}
	client, err := rpc.Dial("tcp", addr+port)
	log.Println("dialing", addr+port)
	if err != nil {
		log.Fatal(fmt.Sprintf("could not connect to %v", addr), err)
	}
	defer client.Close()
	log.Println("connected to", addr)
	log.Println("calling", rpcname)
	err = client.Call(rpcname, args, reply)
	if err == nil {
		return true
	}
	return false
}

func (c *Client) initialize(centralAddr string, me int) {
	go c.initializeRPC()
	c.central = centralAddr
	c.id = me
	c.mu = sync.Mutex{}
	reply := &RegisterReply{}
	ok := call(c.central, "Central.RegisterClient", &RegisterArgs{ClientID: c.id}, reply)
	if !ok {
		log.Println("error could not register client")
	}
}

func MakeClient(centralAddr string, me int) {
	c := &Client{}
	c.initialize(centralAddr, me)
	client = c
}

func (c *Client) initializeRPC() {
	rpc.Register(c)
	l, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatal("listen error:", err)
	}
	defer l.Close()
	log.Println("client server listening on port", port)

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatal("accept error:", err)
		}
		go rpc.ServeConn(conn)
	}
}

func ClientSetup(numpages int, index int, numservers int, central string) {
	MakeClient(central, index)

	// C.setup(C.int(numpages), C.int(index), C.int(numservers))
	// C.test_one_client(C.int(numpages), C.int(index), C.int(numservers))

	// for client.killed() == false {
	// 	time.Sleep(time.Second)
	// 	if client.ready {
	// 		C.test_concurrent_clients(C.int(numpages), C.int(index), C.int(numservers))
	// 		break
	// 	}
	// }

	C.setup_matmul(C.int(numpages), C.int(index), C.int(numservers))

	for client.killed() == false {
		time.Sleep(time.Second)
		if client.ready {
			C.multiply_matrices(C.int(index), C.int(numservers))
			break
		}
	}
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
