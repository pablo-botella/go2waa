---
mkskill:
  pos: 200
  in: ai*
---

# go2waa — agent notes

Go package `github.com/pablo-botella/go2waa`: the core of the Alaska
Xbase++ WAA gateway protocol. It holds one complete conversation with a
WAA server (the backend behind `WAA1GATE.EXE`) from plain Go code — no
web server required. Faithful port of Alaska's C gateway (1998-2003) on
the wire; fully virtual on the surface.

Layout — one file per piece:

```
waa_ctx.go        WaaCtx (the interface) + KvEntry + Eol* constants
null_waa_ctx.go   NullWaaCtx — do-nothing implementation, embeddable
direct_waa_ctx.go DirectWaaCtx + NewDirectWaaCtx — fill-by-hand context
waa_client.go     WaaClient — connection config + Call(ctx)
caller.go         run() — the conversation engine (unexported)
protocol.go       Message framing: GetMessage/PutMessage/PutMessageString, Cmd*
test/             the test suite (package test), driven strictly through
                  the public API — no mkskill unit of its own: it is just
                  a test, documented here
```

Dependency: `github.com/pablo-botella/linereader` (header line parsing).
Everything else is stdlib.
