package go2waa

import (
	"fmt"
	"net"
	"time"
)

type WaaTargetParam interface {
	GetWaaTargetParam(ctx WaaCtx) (error, *WaaTarget)
}

// Call dials the WAA server, runs one complete conversation against ctx
// and closes the connection. The connection is not persistent: one call,
// one connection, gone when the call returns.
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
