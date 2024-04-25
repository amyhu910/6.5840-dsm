package ivy

type ReadWriteArgs struct {
	ClientID int
	PageID   int
	Access   int
}

type ReadWriteReply struct {
	Err   Err
	Owner int
	Lease Lease
}

type PageRequestArgs struct {
	PageID      int
	RequestType int
	Lease       Lease
}

type PageRequestReply struct {
	Err  Err
	Data []byte
}

type InvalidateArgs struct {
	PageID     int
	NewAccess  int
	ReturnPage bool
}

type InvalidateReply struct {
	Err  Err
	Data []byte
}
