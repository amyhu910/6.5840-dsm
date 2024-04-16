package ivy

type Page struct {
	id     int
	data   []byte
	access int // 0: invalid, 1: read-only, 2: read-write
}

type Client struct {
	id        int
	pagetable map[int]Page
}

func (c *Client) readPage(pageID int) []byte {
	// check locally first, send request to central if necessary
}

func (c *Client) sendReadRequest() {
}

func (c *Client) handleReadResponse() {
}

func (c *Client) writePage(pageID int, data []byte) {
	// check locally first for read-write permissions,
	// send request to central if necessary
}

func (c *Client) sendWriteRequest() {
}

func (c *Client) handleWriteResponse() {
}

func (c *Client) handleInvalidate() {

}

func (c *Client) initialize() {

}
