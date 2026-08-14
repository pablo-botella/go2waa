package test

import (
	"bytes"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/pablo-botella/go2waa"
)

// TestProtocol verifies GetMessage and PutMessage
func TestProtocol(t *testing.T) {
	// Create a pair of connected endpoints
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	// Send message from the client
	go func() {
		err := go2waa.PutMessage(client, go2waa.CmdConnect, nil)
		if err != nil {
			t.Errorf("PutMessage error: %v", err)
		}

		err = go2waa.PutMessageString(client, go2waa.CmdGetEnv, "REQUEST_METHOD")
		if err != nil {
			t.Errorf("PutMessageString error: %v", err)
		}
	}()

	// Receive on the server
	msg, err := go2waa.GetMessage(server)
	if err != nil {
		t.Fatalf("GetMessage error: %v", err)
	}
	if msg.Cmd != go2waa.CmdConnect {
		t.Errorf("Cmd = %d, want %d", msg.Cmd, go2waa.CmdConnect)
	}
	if len(msg.Data) != 0 {
		t.Errorf("Data len = %d, want 0", len(msg.Data))
	}

	msg, err = go2waa.GetMessage(server)
	if err != nil {
		t.Fatalf("GetMessage error: %v", err)
	}
	if msg.Cmd != go2waa.CmdGetEnv {
		t.Errorf("Cmd = %d, want %d", msg.Cmd, go2waa.CmdGetEnv)
	}
	if msg.DataString() != "REQUEST_METHOD" {
		t.Errorf("Data = %q, want %q", msg.DataString(), "REQUEST_METHOD")
	}
}

// virtualContext is an in-memory WaaCtx implementation: it answers from
// maps and slices and collects the output in buffers. Files and email are
// "not supported" (NAK) when their fields are nil.
type virtualContext struct {
	go2waa.NullWaaCtx

	env        map[string]string
	vars       []go2waa.KvEntry
	docRoot    string
	scriptName string
	headers    []string
	endHeader  int
	output     bytes.Buffer
	files      map[string]*bytes.Buffer
	emails     [][]byte
}

func (c *virtualContext) OnGetEnv(name string) string {
	return c.env[name]
}

func (c *virtualContext) OnGetVar(name string) ([]string, bool) {
	for _, e := range c.vars {
		if e.Key == name {
			return e.Values, true
		}
	}
	return nil, false
}

func (c *virtualContext) OnGetAllVars() []go2waa.KvEntry {
	return c.vars
}

func (c *virtualContext) OnGetDocRoot() string {
	return c.docRoot
}

func (c *virtualContext) OnGetScriptName() string {
	return c.scriptName
}

func (c *virtualContext) OnHeaderLine(line string, lineTerminator int) error {
	c.headers = append(c.headers, line)
	return nil
}

func (c *virtualContext) OnEndHeader(lineTerminator int) error {
	c.endHeader = lineTerminator
	return nil
}

func (c *virtualContext) OnPutData(data []byte) error {
	c.output.Write(data)
	return nil
}

// nopCloser gives the file buffers the io.WriteCloser face OnOpenFile needs
type nopCloser struct{ io.Writer }

func (nopCloser) Close() error { return nil }

func (c *virtualContext) OnOpenFile(filename string) (io.WriteCloser, error) {
	if c.files == nil {
		return nil, errors.New("files not supported")
	}
	buf := &bytes.Buffer{}
	c.files[filename] = buf
	return nopCloser{buf}, nil
}

func (c *virtualContext) OnEmail(rfc822data []byte) error {
	if c.emails == nil {
		return errors.New("email not supported")
	}
	c.emails = append(c.emails, rfc822data)
	return nil
}

// mockWaaServer listens on a loopback port and runs one scripted WAA
// conversation per accepted connection.
type mockWaaServer struct {
	listener net.Listener
	port     int
}

func newMockWaaServer(t *testing.T) *mockWaaServer {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("cannot listen: %v", err)
	}
	return &mockWaaServer{
		listener: listener,
		port:     listener.Addr().(*net.TCPAddr).Port,
	}
}

func (s *mockWaaServer) close() {
	s.listener.Close()
}

func (s *mockWaaServer) handleOne(handler func(*script)) {
	go func() {
		conn, err := s.listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		handler(&script{conn: conn})
	}()
}

// script drives the mock side of a conversation. After a failure every
// step is a no-op, the handler unwinds and the closed connection makes
// the client's Call fail the test.
type script struct {
	conn   net.Conn
	failed string
}

func (s *script) expect(wantCmd int32, wantData string) {
	if s.failed != "" {
		return
	}
	msg, err := go2waa.GetMessage(s.conn)
	if err != nil {
		s.failed = "mock GetMessage: " + err.Error()
		return
	}
	if msg.Cmd != wantCmd {
		s.failed = "mock: unexpected command"
		return
	}
	if msg.DataString() != wantData {
		s.failed = "mock: unexpected data: " + msg.DataString()
		return
	}
}

func (s *script) send(cmd int32, data string) {
	if s.failed != "" {
		return
	}
	if err := go2waa.PutMessageString(s.conn, cmd, data); err != nil {
		s.failed = "mock PutMessageString: " + err.Error()
	}
}

// TestCallVirtual runs a complete WAA conversation through the public
// API: WaaClient.Call against a scripted mock server, with a virtual
// context collecting the results.
func TestCallVirtual(t *testing.T) {
	mock := newMockWaaServer(t)
	defer mock.close()

	ctx := &virtualContext{
		env: map[string]string{"REQUEST_METHOD": "GET"},
		vars: []go2waa.KvEntry{
			{Key: "name", Values: []string{"john", "johnny"}},
			{Key: "city", Values: []string{"madrid"}},
		},
		docRoot:    "/var/www",
		scriptName: "waa1gate",
		files:      make(map[string]*bytes.Buffer),
		emails:     [][]byte{},
	}

	var sc *script
	done := make(chan struct{})
	mock.handleOne(func(s *script) {
		sc = s
		defer close(done)

		// Handshake
		s.expect(go2waa.CmdConnect, "")
		s.send(go2waa.CmdReady, "")

		// Environment variable, known and unknown
		s.send(go2waa.CmdGetEnv, "REQUEST_METHOD")
		s.expect(go2waa.CmdPutEnv, "GET")
		s.send(go2waa.CmdGetEnv, "NO_SUCH_VAR")
		s.expect(go2waa.CmdPutEnv, "")

		// Form variable: the protocol answer carries the first value
		s.send(go2waa.CmdGetVar, "name")
		s.expect(go2waa.CmdPutVar, "john")
		s.send(go2waa.CmdGetVar, "missing")
		s.expect(go2waa.CmdPutVar, "")

		// All variables, URL-encoded in order of first appearance
		s.send(go2waa.CmdGetAllVars, "")
		s.expect(go2waa.CmdPutAllVars, "name=john&name=johnny&city=madrid")

		// Doc root and script name answer with CmdPutVar
		s.send(go2waa.CmdGetDocRoot, "")
		s.expect(go2waa.CmdPutVar, "/var/www")
		s.send(go2waa.CmdGetScriptName, "")
		s.expect(go2waa.CmdPutVar, "waa1gate")

		// Output: a CGI block — header, separator, body — split across
		// messages, with the separator arriving in a later chunk
		s.send(go2waa.CmdPut, "Content-Type: text/html\r\n")
		s.expect(go2waa.CmdAck, "")
		s.send(go2waa.CmdPut, "\r\nHello ")
		s.expect(go2waa.CmdAck, "")
		s.send(go2waa.CmdPut, "World")
		s.expect(go2waa.CmdAck, "")

		// File transfer
		s.send(go2waa.CmdOpenFile, "out.txt")
		s.expect(go2waa.CmdAck, "")
		s.send(go2waa.CmdPutFileData, "file content")
		s.expect(go2waa.CmdAck, "")
		s.send(go2waa.CmdCloseFile, "")
		s.expect(go2waa.CmdAck, "")

		// Email
		s.send(go2waa.CmdEmail, "From: a@b\r\nTo: c@d\r\n\r\nhi")
		s.expect(go2waa.CmdAck, "")

		// Disconnect
		s.send(go2waa.CmdDisconnect, "")
		s.expect(go2waa.CmdAck, "")
	})

	target := &go2waa.WaaTarget{}
	target.Init("127.0.0.1", mock.port)
	tp := go2waa.WaaTargetParam(target) // interface conversion
	if err, _ := go2waa.Call(tp, ctx); err != nil {
		t.Fatalf("Call error: %v", err)
	}

	<-done
	if sc.failed != "" {
		t.Fatal(sc.failed)
	}

	if len(ctx.headers) != 1 || ctx.headers[0] != "Content-Type: text/html" {
		t.Errorf("headers = %q, want [%q]", ctx.headers, "Content-Type: text/html")
	}
	if ctx.endHeader != go2waa.EolCrLf {
		t.Errorf("endHeader terminator = %#x, want %#x", ctx.endHeader, go2waa.EolCrLf)
	}
	if got := ctx.output.String(); got != "Hello World" {
		t.Errorf("output (body) = %q, want %q", got, "Hello World")
	}
	if buf := ctx.files["out.txt"]; buf == nil || buf.String() != "file content" {
		t.Errorf("file out.txt = %v, want %q", buf, "file content")
	}
	email := "From: a@b\r\nTo: c@d\r\n\r\nhi"
	if len(ctx.emails) != 1 || string(ctx.emails[0]) != email {
		t.Errorf("emails = %q, want one %q", ctx.emails, email)
	}
}

// TestCallNak verifies the NAK answers of a context without file and
// email support
func TestCallNak(t *testing.T) {
	mock := newMockWaaServer(t)
	defer mock.close()

	ctx := &virtualContext{} // no files, no emails

	var sc *script
	done := make(chan struct{})
	mock.handleOne(func(s *script) {
		sc = s
		defer close(done)

		// Handshake
		s.expect(go2waa.CmdConnect, "")
		s.send(go2waa.CmdReady, "")

		// File not supported
		s.send(go2waa.CmdOpenFile, "out.txt")
		s.expect(go2waa.CmdNak, "")

		// File data without an open file
		s.send(go2waa.CmdPutFileData, "data")
		s.expect(go2waa.CmdNak, "")

		// Close without an open file
		s.send(go2waa.CmdCloseFile, "")
		s.expect(go2waa.CmdNak, "")

		// Email not supported
		s.send(go2waa.CmdEmail, "From: a@b")
		s.expect(go2waa.CmdNak, "")

		// Disconnect still works
		s.send(go2waa.CmdDisconnect, "")
		s.expect(go2waa.CmdAck, "")
	})

	target := &go2waa.WaaTarget{}
	target.Init("127.0.0.1", mock.port)
	tp := go2waa.WaaTargetParam(target) // interface conversion
	if err, _ := go2waa.Call(tp, ctx); err != nil {
		t.Fatalf("Call error: %v", err)
	}

	<-done
	if sc.failed != "" {
		t.Fatal(sc.failed)
	}
}

// BenchmarkProtocol benchmarks the protocol
func BenchmarkProtocol(b *testing.B) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	done := make(chan bool)

	go func() {
		for i := 0; i < b.N; i++ {
			go2waa.GetMessage(server)
		}
		done <- true
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		go2waa.PutMessageString(client, go2waa.CmdGetEnv, "REQUEST_METHOD")
	}

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		b.Fatal("timeout")
	}
}
