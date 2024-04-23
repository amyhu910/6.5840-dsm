package ivy

import (
	"sync"
	"sync/atomic"

	"6.5840-dsm/labrpc"
)

type Central struct {
	// The central's name
	clients map[int]*labrpc.ClientEnd
	copyset map[int]map[int]int
	owner   map[int]int
	locks   map[int]*sync.Mutex //I think we don't need to use references for mutexes: https://www.reddit.com/r/golang/comments/u9o5wj/mutex_struct_field_as_reference_or_value/
	mu      sync.Mutex
	dead    int32 // for testing
}

func (c *Central) Kill() {
	atomic.StoreInt32(&c.dead, 1)
	// Your code here, if desired.
}

func (c *Central) killed() bool {
	z := atomic.LoadInt32(&c.dead)
	return z == 1
}

func (c *Central) lockPage(pageID int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.locks[pageID]; !ok {
		c.locks[pageID] = &sync.Mutex{}
	}
	c.locks[pageID].Lock()
}

func (c *Central) handleReadWrite(args *ReadWriteArgs, reply *ReadWriteReply) {
	if args.Access == 1 {
		data := c.makeReadOnly(args.PageID)
		c.lockPage(args.PageID)
		if _, ok := c.copyset[args.PageID]; !ok {
			c.copyset[args.PageID] = make(map[int]int)
		}
		c.copyset[args.PageID][args.ClientID] = 1 // TODO: Is the reason we use a map here because it's a "set"?
		reply.Err = OK
		reply.Data = data
		c.locks[args.PageID].Unlock()
	} else if args.Access == 2 {
		c.invalidateCaches(args.PageID)
		c.locks[args.PageID].Lock()
		for len(c.copyset[args.PageID]) > 0 {
			c.locks[args.PageID].Unlock()
			c.locks[args.PageID].Lock()
		}
		c.owner[args.PageID] = args.ClientID
		c.locks[args.PageID].Unlock()
	}
}

func (c *Central) invalidateCaches(pageID int) {
	c.lockPage(pageID)
	defer c.locks[pageID].Unlock()
	copyset, ok := c.copyset[pageID]
	if !ok || len(copyset) == 0 {
		return
	}

	ownerID, hasOwner := c.owner[pageID]
	if hasOwner && ownerID != -1 {
		go c.makeInvalid(pageID, ownerID)
	}
	for clientID, _ := range copyset {
		go c.makeInvalid(pageID, clientID)
	}
}

func (c *Central) makeInvalid(pageID int, clientID int) {
	args := AccessArgs{PageID: pageID, NewAccess: 0}
	reply := AccessReply{}
	ok := c.clients[clientID].Call("Client.ChangeAccess", &args, &reply)
	for !ok {
		// Wait until expires
		ok = c.clients[clientID].Call("Client.ChangeAccess", &args, &reply)
	}
	c.lockPage(pageID)
	delete(c.copyset[pageID], clientID)
	c.locks[pageID].Unlock()
}

func (c *Central) makeReadOnly(pageID int) []byte {
	c.lockPage(pageID)
	clientID, hasOwner := c.owner[pageID]
	if !hasOwner || clientID == -1 {
		return nil
	}
	args := AccessArgs{PageID: pageID, NewAccess: 1}
	reply := AccessReply{}
	c.locks[pageID].Unlock()
	ok := c.clients[clientID].Call("Client.ChangeAccess", &args, &reply)
	for !ok {
		// Modify to account for leases
		ok = c.clients[clientID].Call("Client.ChangeAccess", &args, &reply)
	}
	c.lockPage(pageID)
	defer c.locks[pageID].Unlock()
	c.owner[pageID] = -1
	// Add owner to the copyset
	return reply.Data
}

func (c *Central) initialize() {
	c.clients = make(map[int]*labrpc.ClientEnd)
	c.copyset = make(map[int]map[int]int)
	c.owner = make(map[int]int)
}

func (c *Central) registerClient(client *labrpc.ClientEnd, clientID int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.clients[clientID] = client
}

func (c *Central) unregisterClient(clientID int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.clients, clientID)
}

func MakeCentral() *Central {
	c := &Central{}
	c.initialize()
	labrpc.MakeService(c)
	return c
}
