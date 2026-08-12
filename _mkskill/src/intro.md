---
mkskill:
  pos: 10
  in: readme
---

# go2waa

Go core for the Alaska Xbase++ WAA gateway protocol: hold one complete
conversation with a WAA server — the backend behind `WAA1GATE.EXE` — from
any Go code, with or without a web server in between.

The wire framing, the conversation flow and the CGI output handling are
compatible with Alaska's original C gateway (1998-2003). The surface
around them is new: a fully virtual context (`WaaCtx`) decides where every
answer comes from and where the output goes — an HTTP request, a plain Go
call, a test, anything.
