package dsm

type Err string

type Args struct{}

type Reply struct {
	Err Err
}

type ConfirmationArgs struct {
	ClientID int
	Addr     uintptr
}

type RegisterArgs struct {
	ClientID int
}

type RegisterReply struct {
	Err Err
}

type ReadWriteArgs struct {
	ClientID int
	Addr     uintptr
	Access   int
}

type ReadWriteReply struct {
	Err      Err
	HadOwner bool
	Owner    string
	Data     []byte
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

var default_owner_id int = 0
var default_owner_addr string = "123"

type DReadWriteArgs struct {
	ClientID int
	Addr     uintptr
	Access   int
}

type DReadWriteReply struct {
	Err      Err
	HadOwner bool
	Owner    string
	Data     []byte
	// Lease Lease
}
