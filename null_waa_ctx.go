package go2waa

import (
	"errors"
	"io"
)

var errNotSupported = errors.New("operation not supported")

// NullWaaCtx is a do-nothing WaaCtx: empty answers, discarded output,
// files and email rejected (NAK). Embed it to implement only the methods
// you need.
type NullWaaCtx struct{}

func (NullWaaCtx) OnGetEnv(name string) string { return "" }

func (NullWaaCtx) OnGetVar(name string) ([]string, bool) { return nil, false }

func (NullWaaCtx) OnGetAllVars() []KvEntry { return nil }

func (NullWaaCtx) OnGetDocRoot() string { return "" }

func (NullWaaCtx) OnGetScriptName() string { return "" }

func (NullWaaCtx) OnHeaderLine(line string, lineTerminator int) error { return nil }

func (NullWaaCtx) OnEndHeader(lineTerminator int) error { return nil }

func (NullWaaCtx) OnPutData(data []byte) error { return nil }

func (NullWaaCtx) OnOpenFile(filename string) (io.WriteCloser, error) {
	return nil, errNotSupported
}

func (NullWaaCtx) OnEmail(rfc822data []byte) error { return errNotSupported }
