---
mkskill:
  pos: 35
  in: readme
---

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
