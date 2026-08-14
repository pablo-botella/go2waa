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
