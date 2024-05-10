package dsm

import (
	"log"
	"net"
	"net/rpc"
	"sync"
	"sync/atomic"
)

type Owner struct {
	OwnerAddr  string
	AccessType int
}

type Central struct {
	// The central's name
	num_clients int
	register    map[int]bool
	clients     map[int]string
	copyset     map[uintptr]map[int]int
	owner       map[uintptr]Owner
	locks       map[uintptr]*sync.Mutex
	dead        int32 // for testing
}

func (c *Central) Kill() {
	atomic.StoreInt32(&c.dead, 1)
	// Your code here, if desired.
}

func (c *Central) killed() bool {
	z := atomic.LoadInt32(&c.dead)
	return z == 1
}

func (c *Central) RegisterClient(args *RegisterArgs, reply *RegisterReply) error {
	c.register[args.ClientID] = true
	c.num_clients++
	if c.num_clients == len(c.clients) {
		// all clients have registered
		go c.allClientsRegistered()
	}
	reply.Err = OK
	return nil
}

func (c *Central) allClientsRegistered() {
	for id, _ := range c.clients {
		call(c.clients[id], "Client.AllClientsRegistered", Args{}, Reply{})
	}
}

func (c *Central) HandleConfirmation(args *ConfirmationArgs, reply *Reply) error {
	c.locks[args.Addr].Unlock()
	return nil
}

func (c *Central) HandleReadWrite(args *ReadWriteArgs, reply *ReadWriteReply) error {
	// Handle no owner starting state
	// Handle safety checks for no invalidating ourselves
	c.locks[args.Addr].Lock()
	log.Println("owner", c.owner)
	log.Println("copyset", c.copyset)
	if args.Access == 1 {
		log.Println("central handling read on go side", args.Addr, c.clients[args.ClientID])
		// make owner readonly
		pageOwner, found := c.owner[args.Addr]
		if _, ok := c.copyset[args.Addr]; !ok {
			c.copyset[args.Addr] = make(map[int]int)
		}
		if found {
			c.makeReadonlyOwner(args.Addr, pageOwner.OwnerAddr)
			// update copyset
			c.copyset[args.Addr][args.ClientID] = 1
			reply.HadOwner = true
		} else {
			c.owner[args.Addr] = Owner{OwnerAddr: c.clients[args.ClientID], AccessType: 1}
			reply.HadOwner = false
		}
		reply.Err = OK
		reply.Owner = c.owner[args.Addr].OwnerAddr
	} else if args.Access == 2 {
		log.Println("central handling write on go side", args.Addr, c.clients[args.ClientID])
		// invalidate all pages and return data
		delete(c.copyset[args.Addr], args.ClientID)
		reply.Data = c.invalidateCaches(args.Addr, args.ClientID)
		// wait for invalidation to finish
		for len(c.copyset[args.Addr]) > 0 {
		}
		reply.Err = OK
		// update owner
		c.owner[args.Addr] = Owner{OwnerAddr: c.clients[args.ClientID], AccessType: 2}
	}
	log.Println("done handling")
	return nil
}

func (c *Central) invalidateCaches(pageID uintptr, thisClient int) []byte {
	copyset, ok := c.copyset[pageID]
	if ok {
		for clientID, _ := range copyset {
			if clientID == thisClient {
				continue
			}
			log.Println(c.clients[clientID])
			go c.makeInvalidCopyset(pageID, clientID)
		}
	}
	if owner, ok := c.owner[pageID]; ok && owner.OwnerAddr != c.clients[thisClient] {
		return c.makeInvalidOwner(pageID, owner.OwnerAddr)
	}
	return nil
}

func (c *Central) makeReadonlyOwner(addr uintptr, clientAddr string) {
	log.Println("make readonly owner", clientAddr)
	args := InvalidateArgs{Addr: addr, NewAccess: 1, ReturnPage: false}
	reply := InvalidateReply{}
	ok := call(clientAddr, "Client.ChangeAccess", &args, &reply)
	for !ok {
		// Wait until expires
		ok = call(clientAddr, "Client.ChangeAccess", &args, &reply)
	}
	c.owner[addr] = Owner{OwnerAddr: clientAddr, AccessType: 1}
}

func (c *Central) makeInvalidOwner(addr uintptr, clientAddr string) []byte {
	log.Println("make invalid owner", clientAddr)
	args := InvalidateArgs{Addr: addr, NewAccess: 0, ReturnPage: true}
	reply := InvalidateReply{}
	ok := call(clientAddr, "Client.ChangeAccess", &args, &reply)
	for !ok {
		// Wait until expires
		ok = call(clientAddr, "Client.ChangeAccess", &args, &reply)
	}
	c.owner[addr] = Owner{OwnerAddr: clientAddr, AccessType: 0}
	return reply.Data
}

func (c *Central) makeInvalidCopyset(addr uintptr, clientID int) {
	log.Println("make invalid copyset", c.clients[clientID])
	args := InvalidateArgs{Addr: addr, NewAccess: 0, ReturnPage: false}
	reply := InvalidateReply{}
	ok := call(c.clients[clientID], "Client.ChangeAccess", &args, &reply)
	for !ok {
		// Wait until expires
		ok = call(c.clients[clientID], "Client.ChangeAccess", &args, &reply)
	}
	delete(c.copyset[addr], clientID)
}

func (c *Central) initialize(clients map[int]string, numpages int) {
	c.clients = make(map[int]string)
	c.register = make(map[int]bool)
	c.owner = make(map[uintptr]Owner)
	c.locks = make(map[uintptr]*sync.Mutex)
	for id, addr := range clients {
		c.clients[id] = addr
	}
	for i := 0; i < numpages; i++ {
		c.locks[uintptr(i*PageSize)] = &sync.Mutex{}
	}
	c.copyset = make(map[uintptr]map[int]int)
	go c.initializeRPC()
}

func MakeCentral(clients map[int]string, numpages int) *Central {
	c := Central{}
	c.initialize(clients, numpages)
	return &c
}

func (c *Central) initializeRPC() {
	rpc.Register(c)
	l, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatal("listen error:", err)
	}
	defer l.Close()
	log.Println("central server listening on port", port)

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatal("accept error:", err)
		}
		go rpc.ServeConn(conn)
	}
}
