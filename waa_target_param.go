package go2waa

import (
	"fmt"
	"net"
	"time"
)

// WaaTargetParam is the virtual destination of a call: given the
// conversation context it answers WHICH server to dial. A WaaTarget
// answers itself; a WaaRouter picks by the package variable; anything
// else may implement its own policy.
type WaaTargetParam interface {
	GetWaaTargetParam(ctx WaaCtx) (error, *WaaTarget)
}

// Call resolves the destination through w, dials it, runs one complete
// conversation against ctx and closes the connection — not persistent:
// one call, one connection, gone when the call returns.
//
// The second result says whether the request was dispatched to WAA at
// all: false means nobody was dialed (e.g. the router answered
// ErrShouldBeDispatchedElsewhere) and the request belongs outside WAA —
// typically a package already migrated to another technology, but
// whatever the caller decides: it should serve the request itself.
func Call(w WaaTargetParam, ctx WaaCtx) (error, bool) {
	err, t := w.GetWaaTargetParam(ctx)
	if err != nil {
		return err, false // no waa handler maybe alt method or just an error
	}

	addr := net.JoinHostPort(t.Host, fmt.Sprintf("%d", t.Port))
	conn, err := net.DialTimeout("tcp", addr, time.Duration(t.ConnTimeout)*time.Second)
	if err != nil {
		return err, true // dispatched with error
	}
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(time.Duration(t.ReadTimeout) * time.Second))
	conn.SetWriteDeadline(time.Now().Add(time.Duration(t.WriteTimeout) * time.Second))

	return run(conn, ctx), true // dispatched
}
