package ivy

import (
	"sync"
	"sync/atomic"
	"time"

	"6.5840-dsm/labrpc"
)

const (
	LeaseDuration = 10 * time.Second
)

type Lease struct {
	Owner int
	Valid bool
	Start time.Time
}

type Owner struct {
	OwnerID    int
	AccessType int
	Lease      Lease
}

type Central struct {
	// The central's name
	clients map[int]*labrpc.ClientEnd
	copyset map[int]map[int]int
	owner   map[int]Owner
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
		c.lockPage(args.PageID)
		if _, ok := c.copyset[args.PageID]; !ok {
			c.copyset[args.PageID] = make(map[int]int)
		}
		c.copyset[args.PageID][args.ClientID] += 1 // TODO: Is the reason we use a map here because it's a "set"? yes
		reply.Err = OK
		reply.Owner = c.owner[args.PageID].OwnerID
		c.locks[args.PageID].Unlock()
	} else if args.Access == 2 {
		c.invalidateCaches(args.PageID)
		c.locks[args.PageID].Lock()
		for len(c.copyset[args.PageID]) > 0 || time.Now().Before(c.owner[args.PageID].Lease.Start.Add(LeaseDuration)) {
			c.locks[args.PageID].Unlock()
			c.locks[args.PageID].Lock()
		}
		reply.Err = OK
		reply.Owner = c.owner[args.PageID].OwnerID
		newLease := Lease{Owner: args.ClientID, Valid: true, Start: time.Now()}
		c.owner[args.PageID] = Owner{OwnerID: args.ClientID, AccessType: 2, Lease: newLease}
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
	if owner, ok := c.owner[pageID]; ok && owner.Lease.Valid {
		go c.makeInvalid(pageID, owner.OwnerID)
	}
	for clientID, _ := range copyset {
		go c.makeInvalid(pageID, clientID)
	}
}

func (c *Central) makeInvalid(pageID int, clientID int) {
	args := InvalidateArgs{PageID: pageID, NewAccess: 0, ReturnPage: false}
	reply := InvalidateReply{}
	ok := c.clients[clientID].Call("Client.ChangeAccess", &args, &reply)
	for !ok {
		// Wait until expires
		ok = c.clients[clientID].Call("Client.ChangeAccess", &args, &reply)
	}
	c.lockPage(pageID)
	delete(c.copyset[pageID], clientID)
	c.locks[pageID].Unlock()
}

func (c *Central) initialize() {
	c.clients = make(map[int]*labrpc.ClientEnd)
	c.copyset = make(map[int]map[int]int)
	c.owner = make(map[int]Owner)
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
