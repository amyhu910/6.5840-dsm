package ivy

import "6.5840-dsm/labrpc"

type Central struct {
	// The central's name
	clients map[int]*labrpc.ClientEnd
	copyset map[int]map[int]int
	owner   map[int]int
}

func (c *Central) handleReadWrite(args *ReadWriteArgs, reply *ReadWriteReply) {
	if args.Access == 1 {
		data := c.makeReadOnly(args.PageID)
		if _, ok := c.copyset[args.PageID]; !ok {
			c.copyset[args.PageID] = make(map[int]int)
		}
		c.copyset[args.PageID][args.ClientID] = 1
		reply.Err = OK
		reply.Data = data
	} else if args.Access == 2 {
		c.invalidateCaches(args.PageID)
		for len(c.copyset[args.PageID]) > 0 {
		}
		c.owner[args.PageID] = args.ClientID
	}
}

func (c *Central) invalidateCaches(pageID int) {
	copyset, ok := c.copyset[pageID]
	if !ok || len(copyset) == 0 {
		return
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
		ok = c.clients[clientID].Call("Client.ChangeAccess", &args, &reply)
	}
	delete(c.copyset[pageID], clientID)
}

func (c *Central) makeReadOnly(pageID int) []byte {
	clientID, hasOwner := c.owner[pageID]
	if !hasOwner || clientID == -1 {
		return nil
	}
	args := AccessArgs{PageID: pageID, NewAccess: 1}
	reply := AccessReply{}
	ok := c.clients[clientID].Call("Client.ChangeAccess", &args, &reply)
	for !ok {
		ok = c.clients[clientID].Call("Client.ChangeAccess", &args, &reply)
	}
	c.owner[pageID] = -1
	return reply.Data
}

func (c *Central) initialize() {
	c.clients = make(map[int]*labrpc.ClientEnd)
	c.copyset = make(map[int]map[int]int)
	c.owner = make(map[int]int)
}
