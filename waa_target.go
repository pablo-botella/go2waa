package go2waa

// WaaTarget holds how to reach a WAA server: host, port, timeouts.
type WaaTarget struct {
	Host string // default 127.0.0.1
	Port int    // default 1024

	// Timeouts in seconds
	ConnTimeout  int    // default 30
	ReadTimeout  int    // default 60
	WriteTimeout int    // default 60
	Name         string // optional name for this target, used in a router
	_init        bool   // internal: true if initialized
}

var _ WaaTargetParam = (*WaaTarget)(nil)

// Init fills the target: empty host → 127.0.0.1, zero port → 1024, and
// the optional timeouts are conn, read, write in seconds (zero → 30,
// 60, 60). A target used without Init gets the same defaults on its
// first call.
func (t *WaaTarget) Init(host string, port int, timeouts ...int) {
	if host == "" {
		t.Host = "127.0.0.1"
	} else {
		t.Host = host
	}
	if port == 0 {
		t.Port = 1024
	} else {
		t.Port = port
	}
	if len(timeouts) > 0 {
		t.ConnTimeout = timeouts[0]
	}
	if len(timeouts) > 1 {
		t.ReadTimeout = timeouts[1]
	}
	if len(timeouts) > 2 {
		t.WriteTimeout = timeouts[2]
	}
	if t.ConnTimeout == 0 {
		t.ConnTimeout = 30
	}
	if t.ReadTimeout == 0 {
		t.ReadTimeout = 60
	}
	if t.WriteTimeout == 0 {
		t.WriteTimeout = 60
	}
	t._init = true

}

// GetWaaTargetParam satisfies WaaTargetParam by answering itself: a
// WaaTarget IS its own destination, whatever the conversation carries.
func (w *WaaTarget) GetWaaTargetParam(ctx WaaCtx) (error, *WaaTarget) {
	_ = ctx // currently unused, but part of the interface
	if !w._init {
		w.Init("127.0.0.1", 1024, 30, 60, 60)
	}
	return nil, w
}
