package dsm_go

type ReadWriteArgs struct {
	ClientID int
	Addr     uintptr
	Access   int
}

type ReadWriteReply struct {
	Err   Err
	Owner int
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
