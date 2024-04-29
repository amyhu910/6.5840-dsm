package dsm_go

/*
#cgo CFLAGS: -Wall
#include "dsm.h"
*/
import "C"
import (
	"sync/atomic"
	"syscall"
	"unsafe"

	"6.5840-dsm/labrpc"
)

const (
	OK = "OK"
)

var PageSize = syscall.Getpagesize()

type Err string

type Client struct {
	peers   []*labrpc.ClientEnd
	central *labrpc.ClientEnd
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

func convertVoidPtrToByteSlice(ptr unsafe.Pointer, length int) []byte {
	return (*[1 << 30]byte)(ptr)[:length:length]
}

func (c *Client) handlePageRequest(args *PageRequestArgs, reply *PageRequestReply) {
	// lock page somehow
	if args.RequestType == 1 {
		C.change_access(C.uintptr_t(args.Addr), 1)
	} else if args.RequestType == 2 {
		C.change_access(C.uintptr_t(args.Addr), 0)
	}
	reply.Data = C.GoBytes(C.get_page(C.uintptr_t(args.Addr)), C.int(PageSize))
}

func (c *Client) handleRead(addr uintptr) []byte {
	ownerReply := &ReadWriteReply{}
	ok := c.central.Call("Central.handleReadWrite", &ReadWriteArgs{ClientID: c.id, Addr: addr, Access: 1}, ownerReply)
	if !ok {
		return nil
	}
	pageReply := &PageRequestReply{}
	ok = c.peers[ownerReply.Owner].Call("Client.handlePageRequest", &PageRequestArgs{Addr: addr, RequestType: 1}, pageReply)
	if !ok {
		return nil
	}
	return pageReply.Data
}

func (c *Client) handleWrite(addr uintptr) {
	ownerReply := &ReadWriteReply{}
	ok := c.central.Call("Central.handleReadWrite", &ReadWriteArgs{ClientID: c.id, Addr: addr, Access: 2}, ownerReply)
	if !ok {
		return
	}
	if ownerReply.Err != OK {
		return
	}
	pageReply := &PageRequestReply{}
	if ownerReply.Owner != -1 {
		ok = c.peers[ownerReply.Owner].Call("Client.handlePageRequest", &PageRequestArgs{Addr: addr, RequestType: 2}, pageReply)
	}
	if !ok {
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

func (c *Client) initialize(peers []*labrpc.ClientEnd, server *labrpc.ClientEnd, me int) {
	c.peers = peers
	c.central = server
	c.id = me
}

func MakeClient(peers []*labrpc.ClientEnd, server *labrpc.ClientEnd, me int) *Client {
	c := &Client{}
	c.initialize(peers, server, me)
	return c
}
