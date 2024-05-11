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
	OK    = "OK"
	ERROR = "Error"
)

var PageSize = syscall.Getpagesize()

const port = ":1234"

type Client struct {
	address string
	dead    int32 // for testing
	mu      sync.Mutex
	ready   bool

	prob_owner map[uintptr]Owner
	copyset    map[uintptr]map[string]int
	owns_page  map[uintptr]bool
	locks      map[uintptr]*sync.Mutex
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
	ownerReply := &DReadWriteReply{}
	// get owner of page
	// Contact last owner
	ownerAddr := c.prob_owner[addr].OwnerAddr
	ok := call(ownerAddr, "Client.DistributedHandleReadWrite", &DReadWriteArgs{ClientAddress: c.address, Addr: addr, Access: 1}, ownerReply)
	if !ok {
		log.Println("error could not get owner of page")
	}
	C.set_page(C.uintptr_t(addr), C.CBytes(ownerReply.Data))
	C.change_access(C.uintptr_t(addr), 1)

	if ownerReply.Owner != ownerAddr {
		c.prob_owner[addr] = Owner{OwnerAddr: ownerReply.Owner, AccessType: 1}
	}

	ok = call(ownerAddr, "Client.HandleConfirmation", &ConfirmationArgs{ClientAddress: c.address, Addr: addr}, &Reply{})
}

//export HandleWrite
func HandleWrite(addr C.uintptr_t) {
	client.handleWrite(uintptr(addr))
}

func (c *Client) handleWrite(addr uintptr) {
	log.Println("handling write on go side", addr)
	ownerReply := &DReadWriteReply{}
	// Contact last owner
	ownerAddr := c.prob_owner[addr].OwnerAddr
	ok := call(ownerAddr, "Client.DistributedHandleReadWrite", &DReadWriteArgs{ClientAddress: c.address, Addr: addr, Access: 2}, ownerReply)
	if !ok {
		return
	}
	if ownerReply.Err != OK {
		return
	}
	// write to page
	C.set_page(C.uintptr_t(addr), C.CBytes(ownerReply.Data))
	ok = call(ownerAddr, "Client.HandleConfirmation", &ConfirmationArgs{ClientAddress: c.address, Addr: addr}, &Reply{})

	// Identify self as the new owner
	c.prob_owner[addr] = Owner{OwnerAddr: c.address, AccessType: 2}
	c.owns_page[addr] = true
	c.copyset[addr] = make(map[string]int)
}

func (c *Client) HandleConfirmation(args *ConfirmationArgs, reply *Reply) error {
	c.locks[args.Addr].Unlock()
	return nil
}

func (c *Client) ChangeAccess(args *InvalidateArgs, reply *InvalidateReply) error {
	if args.ReturnPage {
		log.Println("changing access on go side and returning page first", args.Addr)
		reply.Data = C.GoBytes(C.get_page(C.uintptr_t(args.Addr)), C.int(PageSize))
	}
	C.change_access(C.uintptr_t(args.Addr), C.int(args.NewAccess))
	return nil
}

func (c *Client) DistributedHandleReadWrite(args *DReadWriteArgs, reply *DReadWriteReply) error {
	if c.owns_page[args.Addr] {
		c.locks[args.Addr].Lock()
		log.Println("prob_owner", c.prob_owner[args.Addr])
		log.Println("copyset", c.copyset)
		if args.Access == 1 {
			log.Println("client handling read on go side", args.Addr, args.ClientAddress)
			// make owner readonly
			if _, ok := c.copyset[args.Addr]; !ok {
				c.copyset[args.Addr] = make(map[string]int)
			}
			reply.Data = c.makeReadonlyOwner(args.Addr)
			if reply.Data != nil {
				reply.Err = OK
			} else {
				reply.Err = ERROR
			}
			c.copyset[args.Addr][args.ClientAddress] = 1
			c.owns_page[args.Addr] = true // TODO: Prob redundant, should already be true
			reply.Err = OK
			reply.Owner = c.address
		} else if args.Access == 2 {
			log.Println("client handling write on go side", args.Addr, args.ClientAddress)
			// invalidate all pages and return data
			delete(c.copyset[args.Addr], args.ClientAddress)
			reply.Data = c.invalidateCaches(args.Addr, args.ClientAddress)
			reply.Owner = c.address
			// wait for invalidation to finish
			for len(c.copyset[args.Addr]) > 0 {
			}
			reply.Err = OK
		}
		log.Println("done handling")
		return nil
	} else {
		// Forwarding logic here
		log.Println("Client %v is not the owner. Forwarding to %v\n", c.address, c.prob_owner[args.Addr].OwnerAddr)
		prob_owner := c.prob_owner[args.Addr]
		ok := call(prob_owner.OwnerAddr, "Client.DistributedHandleReadWrite", args, reply)
		if !ok {
			log.Println("Issue forwarding.")
		}
		if reply.Owner != prob_owner.OwnerAddr {
			if args.Access == 1 {
				c.prob_owner[args.Addr] = Owner{OwnerAddr: reply.Owner, AccessType: 1}
			} else {
				c.prob_owner[args.Addr] = Owner{OwnerAddr: args.ClientAddress, AccessType: 2}
			}
		}
	}
	return nil
}

func (c *Client) makeReadonlyOwner(addr uintptr) []byte {
	log.Println("making myself(%v) into readonly owner", c.address)
	args := InvalidateArgs{Addr: addr, NewAccess: 1, ReturnPage: true}
	reply := InvalidateReply{}
	c.ChangeAccess(&args, &reply)
	c.prob_owner[addr] = Owner{OwnerAddr: c.address, AccessType: 1}
	return reply.Data
}

func (c *Client) invalidateCaches(pageID uintptr, requestAddress string) []byte {
	log.Println("Invalidating Caches")
	copyset, ok := c.copyset[pageID]
	if ok {
		for clientAddr, _ := range copyset {
			if clientAddr == requestAddress {
				continue
			}
			log.Println(clientAddr)
			go c.makeInvalidCopyset(pageID, clientAddr)
		}
	}
	// Make the owner, or the current client invalid
	c.prob_owner[pageID] = Owner{OwnerAddr: requestAddress, AccessType: 2}
	c.owns_page[pageID] = false
	return nil
}

func (c *Client) makeInvalidCopyset(addr uintptr, clientAddr string) {
	log.Println("make invalid copyset", clientAddr)
	args := InvalidateArgs{Addr: addr, NewAccess: 0, ReturnPage: false}
	reply := InvalidateReply{}
	ok := false
	for !ok {
		// Wait until expires
		ok = call(clientAddr, "Client.ChangeAccess", &args, &reply)
	}
	delete(c.copyset[addr], clientAddr)
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

func (c *Client) initialize(numpages int, address string) {
	go c.initializeRPC()
	c.address = address
	c.prob_owner = make(map[uintptr]Owner)
	c.owns_page = make(map[uintptr]bool)
	c.locks = make(map[uintptr]*sync.Mutex)
	c.copyset = make(map[uintptr]map[string]int)
	for i := 0; i < numpages; i++ {
		c.prob_owner[uintptr(i*PageSize)] = Owner{OwnerAddr: default_owner_address, AccessType: 2}
		c.owns_page[uintptr(i*PageSize)] = c.address == default_owner_address
		c.locks[uintptr(i*PageSize)] = &sync.Mutex{}
	}
}

func MakeClient(numpages int, address string) {
	c := &Client{}
	c.initialize(numpages, address)
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

func ClientSetup(numpages int, index int, address string, numservers int) {
	MakeClient(numpages, address)

	C.setup(C.int(numpages), C.int(index), C.int(numservers))
	C.test_one_client(C.int(numpages), C.int(index), C.int(numservers))
	/*
		for client.killed() == false {
			time.Sleep(time.Second)
			if client.ready {
				C.test_concurrent_clients(C.int(numpages), C.int(index), C.int(numservers))
				break
			}
		}

		C.setup_matmul(C.int(numpages), C.int(index), C.int(numservers))

		for client.killed() == false {
			time.Sleep(time.Second)
			if client.ready {
				C.multiply_matrices(C.int(index), C.int(numservers))
				break
			}
		}
	*/
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
