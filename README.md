# go2waa

Go core for the Alaska Xbase++ WAA gateway protocol: hold one complete
conversation with a WAA server from any Go code, with or without a 
web server in between.

The wire framing, the conversation flow and the CGI output handling are
compatible with Alaska's WAA gateway. The surface
around them is new: a fully virtual context (`WaaCtx`) decides where every
answer comes from and where the output goes — an HTTP request, a plain Go
call, a test, anything.

## The pieces

| Piece | File | Role |
|---|---|---|
| `WaaCtx` | `waa_ctx.go` | The interface: ten `On*` methods answering the server's commands |
| `KvEntry` | `waa_ctx.go` | One form variable with its collected values |
| `NullWaaCtx` | `null_waa_ctx.go` | Do-nothing context — embed it, override what you need |
| `DirectWaaCtx` | `direct_waa_ctx.go` | Fill-by-hand context: data in, results collected in memory |
| `WaaTargetParam` | `waa_target_param.go` | The virtual destination + `Call(param, ctx)` |
| `WaaTarget` | `waa_target.go` | How to reach one WAA server: host, port, timeouts |
| `WaaRouter` | `waa_router.go` | Multibinding: `WAA_PACKAGE` picks the target |
| Framing | `protocol.go` | `Message`, `GetMessage`/`PutMessage`, the `Cmd*` constants |

## A direct call, no web server

```go
ctx := go2waa.NewDirectWaaCtx(nil)
ctx.SetValue("WAA_PACKAGE", "echo")
ctx.SetValue("WAA_FORM", "echo")
ctx.AddValue("multi", "one")
ctx.AddValue("multi", "two")

target := &go2waa.WaaTarget{}
target.Init("127.0.0.1", 1027)
if err, _ := go2waa.Call(target, ctx); err != nil {
    log.Fatal(err)
}

// ctx.Header []string  — the CGI header lines
// ctx.Body   []byte    — the response body
// ctx.Files            — files the application sent, name -> buffer
// ctx.Emails           — raw email messages it requested
```

One `WaaTarget` is safe for any number of concurrent calls — each call
opens its own connection, and each live call needs its own context.

## Several backends, and a way out

The destination of a call is virtual (`WaaTargetParam`): a `WaaTarget`
is one server; a `WaaRouter` holds several and picks by the package
variable (`WAA_PACKAGE` unless you `SetPackageVarName` another) —
unmapped packages go to the first target, mapped ones to their own
backend.

And one mapping is special: `"!"` means *not WAA's* — elsewhere is
anything else you serve your own way. The typical use is the migration
lever for an Xbase++/WAA program made of several packages: as each
package's Go replacement lands, map it to `"!"` — `Call` answers
`dispatched == false`, your catch takes over, and WAA keeps serving the
rest. Package by package, no seams.

```go
r := &go2waa.WaaRouter{}
r.AddTarget("main", "10.0.0.1", 1027)  // the default for every package
r.MapPackageToTarget("ventas", "!")    // ventas already lives in Go

if err, dispatched := go2waa.Call(r, ctx); !dispatched {
    // not WAA's: serve it with the new stack
}
```

And since `Call` only sees the `WaaTargetParam` interface, special needs
are one method away: implement your own — pick the target by form, by
user, by anything the context can answer.

## How a conversation works

The gateway starts, the server ends: `Call` connects, sends CONNECT,
waits for READY, and then serves the application's commands until
DISCONNECT — environment variables, form variables (single or all),
document root, script name, output, file transfers, email requests.
Every command becomes one method call on your `WaaCtx`.

The WAA application's output is a CGI block — header lines, an empty line,
then the body. The engine does the framing once, for every context:
each header line arrives through `OnHeaderLine` (with its terminator:
the `int` value carries the actual EOL bytes, `0x0D0A` for CRLF), the
separator through `OnEndHeader`, and everything after it streams through
`OnPutData`. 

The transport is anything that reads and writes bytes; the connection is
never persistent: one call, one TCP connection, gone when `Call` returns.

## Files and email

A WAA application can do more than answer pages: it can send files back
through the gateway, and ask it to send email. go2waa itself saves no
file and sends no email — it delivers both as calls on your context,
and doing the actual work is the application's job:

- **Files** — `OnOpenFile(filename)` returns the destination stream
  (`io.WriteCloser`) that YOUR code chooses; the engine writes every
  data block to it and closes it when the server closes the file — also
  on abort, so no stream is ever left open. Returning an error (or a
  nil writer) answers NAK: politely refused. `DirectWaaCtx` collects
  them in memory (`ctx.Files`, name → buffer); nothing ever touches a
  disk unless your context puts it there.
Why files at all? A WAA response is essentially a page — a 200 (or a
302 redirect) with `text/html`, with barely any control over the
header — so an application cannot serve a download or an image through
it. The classic circuit: the application shipped the file through the
conversation and the GATEWAY wrote it into the web root — the WAA
server could live on another machine, so the gateway, sitting next to
the web server, was the one holding the pen — and the page referenced
it by URL. A security hazard, and it litters the static content.
go2waa stays aseptic — it only hands your context the file; the
application picks the right home: a database, memory, a private file
when it fits… or the web root, if that is truly what it wants. And
precisely because the gateway seat is now yours, this is the chance to
change that behavior: the same file that yesterday had to land in the
web root can today be served as a real download, kept private behind
authentication, or stored where it belongs.

- **Email** — `OnEmail(rfc822data)` receives the raw message exactly as
  the application composed it (de facto RFC 822: `From:` first, then
  recipients; the package does not parse it). Error = NAK. Handing it
  to an SMTP client, queueing it or dropping it is the application's
  call; `DirectWaaCtx` just collects them (`ctx.Emails`).

`NullWaaCtx` answers NAK to both, so a context that never expects files
or email does not need to think about them.

## Limitations

- **No file upload** (multipart) from the browser to the application.
- **No charset conversion** — bytes in, bytes out: whatever codepage the
  application lives in travels untouched, and any conversion, if ever
  needed, belongs to the caller.

## Install

```
go get github.com/pablo-botella/go2waa
```

## Tests

```
go test ./...
```

The suite under `test/` uses only the public API: protocol framing over
`net.Pipe`, and full conversations through `WaaClient.Call` against a
scripted mock server on a loopback port.

`TestWaaCall` goes further: driven by `test/test.json` (the Go flavor of
the Xbase `ao_waa_call` request file — a `server` block instead of the
in-process `application` one), it performs one real call against a live
WAA server and writes the response to `output.txt`/`output.html`.
It is opt-in, behind a build tag — a plain `go test ./...` never runs it:

```
go test -tags live -run TestWaaCall ./test
```

It needs the echo package (`test/waa/echo.dll`) loaded in a running WAA
server; `WAA_HOST` and `WAA_PORT` override the JSON, `WAA_CALL_REQUEST`
points at another request file. Once invoked, a missing server is a
failure, not a skip — on purpose: a test that cannot do its job should
say so.

`test/waa/` carries that echo fixture: an Xbase++ package (`echo.prg`,
built by `build.bat` into `echo.dll`) that dumps every form variable it
receives back as an HTML table — multi-values included.

## License

MIT — see [LICENSE.md](LICENSE.md).
