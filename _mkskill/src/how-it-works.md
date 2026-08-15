---
mkskill:
  pos: 30
  in: readme
---

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
