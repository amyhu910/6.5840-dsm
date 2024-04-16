package ivy

type Central struct {
	// The central's name
	copyset map[int][]int
	owner   map[int]int
}

func (c *Central) handleRead() {
}

func (c *Central) handleWrite() {
}

func (c *Central) invalidateCaches() {
}

func (c *Central) initialize() {
}
