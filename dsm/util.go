package dsm

type Err string

type ReadWriteArgs struct {
	ClientID int
	Addr     uintptr
	Access   int
}

type ReadWriteReply struct {
	Err   Err
	Owner string
	// Lease Lease
}

type PageRequestArgs struct {
	Addr        uintptr
	RequestType int
	// Lease       Lease
}

type PageRequestReply struct {
	Err  Err
	Data []byte
}

type InvalidateArgs struct {
	Addr       uintptr
	NewAccess  int
	ReturnPage bool
}

type InvalidateReply struct {
	Err  Err
	Data []byte
}

type NewClientArgs struct {
	Id    int
	Addr  string
	Pages []uintptr
}

type NewClientReply struct {
	Err Err
}
