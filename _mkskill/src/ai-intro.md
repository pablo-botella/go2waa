---
mkskill:
  pos: 200
  in: ai*
---

# go2waa — agent notes

Go package `github.com/pablo-botella/go2waa`: the core of the Alaska
Xbase++ WAA gateway protocol. It holds one complete conversation with a
WAA server (the backend behind `WAA1GATE.EXE`) from plain Go code — no
web server required. Wire-compatible with Alaska's C gateway (1998-2003);
fully virtual on the surface.

The deployment picture — classic, and where this package sits:

```
classic:  browser → web server → CGI gateway → WAA server
                                                └─ application packages

now:      browser → web server → Go application + go2waa → WAA server
          Go application (a service, a CLI, a test) → go2waa → WAA server
```

go2waa takes the gateway's seat; the frequent shapes are a web server
fronting a Go application, or the Go application on its own. Note the
two "applications" in play: the Xbase++ one living in the WAA server
(the one the conversation serves) and the Go one using this package.

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
