package dsm

import (
	"log"
	"net"
	"net/rpc"
	"sync/atomic"
)

// const (
// 	LeaseDuration = 10 * time.Second
// )

// type Lease struct {
// 	Owner int
// 	Valid bool
// 	Start time.Time
// }

type Owner struct {
	OwnerID    int
	AccessType int
}

type Central struct {
	// The central's name
	clients map[int]*rpc.Client
	copyset map[uintptr]map[int]int
	owner   map[uintptr]Owner
	// locks   map[int]*sync.Mutex //I think we don't need to use references for mutexes: https://www.reddit.com/r/golang/comments/u9o5wj/mutex_struct_field_as_reference_or_value/
	// mu      sync.Mutex
	dead int32 // for testing
}

func (c *Central) Kill() {
	atomic.StoreInt32(&c.dead, 1)
	// Your code here, if desired.
}

func (c *Central) killed() bool {
	z := atomic.LoadInt32(&c.dead)
	return z == 1
}

// func (c *Central) lockPage(pageID int) {
// 	c.mu.Lock()
// 	defer c.mu.Unlock()
// 	if _, ok := c.locks[pageID]; !ok {
// 		c.locks[pageID] = &sync.Mutex{}
// 	}
// 	c.locks[pageID].Lock()
// }

func (c *Central) handleReadWrite(args *ReadWriteArgs, reply *ReadWriteReply) {
	if args.Access == 1 {
		// c.lockPage(args.Addr)
		if _, ok := c.copyset[args.Addr]; !ok {
			c.copyset[args.Addr] = make(map[int]int)
		}
		c.copyset[args.Addr][args.ClientID] += 1 // TODO: Is the reason we use a map here because it's a "set"? yes
		reply.Err = OK
		reply.Owner = c.owner[args.Addr].OwnerID
		// c.locks[args.Addr].Unlock()
	} else if args.Access == 2 {
		c.invalidateCaches(args.Addr)
		// c.locks[args.Addr].Lock()
		for len(c.copyset[args.Addr]) > 0 {
			// c.locks[args.Addr].Unlock()
			// c.locks[args.Addr].Lock()
		}
		reply.Err = OK
		reply.Owner = c.owner[args.Addr].OwnerID
		c.owner[args.Addr] = Owner{OwnerID: args.ClientID, AccessType: 2}
		// c.locks[args.Addr].Unlock()
	}
}

func (c *Central) invalidateCaches(pageID uintptr) {
	// c.lockPage(pageID)
	// defer c.locks[pageID].Unlock()
	copyset, ok := c.copyset[pageID]
	if !ok || len(copyset) == 0 {
		return
	}
	if owner, ok := c.owner[pageID]; ok {
		go c.makeInvalid(pageID, owner.OwnerID)
	}
	for clientID, _ := range copyset {
		go c.makeInvalid(pageID, clientID)
	}
}

func (c *Central) makeInvalid(addr uintptr, clientID int) {
	args := InvalidateArgs{Addr: addr, NewAccess: 0, ReturnPage: false}
	reply := InvalidateReply{}
	err := c.clients[clientID].Call("Client.ChangeAccess", &args, &reply)
	for err != nil {
		// Wait until expires
		err = c.clients[clientID].Call("Client.ChangeAccess", &args, &reply)
	}
	delete(c.copyset[addr], clientID)
}

func (c *Central) initialize(clients map[int]string) {
	c.clients = make(map[int]*rpc.Client)
	go c.initializeRPC()
	for id, addr := range clients {
		peer, err := rpc.Dial("tcp", addr)
		if err != nil {
			log.Fatal("could not connect to peer", err)
		}
		c.clients[id] = peer
	}
	c.copyset = make(map[uintptr]map[int]int)
	c.owner = make(map[uintptr]Owner)
}

func MakeCentral(clients map[int]string) *Central {
	c := &Central{}
	c.initialize(clients)
	return c
}

func (c *Central) initializeRPC() {
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