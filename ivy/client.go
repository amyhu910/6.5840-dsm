package ivy

import (
	"sync"
	"sync/atomic"

	"6.5840-dsm/labrpc"
)

const (
	OK = "OK"
)

type Err string

type Page struct {
	id     int
	data   []byte
	access int // 0: invalid, 1: read-only, 2: read-write
}

type Client struct {
	central   *labrpc.ClientEnd
	id        int
	pagetable map[int]Page
	locks     map[int]*sync.Mutex
	mu        sync.Mutex
	dead      int32 // for testing
}

type ReadWriteArgs struct {
	ClientID int
	PageID   int
	Access   int
}

type ReadWriteReply struct {
	Err  Err
	Data []byte
}

type AccessArgs struct {
	PageID    int
	NewAccess int
}

type AccessReply struct {
	Err  Err
	Data []byte
}

func (c *Client) Kill() {
	atomic.StoreInt32(&c.dead, 1)
	// Your code here, if desired.
}

func (c *Client) killed() bool {
	z := atomic.LoadInt32(&c.dead)
	return z == 1
}

func (c *Client) readPage(pageID int) []byte {
	// check locally first, send request to central if necessary
	c.lockPage(pageID)
	if page, ok := c.pagetable[pageID]; ok && page.access != 0 {
		defer c.locks[pageID].Unlock()
		return c.pagetable[pageID].data
	} else {
		c.locks[pageID].Unlock()
		return c.sendReadRequest(pageID)
	}
}

func (c *Client) lockPage(pageID int) {
	// Would it be better for us to group the lock into the pagetable Page instead to save some locking/unlocking of the c.mu?
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.locks[pageID]; !ok {
		c.locks[pageID] = &sync.Mutex{}
	}
	c.locks[pageID].Lock()
}

func (c *Client) sendReadRequest(pageID int) []byte {
	reply := &ReadWriteReply{}
	ok := c.central.Call("Central.handleReadWrite", &ReadWriteArgs{ClientID: c.id, PageID: pageID, Access: 1}, reply)
	if !ok {
		return nil
	}
	c.locks[pageID].Lock()
	defer c.locks[pageID].Unlock()
	if _, ok := c.pagetable[pageID]; !ok {
		c.pagetable[pageID] = Page{id: pageID, data: reply.Data, access: 1}
	} else {
		page := c.pagetable[pageID]
		page.data = reply.Data
		// TODO: Do we want to also change the page access from 0 to 1?
		c.pagetable[pageID] = page
	}
	return reply.Data
}

func (c *Client) writePage(pageID int, data []byte) {
	// check locally first for read-write permissions,
	// send request to central if necessary
	c.lockPage(pageID)
	if page, ok := c.pagetable[pageID]; ok && page.access == 2 {
		page.data = data           // Assign data to a variable before updating the struct field
		c.pagetable[pageID] = page // Update the struct field in the map
		c.locks[pageID].Unlock()
	} else {
		c.locks[pageID].Unlock()
		reply := c.sendWriteRequest(pageID)
		c.locks[pageID].Lock()
		if reply == OK {
			if _, ok := c.pagetable[pageID]; !ok {
				c.pagetable[pageID] = Page{id: pageID, data: data, access: 2}
			} else {
				page := c.pagetable[pageID]
				page.data = data
				// TODO: Probably also want to update page access here?
				c.pagetable[pageID] = page
			}
		}
		c.locks[pageID].Unlock()
	}
}

func (c *Client) sendWriteRequest(pageID int) Err {
	reply := &ReadWriteReply{}
	ok := c.central.Call("Central.handleReadWrite", &ReadWriteArgs{ClientID: c.id, PageID: pageID, Access: 2}, reply)
	if !ok {
		return "Failed to write"
	}
	return reply.Err
}

func (c *Client) ChangeAccess(args *AccessArgs, reply *AccessReply) {
	c.lockPage(args.PageID)
	defer c.locks[args.PageID].Unlock()
	if page, ok := c.pagetable[args.PageID]; ok {
		page.access = args.NewAccess
		c.pagetable[args.PageID] = page
		reply.Err = OK
		reply.Data = page.data
	} else {
		reply.Err = "Page not found"
	}
}

func (c *Client) initialize(server *labrpc.ClientEnd, me int) {
	c.central = server
	c.pagetable = make(map[int]Page)
	c.id = me
}

func MakeClient(server *labrpc.ClientEnd, me int) *Client {
	c := &Client{}
	c.initialize(server, me)
	return c
}
