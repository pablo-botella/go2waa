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
