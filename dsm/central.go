package dsm

import (
	"fmt"
	"log"
	"math"
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
	OwnerAddr  string
	AccessType int
}

type Central struct {
	// The central's name
	clients map[int]string
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

func (c *Central) HandleReadWrite(args *ReadWriteArgs, reply *ReadWriteReply) {
	fmt.Println("central handling read write on go side")
	if args.Access == 1 {
		// c.lockPage(args.Addr)
		go c.makeReadonlyOwner(args.Addr, c.owner[args.Addr].OwnerAddr)
		if _, ok := c.copyset[args.Addr]; !ok {
			c.copyset[args.Addr] = make(map[int]int)
		}
		c.copyset[args.Addr][args.ClientID] += 1 // TODO: Is the reason we use a map here because it's a "set"? yes
		reply.Err = OK
		reply.Owner = c.owner[args.Addr].OwnerAddr
		// c.locks[args.Addr].Unlock()
	} else if args.Access == 2 {
		c.invalidateCaches(args.Addr)
		// c.locks[args.Addr].Lock()
		for len(c.copyset[args.Addr]) > 0 {
			// c.locks[args.Addr].Unlock()
			// c.locks[args.Addr].Lock()
		}
		reply.Err = OK
		reply.Owner = c.owner[args.Addr].OwnerAddr
		c.owner[args.Addr] = Owner{OwnerAddr: c.clients[args.ClientID], AccessType: 2}
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
		go c.makeInvalidOwner(pageID, owner.OwnerAddr)
	}
	for clientID, _ := range copyset {
		go c.makeInvalidCopyset(pageID, clientID)
	}
}

func (c *Central) makeReadonlyOwner(addr uintptr, clientAddr string) {
	args := InvalidateArgs{Addr: addr, NewAccess: 1, ReturnPage: false}
	reply := InvalidateReply{}
	ok := call(clientAddr, "Client.ChangeAccess", &args, &reply)
	// err := c.clients[clientID].Call("Client.ChangeAccess", &args, &reply)
	for !ok {
		// Wait until expires
		ok = call(clientAddr, "Client.ChangeAccess", &args, &reply)
		// err = c.clients[clientID].Call("Client.ChangeAccess", &args, &reply)
	}
}

func (c *Central) makeInvalidOwner(addr uintptr, clientAddr string) {
	args := InvalidateArgs{Addr: addr, NewAccess: 0, ReturnPage: false}
	reply := InvalidateReply{}
	ok := call(clientAddr, "Client.ChangeAccess", &args, &reply)
	// err := c.clients[clientID].Call("Client.ChangeAccess", &args, &reply)
	for !ok {
		// Wait until expires
		ok = call(clientAddr, "Client.ChangeAccess", &args, &reply)
		// err = c.clients[clientID].Call("Client.ChangeAccess", &args, &reply)
	}
}

func (c *Central) makeInvalidCopyset(addr uintptr, clientID int) {
	args := InvalidateArgs{Addr: addr, NewAccess: 0, ReturnPage: false}
	reply := InvalidateReply{}
	ok := call(c.clients[clientID], "Client.ChangeAccess", &args, &reply)
	// err := c.clients[clientID].Call("Client.ChangeAccess", &args, &reply)
	for !ok {
		// Wait until expires
		ok = call(c.clients[clientID], "Client.ChangeAccess", &args, &reply)
		// err = c.clients[clientID].Call("Client.ChangeAccess", &args, &reply)
	}
	delete(c.copyset[addr], clientID)
}

func (c *Central) AddClient(NewClientArgs *NewClientArgs, reply *NewClientReply) {
	c.clients[NewClientArgs.Id] = NewClientArgs.Addr
	reply.Err = OK
	for _, page := range NewClientArgs.Pages {
		if _, ok := c.copyset[page]; ok {
			// page already exists - handle somehow?
		}
		c.copyset[page] = make(map[int]int)
		c.owner[page] = Owner{OwnerAddr: NewClientArgs.Addr, AccessType: 2}
	}
}

func (c *Central) initialize(clients map[int]string, numpages int) {
	c.clients = make(map[int]string)
	go c.initializeRPC()
	for id, addr := range clients {
		c.clients[id] = addr
	}
	c.owner = make(map[uintptr]Owner)

	for i := 0; i < numpages; i++ {
		id := int(math.Floor(float64(numpages) / float64(i)))
		c.owner[uintptr(i*PageSize)] = Owner{OwnerAddr: c.clients[id], AccessType: 2}
	}
	c.copyset = make(map[uintptr]map[int]int)
}

func MakeCentral(clients map[int]string, numpages int) *Central {
	c := Central{}
	c.initialize(clients, numpages)
	return &c
}

func (c *Central) initializeRPC() {
	// rpc.Register(c)
	l, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatal("listen error:", err)
	}
	defer l.Close()
	fmt.Println("central server listening on port", port)

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatal("accept error:", err)
		}
		go rpc.ServeConn(conn)
	}
}
