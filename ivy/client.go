package ivy

import (
	"sync"
	"sync/atomic"
	"time"

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
	lease  Lease
}

type Client struct {
	peers     []*labrpc.ClientEnd
	central   *labrpc.ClientEnd
	id        int
	pagetable map[int]Page
	locks     map[int]*sync.Mutex
	mu        sync.Mutex
	dead      int32 // for testing
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

func (c *Client) handlePageRequest(args *PageRequestArgs, reply *PageRequestReply) {
	c.lockPage(args.PageID)
	defer c.locks[args.PageID].Unlock()
	if args.RequestType == 1 {
		page := c.pagetable[args.PageID]
		page.access = 1
		c.pagetable[args.PageID] = page
	} else if args.RequestType == 2 {
		page := c.pagetable[args.PageID]
		page.access = 0
		c.pagetable[args.PageID] = page
	}
	if page, ok := c.pagetable[args.PageID]; ok {
		reply.Err = OK
		reply.Data = page.data
	} else {
		reply.Err = "Page not found"
	}
}

func (c *Client) sendReadRequest(pageID int) []byte {
	ownerReply := &ReadWriteReply{}
	ok := c.central.Call("Central.handleReadWrite", &ReadWriteArgs{ClientID: c.id, PageID: pageID, Access: 1}, ownerReply)
	if !ok {
		return nil
	}
	pageReply := &PageRequestReply{}
	ok = c.peers[ownerReply.Owner].Call("Client.handlePageRequest", &PageRequestArgs{PageID: pageID, RequestType: 1}, pageReply)
	if !ok {
		return nil
	}
	c.locks[pageID].Lock()
	defer c.locks[pageID].Unlock()
	if _, ok := c.pagetable[pageID]; !ok {
		c.pagetable[pageID] = Page{id: pageID, data: pageReply.Data, access: 1}
	} else {
		page := c.pagetable[pageID]
		page.data = pageReply.Data
		page.access = 1
		c.pagetable[pageID] = page
	}
	return pageReply.Data
}

func (c *Client) monitorLease(pageID int) {
	expiry := c.pagetable[pageID].lease.Start.Add(LeaseDuration)
	for time.Now().Before(expiry) {
		if c.pagetable[pageID].access != 2 {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	page := c.pagetable[pageID]
	page.access = 1
	c.pagetable[pageID] = page
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
		ownerReply := &ReadWriteReply{}
		ok := c.central.Call("Central.handleReadWrite", &ReadWriteArgs{ClientID: c.id, PageID: pageID, Access: 2}, ownerReply)
		if !ok {
			return
		}
		c.locks[pageID].Lock()
		defer c.locks[pageID].Unlock()
		if ownerReply.Err != OK {
			return
		}
		pageReply := &PageRequestReply{}
		if ownerReply.Owner != -1 {
			ok = c.peers[ownerReply.Owner].Call("Client.handlePageRequest", &PageRequestArgs{PageID: pageID, RequestType: 2}, pageReply)
		}
		if _, ok := c.pagetable[pageID]; !ok {
			c.pagetable[pageID] = Page{id: pageID, data: pageReply.Data, access: 2, lease: ownerReply.Lease}
		} else {
			page := c.pagetable[pageID]
			page.data = pageReply.Data
			page.access = 2
			page.lease = ownerReply.Lease
			c.pagetable[pageID] = page
		}
		go c.monitorLease(pageID)
	}
}

func (c *Client) ChangeAccess(args *InvalidateArgs, reply *InvalidateReply) {
	c.lockPage(args.PageID)
	defer c.locks[args.PageID].Unlock()
	if page, ok := c.pagetable[args.PageID]; ok {
		page.access = args.NewAccess
		c.pagetable[args.PageID] = page
		reply.Err = OK
		if args.ReturnPage {
			reply.Data = page.data
		}
	} else {
		reply.Err = "Page not found"
	}
}

func (c *Client) initialize(server *labrpc.ClientEnd, me int) {
	c.central = server
	c.pagetable = make(map[int]Page)
	c.id = me
}

func MakeClient(peers []*labrpc.ClientEnd, server *labrpc.ClientEnd, me int) *Client {
	c := &Client{}
	c.initialize(server, me)
	return c
}
