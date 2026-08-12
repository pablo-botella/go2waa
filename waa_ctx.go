package go2waa

// The docs are composed from _mkskill/ — edit the sources there, then:
//go:generate go run github.com/pablo-botella/mkskill/cmd/mkskill@latest -q build
//go:generate go run github.com/pablo-botella/mkskill/cmd/mkskill@latest -q -vbuild

import (
	"io"

	"github.com/pablo-botella/linereader"
)

// Line terminators reported to OnHeaderLine and OnEndHeader; the values
// are linereader's EolType.
const (
	EolLf   = int(linereader.EolLf)
	EolCr   = int(linereader.EolCr)
	EolCrLf = int(linereader.EolCrLf)
)

// KvEntry is one form variable with its collected values, in order of
// first appearance.
type KvEntry struct {
	Key    string
	Values []string
}

// WaaCtx is what one WAA conversation talks to. It is fully virtual:
// only methods, no data. The protocol loop (Run) translates each server
// command into a method call, so implementations decide where the data
// comes from and where the output goes — an HTTP request, a plain Go
// call, a test, anything.
type WaaCtx interface {
	// OnGetEnv answers CmdGetEnv: the value of an environment variable,
	// empty if unknown.
	OnGetEnv(name string) string

	// OnGetVar answers CmdGetVar: the decoded values of a form variable
	// (a form key may repeat), and whether it exists. The protocol
	// answer carries the first value.
	OnGetVar(name string) ([]string, bool)

	// OnGetAllVars answers CmdGetAllVars: all form variables in order of
	// first appearance, each with its collected values (a form key may
	// repeat). The protocol answer carries them URL-encoded.
	OnGetAllVars() []KvEntry

	// OnGetDocRoot answers CmdGetDocRoot.
	OnGetDocRoot() string

	// OnGetScriptName answers CmdGetScriptName.
	OnGetScriptName() string

	// OnHeaderLine receives one CGI header line emitted by the WAA
	// application, without its terminator; lineTerminator is one of the
	// Eol* constants. Returning an error aborts the conversation.
	OnHeaderLine(line string, lineTerminator int) error

	// OnEndHeader signals the empty line that closes the header block,
	// with its terminator. Returning an error aborts the conversation.
	OnEndHeader(lineTerminator int) error

	// OnPutData receives the application output that follows the header
	// block — the body. If the application never emits a header
	// separator, all its output arrives here at the end of the
	// conversation. Returning an error aborts the conversation.
	OnPutData(data []byte) error

	// OnOpenFile returns the destination stream for a file sent by the WAA
	// server. The loop writes the received data to it and closes it when
	// the server closes the file. Error or nil writer = NAK.
	OnOpenFile(filename string) (io.WriteCloser, error)

	// OnEmail sends an email message, passed through untouched as received
	// from the WAA server. Error = NAK.
	OnEmail(rfc822data []byte) error
}
