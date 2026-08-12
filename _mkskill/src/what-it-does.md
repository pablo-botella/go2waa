---
mkskill:
  pos: 20
  in: readme
---

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
