package go2waa

import (
	"fmt"
	"net"
	"time"
)

// WaaClient holds how to reach a WAA server: host, port, timeouts.
// It keeps no live state — each Call opens its own connection — so one
// WaaClient is safe for any number of concurrent calls, each with its
// own WaaCtx.
type WaaClient struct {
	Host string // default 127.0.0.1
	Port int    // default 1024

	// Timeouts in seconds
	ConnTimeout  int // default 30
	ReadTimeout  int // default 60
	WriteTimeout int // default 60
}

// Call dials the WAA server, runs one complete conversation against ctx
// and closes the connection. The connection is not persistent: one call,
// one connection, gone when the call returns.
func (w *WaaClient) Call(ctx WaaCtx) error {
	host := w.Host
	if host == "" {
		host = "127.0.0.1"
	}
	port := w.Port
	if port == 0 {
		port = 1024
	}
	connTimeout := w.ConnTimeout
	if connTimeout == 0 {
		connTimeout = 30
	}
	readTimeout := w.ReadTimeout
	if readTimeout == 0 {
		readTimeout = 60
	}
	writeTimeout := w.WriteTimeout
	if writeTimeout == 0 {
		writeTimeout = 60
	}

	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	conn, err := net.DialTimeout("tcp", addr, time.Duration(connTimeout)*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(time.Duration(readTimeout) * time.Second))
	conn.SetWriteDeadline(time.Now().Add(time.Duration(writeTimeout) * time.Second))

	return run(conn, ctx)
}
