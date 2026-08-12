package go2waa

import (
	"bytes"
	"io"
	"slices"
	"strings"
)

// DirectWaaCtx is a WaaCtx you fill by hand: direct data in, output
// collected in memory to read after the call. It does not perform the
// call itself — pass it to WaaClient.Call. A DirectWaaCtx belongs to one
// conversation: do not share a live one between concurrent calls.
type DirectWaaCtx struct {
	NullWaaCtx

	// Env answers OnGetEnv verbatim (CGI names, e.g. "SERVER_NAME").
	// REQUEST_METHOD defaults to "POST" when not set.
	Env map[string]string

	// Header collects the CGI header lines emitted by the application,
	// one entry per line, without terminators.
	Header []string

	// Body collects the application output after the header separator —
	// or all of it, when the application never emitted a separator.
	Body []byte

	// Files collects the files sent by the application, name -> content.
	Files map[string]*bytes.Buffer

	// Emails collects the email messages requested by the application,
	// raw and in order of arrival.
	Emails [][]byte

	// Form variables grouped by uppercase key; each entry keeps the case
	// of its first appearance.
	vars map[string]*KvEntry
}

// NewDirectWaaCtx returns a DirectWaaCtx ready to fill. env may be nil
// (empty environment).
func NewDirectWaaCtx(env map[string]string) *DirectWaaCtx {
	if env == nil {
		env = make(map[string]string)
	}
	return &DirectWaaCtx{
		Env:   env,
		Files: make(map[string]*bytes.Buffer),
		vars:  make(map[string]*KvEntry),
	}
}

// SetMethod sets the REQUEST_METHOD environment variable.
func (c *DirectWaaCtx) SetMethod(method string) {
	c.Env["REQUEST_METHOD"] = method
}

// SetValue sets a form variable to a single value, replacing whatever
// it had.
func (c *DirectWaaCtx) SetValue(name, value string) {
	c.SetValues(name, []string{value})
}

// SetValues sets a form variable to the given values, replacing whatever
// it had. The variable keeps the name case of its first appearance.
func (c *DirectWaaCtx) SetValues(name string, values []string) {
	key := strings.ToUpper(name)
	if e, ok := c.vars[key]; ok {
		e.Values = slices.Clone(values)
		return
	}
	c.vars[key] = &KvEntry{Key: name, Values: slices.Clone(values)}
}

// AddValue adds one more value to a form variable, creating it if it
// does not exist.
func (c *DirectWaaCtx) AddValue(name, value string) {
	key := strings.ToUpper(name)
	if e, ok := c.vars[key]; ok {
		e.Values = append(e.Values, value)
		return
	}
	c.vars[key] = &KvEntry{Key: name, Values: []string{value}}
}

func (c *DirectWaaCtx) OnGetEnv(name string) string {
	if value, ok := c.Env[name]; ok {
		return value
	}
	if name == "REQUEST_METHOD" {
		return "POST"
	}
	return ""
}

func (c *DirectWaaCtx) OnGetVar(name string) ([]string, bool) {
	e, ok := c.vars[strings.ToUpper(name)]
	if !ok {
		return nil, false
	}
	return e.Values, true
}

func (c *DirectWaaCtx) OnGetAllVars() []KvEntry {
	keys := make([]string, 0, len(c.vars))
	for key := range c.vars {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	entries := make([]KvEntry, 0, len(keys))
	for _, key := range keys {
		entries = append(entries, *c.vars[key])
	}
	return entries
}

func (c *DirectWaaCtx) OnHeaderLine(line string, lineTerminator int) error {
	c.Header = append(c.Header, line)
	return nil
}

func (c *DirectWaaCtx) OnPutData(data []byte) error {
	c.Body = append(c.Body, data...)
	return nil
}

// bufCloser gives the file buffers the io.WriteCloser face OnOpenFile needs
type bufCloser struct{ *bytes.Buffer }

func (bufCloser) Close() error { return nil }

func (c *DirectWaaCtx) OnOpenFile(filename string) (io.WriteCloser, error) {
	buf := &bytes.Buffer{}
	c.Files[filename] = buf
	return bufCloser{buf}, nil
}

func (c *DirectWaaCtx) OnEmail(rfc822data []byte) error {
	c.Emails = append(c.Emails, rfc822data)
	return nil
}
