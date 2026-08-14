---
name: go2waa
description: "Go core for the Alaska Xbase++ WAA gateway protocol — a fully virtual conversation context (WaaCtx), an in-memory DirectWaaCtx, and Call over a virtual destination (WaaTarget, or WaaRouter multibinding by WAA_PACKAGE with a \"!\" migration lever) reaching a WAA server straight from Go code, no web server in between. Use when working with github.com/pablo-botella/go2waa, WAA1GATE-style gateways or Xbase++ Web Application Adaptor backends."
---

# go2waa — agent notes

Go package `github.com/pablo-botella/go2waa`: the core of the Alaska
Xbase++ WAA gateway protocol. It holds one complete conversation with a
WAA server (the backend behind `WAA1GATE.EXE`) from plain Go code — no
web server required. Wire-compatible with Alaska's C gateway (1998-2003);
fully virtual on the surface.

Layout — one file per piece:

```
waa_ctx.go          WaaCtx (the interface) + KvEntry + Eol* constants
null_waa_ctx.go     NullWaaCtx — do-nothing implementation, embeddable
direct_waa_ctx.go   DirectWaaCtx + NewDirectWaaCtx — fill-by-hand context
waa_target_param.go WaaTargetParam (the virtual destination) + Call(param, ctx)
waa_target.go       WaaTarget — how to reach one WAA server
waa_router.go       WaaRouter — multibinding: WAA_PACKAGE → target
error.go            the sentinel errors
caller.go           run() — the conversation engine (unexported)
protocol.go         Message framing: GetMessage/PutMessage/PutMessageString, Cmd*
test/               the test suite (package test), driven strictly through
                    the public API — no mkskill unit of its own: it is just
                    a test, documented here
```

Dependency: `github.com/pablo-botella/linereader` (header line parsing).
Everything else is stdlib.

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

### WaaTarget, WaaTargetParam and Call

```go
type WaaTargetParam interface {          // the virtual destination
    GetWaaTargetParam(ctx WaaCtx) (error, *WaaTarget)
}

target := &go2waa.WaaTarget{}
target.Init("127.0.0.1", 1027)           // timeouts optional: conn, read, write
err, dispatched := go2waa.Call(target, ctx) // dial → one conversation → close
```

- `Call(param, ctx)` resolves the destination THROUGH the param (which
  may inspect the ctx), dials it, runs one conversation, closes. It is
  the only public entry — the loop (`run`) stays unexported. It answers
  `(error, dispatched)`: `dispatched == false` means nobody dialed — the
  request is not for WAA (see the router below); serve it elsewhere.
- `WaaTarget` is how to reach ONE server: Host/Port/timeouts (zero →
  127.0.0.1, 1024, 30/60/60 s), plus a Name used by routers. Stateless
  config: safe for concurrent calls, each with its own ctx. It satisfies
  `WaaTargetParam` by returning itself.
- Note the package's own convention on these signatures: error first.
- Since `Call` only ever sees the interface, rolling your own
  `WaaTargetParam` for special needs is trivial: one method deciding the
  target (or refusing with `ErrShouldBeDispatchedElsewhere`) from
  whatever the ctx answers — per form, per user, per time of day.

### WaaRouter — multibinding and migration

```go
r := &go2waa.WaaRouter{}
r.AddTarget("main", "10.0.0.1", 1027)     // first target = the default
r.AddTarget("aux", "10.0.0.2", 1027)      // name "" → auto "host:port"
r.MapPackageToTarget("almacen", "aux")    // this package goes to aux
r.MapPackageToTarget("ventas", "!")       // "!": already migrated — not WAA's
err, dispatched := go2waa.Call(r, ctx)    // picks by the route variable
```

- Unmapped packages go to the default target (zero value: the FIRST one
  added; `SetDefaultTarget(name)` changes it, `SetDefaultTarget("!")`
  means no default at all).
- `"!"` (or no default) answers `ErrShouldBeDispatchedElsewhere` with
  `dispatched == false` — elsewhere being anything that is NOT WAA; the
  caller's catch serves it its own way. The typical use is the MIGRATION
  lever: an Xbase++/WAA program made of several packages moves to Go one
  package at a time — as each replacement lands, its package is mapped
  to `"!"`; WAA keeps serving the rest, and nobody notices the seam.
- The package variable defaults to `WAA_PACKAGE`; `SetPackageVarName(name)`
  changes it (and `""` resets to the default) — usually you migrate
  package by package, but inside one package another variable may play
  the pivot (form by form, whatever the app calls for).
- Target names match case-insensitively; package values match exactly.
- The sentinels live in `error.go`: `ErrShouldBeDispatchedElsewhere`,
  `ErrTargetNameNotFound`, `ErrTargetNameAlreadyExists`,
  `ErrInvalidTargetName`, `ErrTargetIndexOutOfRange`,
  `ErrNoWaaTargetConfigured`.

## Semantics that matter

- **One connection per call.** The protocol is not persistent: CONNECT →
  READY → commands → DISCONNECT, socket closed. `WaaTarget` holds no live
  state; a live `DirectWaaCtx` belongs to exactly one call.
- **The destination is virtual** (`WaaTargetParam`): a `WaaTarget` is one
  server, a `WaaRouter` picks one by the package variable (`WAA_PACKAGE`
  by default) — and a package mapped to `"!"` is served elsewhere:
  anything that is not WAA. `Call` answers
  `(ErrShouldBeDispatchedElsewhere, false)` and the caller serves it its
  own way — typically the migration case: package by package to Go, WAA
  serving the rest, no seams visible.
- **Var matching is case-insensitive, presentation keeps the original
  case.** `DirectWaaCtx` groups by UPPERCASE key; each `KvEntry` keeps the
  name case of its first appearance. `OnGetAllVars` returns entries sorted
  alphabetically by the grouping key — deterministic output.
- **`CmdGetVar` answers the first value** (like the C `getVar()`); the
  full slice exists only on the Go side. Repeated keys travel through
  `GetAllVars` as repeated `k=v` pairs, URL-encoded.
- **CGI framing lives in the engine.** Header lines are buffered until
  the empty separator line arrives, then served via `OnHeaderLine` /
  `OnEndHeader`; body flows through `OnPutData`. No separator ever → all
  accumulated output is delivered to `OnPutData` at disconnect (the C
  behavior). Line terminators may be CRLF, LF or CR, even mixed; a
  trailing `\r` at a message boundary is held until the next byte decides.
- **The terminator ints are the bytes**: `EolCrLf == 0x0D0A`. They come
  from linereader's `EolType`.
- **Byte transparency.** No charset handling anywhere: bytes in, bytes
  out, like every original gateway (CGI, ISAPI). Legacy apps typically
  live in CP1252 and declare it in their Content-Type header; encoding
  conversion, if ever needed, belongs to the caller.
- **Email is an opaque `[]byte`** (de facto RFC 822: `From:` first, one
  or more `To:`, CRLF endings — but the package does not parse it).
- **File transfers**: `OnOpenFile` returns the destination stream; the
  engine writes and closes it (also on abort). Opening a new file with
  one already open closes the previous one, like the C.
- **Never call git, never touch the filesystem** on the package's own
  behalf: the package does no disk I/O — `DirectWaaCtx` collects files
  in memory; where they land is the caller's decision.

## The wire protocol

Message framing (little-endian, from Alaska's `MESSAGE.C`):

```
<msgLen int32><cmd int32><data>     msgLen = 4 (cmd) + len(data)
```

`GetMessage`/`PutMessage`/`PutMessageString` work over plain
`io.Reader`/`io.Writer` — any transport.

Commands (from `WAA1CMD.CH`; the gateway starts, the server terminates):

```
0   CmdConnect        gateway → server, first message
1   CmdDisconnect     server's last command (gateway ACKs)
2   CmdReady          server's answer to Connect
3   CmdAck            positive acknowledge
4   CmdNak            negative acknowledge
100 CmdGetEnv         server asks one env var    → 110 CmdPutEnv
200 CmdGetVar         server asks one form var   → 210 CmdPutVar
201 CmdPut            server sends output (CGI block) → Ack
300 CmdGetAllVars     server asks everything     → 310 CmdPutAllVars
400 CmdOpenFile       open file for writing      → Ack/Nak
401 CmdPutFileData    file data                  → Ack/Nak
402 CmdCloseFile      close file                 → Ack/Nak
500 CmdGetDocRoot     document root              → 210 CmdPutVar
600 CmdGetScriptName  script name                → 210 CmdPutVar
700 CmdEmail          send email (raw message)   → Ack/Nak
```

Quirks inherited on purpose: `CmdPut` is 201 (inside the GetVar range),
and DocRoot/ScriptName answer with `CmdPutVar`, not a code of their own.
The original C gateways (CGI W32/OS2/Linux and ISAPI) support neither
file upload (multipart) nor any charset conversion — and neither does
this package: fidelity first.

## Tests

`test/` is just the test suite — it carries no mkskill unit of its own
(nothing to document separately: this section covers it), and it
exercises the module strictly through the public API (`run` stays
unexported).

- `go2waa_test.go` — `TestProtocol` (framing over `net.Pipe`),
  `TestCallVirtual` and `TestCallNak` (`go2waa.Call` with a `WaaTarget`
  against a scripted mock WAA server on a loopback port; the `script`
  helper turns any mismatch into a closed connection, which fails the
  call). The `virtualContext` there is the reference in-memory `WaaCtx`
  implementation, embedding `NullWaaCtx`.
- `waa_call_test.go` — `TestWaaCall`: one real call against a live WAA
  server, driven by `test/test.json` (the Go flavor of the Xbase
  `ao_waa_call` request file: a `server` block — host, port, output
  files, `use_process_environment` — instead of the in-process
  `application` one; payload values may be strings or arrays). It
  requires `test/waa/echo.dll` loaded in a running WAA server (the
  request calls `WAA_PACKAGE=echo`, `WAA_FORM=echo`). `WAA_HOST` /
  `WAA_PORT` override the JSON; `WAA_CALL_REQUEST` points elsewhere.
  Opt-in behind the `live` build tag — plain `go test ./...` never runs
  it; invoke with `go test -tags live -run TestWaaCall ./test`, and once
  invoked a missing server is a FAILURE, never a skip.
- `test/waa/` — the echo fixture: `echo.prg` (Xbase++, registers form
  `echo`) dumps every variable received into an HTML table, nested for
  multi-values; `build.bat` compiles it, `echo.dll` is the built package.
  Run a live round trip with the server loaded with echo, then check
  `output.txt` (headers) and `output.html` (body) in `test/`.

Run everything: `go test ./...` from the module root.
