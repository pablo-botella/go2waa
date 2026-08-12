---
mkskill:
  pos: 210
  in: ai*
---

## API

### WaaCtx — the virtual conversation context

Ten `On*` methods; the engine translates each server command into one
call. Implementations decide where data comes from and where output goes.

```go
type WaaCtx interface {
    OnGetEnv(name string) string               // CmdGetEnv; "" if unknown
    OnGetVar(name string) ([]string, bool)     // CmdGetVar; wire carries values[0]
    OnGetAllVars() []KvEntry                   // CmdGetAllVars; wire gets it URL-encoded
    OnGetDocRoot() string                      // CmdGetDocRoot (answers with CmdPutVar)
    OnGetScriptName() string                   // CmdGetScriptName (answers with CmdPutVar)
    OnHeaderLine(line string, lineTerminator int) error // one CGI header line
    OnEndHeader(lineTerminator int) error      // the empty separator line
    OnPutData(data []byte) error               // body data (post-separator)
    OnOpenFile(filename string) (io.WriteCloser, error) // error/nil = NAK
    OnEmail(rfc822data []byte) error           // error = NAK; raw passthrough
}

type KvEntry struct {
    Key    string   // name, case of first appearance
    Values []string // collected values (a form key may repeat)
}

const ( EolLf = 0x0A; EolCr = 0x0D; EolCrLf = 0x0D0A ) // linereader's values:
// the int IS the terminator bytes — self-describing
```

Errors returned from `OnHeaderLine`/`OnEndHeader`/`OnPutData` abort the
conversation; from `OnOpenFile`/`OnEmail` they answer NAK and continue.

### NullWaaCtx

Do-nothing `WaaCtx`: empty answers, discarded output, NAK for files and
email. Embed it and override only what you need — `DirectWaaCtx` does.

### DirectWaaCtx

```go
ctx := go2waa.NewDirectWaaCtx(env)  // env may be nil
ctx.SetMethod("GET")                // REQUEST_METHOD; default POST
ctx.SetValue("name", "v")           // replace with single value
ctx.SetValues("name", []string{…})  // replace with many (slice is cloned)
ctx.AddValue("name", "v2")          // append one more occurrence
// after the call:
ctx.Header []string                 // CGI header lines, no terminators
ctx.Body   []byte                   // body bytes
ctx.Files  map[string]*bytes.Buffer // files the app sent
ctx.Emails [][]byte                 // raw email messages
```

`Env` is a plain exported `map[string]string` (CGI names, e.g.
`SERVER_NAME`). It does not perform the call.

### WaaClient

```go
client := &go2waa.WaaClient{Host: "127.0.0.1", Port: 1027}
err := client.Call(ctx)  // dial → one conversation → close
```

Defaults: 127.0.0.1, port 1024, timeouts 30/60/60 s (fields in seconds).
Stateless config shell: safe for concurrent calls, each with its own ctx.
`Call` is the only public entry to run a conversation — the loop itself
(`run`) is unexported.
