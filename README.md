# go2waa

Go core for the Alaska Xbase++ WAA gateway protocol: hold one complete
conversation with a WAA server — the backend behind `WAA1GATE.EXE` — from
any Go code, with or without a web server in between.

The wire framing, the conversation flow and the CGI output handling are
compatible with Alaska's original C gateway (1998-2003). The surface
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
| `WaaClient` | `waa_client.go` | Connection config; `Call(ctx)` runs one conversation |
| Framing | `protocol.go` | `Message`, `GetMessage`/`PutMessage`, the `Cmd*` constants |

## A direct call, no web server

```go
ctx := go2waa.NewDirectWaaCtx(nil)
ctx.SetValue("WAA_PACKAGE", "echo")
ctx.SetValue("WAA_FORM", "echo")
ctx.AddValue("multi", "one")
ctx.AddValue("multi", "two")

client := &go2waa.WaaClient{Host: "127.0.0.1", Port: 1027}
if err := client.Call(ctx); err != nil {
    log.Fatal(err)
}

// ctx.Header []string  — the CGI header lines
// ctx.Body   []byte    — the response body
// ctx.Files            — files the application sent, name -> buffer
// ctx.Emails           — raw email messages it requested
```

One `WaaClient` is safe for any number of concurrent calls — each call
opens its own connection, and each live call needs its own context.

## How a conversation works

The gateway starts, the server ends: `Call` connects, sends CONNECT,
waits for READY, and then serves the application's commands until
DISCONNECT — environment variables, form variables (single or all),
document root, script name, output, file transfers, email requests.
Every command becomes one method call on your `WaaCtx`.

The application's output is a CGI block — header lines, an empty line,
then the body. The engine does the framing once, for every context:
each header line arrives through `OnHeaderLine` (with its terminator:
the `int` value carries the actual EOL bytes, `0x0D0A` for CRLF), the
separator through `OnEndHeader`, and everything after it streams through
`OnPutData`. If the application never emits a separator, all its output
is delivered as body when the conversation ends — exactly what the C
gateway did.

The transport is anything that reads and writes bytes; the connection is
never persistent: one call, one TCP connection, gone when `Call` returns.

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
