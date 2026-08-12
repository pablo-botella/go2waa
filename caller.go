package go2waa

import (
	"bytes"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/pablo-botella/linereader"
)

// run drives one complete WAA conversation over conn against ctx:
// CONNECT/READY handshake, then serves the application's commands until
// DISCONNECT. The transport is anything that reads and writes bytes;
// deadlines and closing are the caller's business.
func run(conn io.ReadWriter, ctx WaaCtx) error {
	// Handshake
	if err := PutMessage(conn, CmdConnect, nil); err != nil {
		return err
	}
	msg, err := GetMessage(conn)
	if err != nil {
		return err
	}
	if msg.Cmd != CmdReady {
		return fmt.Errorf("protocol error: expected READY, got %d", msg.Cmd)
	}

	// Currently open file stream (CmdOpenFile / CmdPutFileData / CmdCloseFile)
	var curFile io.WriteCloser
	defer func() {
		if curFile != nil {
			curFile.Close()
		}
	}()

	// CGI output framing (CmdPut): accumulate until the header separator
	// arrives, serve the header through OnHeaderLine/OnEndHeader, then
	// stream the rest to OnPutData. Without a separator, everything
	// pending goes to OnPutData at disconnect (all body).
	var pending []byte
	headerDone := false

	// Main loop
	for {
		msg, err := GetMessage(conn)
		if err != nil {
			return err
		}

		switch msg.Cmd {
		case CmdDisconnect:
			if !headerDone && len(pending) > 0 {
				if err := ctx.OnPutData(pending); err != nil {
					return err
				}
			}
			return PutMessage(conn, CmdAck, nil)

		case CmdGetEnv:
			if err := PutMessageString(conn, CmdPutEnv, ctx.OnGetEnv(msg.DataString())); err != nil {
				return err
			}

		case CmdGetVar:
			value := ""
			if values, ok := ctx.OnGetVar(msg.DataString()); ok && len(values) > 0 {
				value = values[0]
			}
			if err := PutMessageString(conn, CmdPutVar, value); err != nil {
				return err
			}

		case CmdGetAllVars:
			var sb strings.Builder
			for _, e := range ctx.OnGetAllVars() {
				for _, v := range e.Values {
					if sb.Len() > 0 {
						sb.WriteByte('&')
					}
					sb.WriteString(url.QueryEscape(e.Key))
					sb.WriteByte('=')
					sb.WriteString(url.QueryEscape(v))
				}
			}
			if err := PutMessageString(conn, CmdPutAllVars, sb.String()); err != nil {
				return err
			}

		case CmdGetDocRoot:
			if err := PutMessageString(conn, CmdPutVar, ctx.OnGetDocRoot()); err != nil {
				return err
			}

		case CmdGetScriptName:
			if err := PutMessageString(conn, CmdPutVar, ctx.OnGetScriptName()); err != nil {
				return err
			}

		case CmdPut:
			if headerDone {
				if err := ctx.OnPutData(msg.Data); err != nil {
					return err
				}
			} else {
				pending = append(pending, msg.Data...)
				if bodyStart, found := findHeaderEnd(pending); found {
					if err := emitHeader(ctx, pending[:bodyStart]); err != nil {
						return err
					}
					headerDone = true
					if bodyStart < len(pending) {
						if err := ctx.OnPutData(pending[bodyStart:]); err != nil {
							return err
						}
					}
					pending = nil
				}
			}
			if err := PutMessage(conn, CmdAck, nil); err != nil {
				return err
			}

		case CmdOpenFile:
			if curFile != nil {
				curFile.Close()
				curFile = nil
			}
			f, err := ctx.OnOpenFile(msg.DataString())
			if err != nil || f == nil {
				if err := PutMessage(conn, CmdNak, nil); err != nil {
					return err
				}
				continue
			}
			curFile = f
			if err := PutMessage(conn, CmdAck, nil); err != nil {
				return err
			}

		case CmdPutFileData:
			if curFile == nil {
				if err := PutMessage(conn, CmdNak, nil); err != nil {
					return err
				}
				continue
			}
			if _, err := curFile.Write(msg.Data); err != nil {
				if err := PutMessage(conn, CmdNak, nil); err != nil {
					return err
				}
				continue
			}
			if err := PutMessage(conn, CmdAck, nil); err != nil {
				return err
			}

		case CmdCloseFile:
			if curFile == nil {
				if err := PutMessage(conn, CmdNak, nil); err != nil {
					return err
				}
				continue
			}
			err := curFile.Close()
			curFile = nil
			if err != nil {
				if err := PutMessage(conn, CmdNak, nil); err != nil {
					return err
				}
				continue
			}
			if err := PutMessage(conn, CmdAck, nil); err != nil {
				return err
			}

		case CmdEmail:
			if err := ctx.OnEmail(msg.Data); err != nil {
				if err := PutMessage(conn, CmdNak, nil); err != nil {
					return err
				}
				continue
			}
			if err := PutMessage(conn, CmdAck, nil); err != nil {
				return err
			}

		default:
			return fmt.Errorf("protocol error: unknown command %d", msg.Cmd)
		}
	}
}

// findHeaderEnd scans accumulated output for the empty line that closes
// a CGI header block and returns the offset where the body starts.
// A trailing '\r' is left undecided: it could be half a CRLF still in
// transit.
func findHeaderEnd(data []byte) (bodyStart int, found bool) {
	pos := 0
	for pos < len(data) {
		// Locate this line's terminator
		i := pos
		for i < len(data) && data[i] != '\r' && data[i] != '\n' {
			i++
		}
		if i >= len(data) {
			return 0, false // unterminated line so far
		}
		termLen := 1
		if data[i] == '\r' {
			if i+1 >= len(data) {
				return 0, false // maybe half a CRLF: wait for more data
			}
			if data[i+1] == '\n' {
				termLen = 2
			}
		}
		if i == pos {
			// Empty line: the separator
			return i + termLen, true
		}
		pos = i + termLen
	}
	return 0, false
}

// emitHeader serves a complete CGI header block (separator included)
// through OnHeaderLine and OnEndHeader.
func emitHeader(ctx WaaCtx, block []byte) error {
	lr := linereader.NewLineReader(bytes.NewReader(block), 0, 0)
	for {
		line, err := lr.ReadLine()
		if err != nil {
			return nil
		}
		term := int(lr.LastEolType)
		if len(line) == 0 {
			return ctx.OnEndHeader(term)
		}
		if err := ctx.OnHeaderLine(string(line), term); err != nil {
			return err
		}
	}
}
