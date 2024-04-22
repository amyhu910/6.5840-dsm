package ivy

import "testing"

func TestBasic(t *testing.T) {
	cfg := make_config(t, false)
	defer cfg.cleanup()
}
