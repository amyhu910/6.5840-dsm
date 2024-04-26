package dsm_go

type ReadWriteArgs struct {
	ClientID int
	Addr     int
	Access   int
}

type ReadWriteReply struct {
	Err   Err
	Owner int
	Lease Lease
}

type PageRequestArgs struct {
	Addr        int
	RequestType int
	Lease       Lease
}

type PageRequestReply struct {
	Err  Err
	Data []byte
}

type InvalidateArgs struct {
	Addr       int
	NewAccess  int
	ReturnPage bool
}

type InvalidateReply struct {
	Err  Err
	Data []byte
}
