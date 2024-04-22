package ivy

import (
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

func (c *Client) readPage(pageID int) []byte {
	// check locally first, send request to central if necessary
	if page, ok := c.pagetable[pageID]; ok && page.access == 0 {
		return c.pagetable[pageID].data
	} else {
		return c.sendReadRequest(pageID)
	}
}

func (c *Client) sendReadRequest(pageID int) []byte {
	reply := &ReadWriteReply{}
	ok := c.central.Call("Central.handleReadWrite", &ReadWriteArgs{ClientID: c.id, PageID: pageID, Access: 1}, reply)
	if !ok {
		return nil
	}
	return reply.Data
}

func (c *Client) writePage(pageID int, data []byte) {
	// check locally first for read-write permissions,
	// send request to central if necessary
	if page, ok := c.pagetable[pageID]; ok && page.access == 2 {
		page.data = data           // Assign data to a variable before updating the struct field
		c.pagetable[pageID] = page // Update the struct field in the map
	} else {
		reply := c.sendWriteRequest(pageID)
		if reply == OK {
			c.pagetable[pageID] = Page{id: pageID, data: data, access: 2}
		}
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
