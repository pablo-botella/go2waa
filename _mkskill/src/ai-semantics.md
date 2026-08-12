---
mkskill:
  pos: 220
  in: ai*
---

## Semantics that matter

- **One connection per call.** The protocol is not persistent: CONNECT →
  READY → commands → DISCONNECT, socket closed. `WaaClient` holds no live
  state; a live `DirectWaaCtx` belongs to exactly one call.
- **Var matching is case-insensitive, presentation keeps the original
  case.** `DirectWaaCtx` groups by UPPERCASE key; each `KvEntry` keeps the
  name case of its first appearance. `OnGetAllVars` returns entries sorted
  alphabetically by the grouping key — deterministic output.
- **`CmdGetVar` answers the first value** (like the C `getVar()`); the
  full slice exists only on the Go side. Repeated keys travel through
  `GetAllVars` as repeated `k=v` pairs, URL-encoded.
- **CGI framing lives in the engine.** Header lines are buffered until
  the empty separator line arrives, then served via `OnHeaderLine` /
  `OnEndHeader`; body flows through `OnPutData`. No separator ever → all
  accumulated output is delivered to `OnPutData` at disconnect (the C
  behavior). Line terminators may be CRLF, LF or CR, even mixed; a
  trailing `\r` at a message boundary is held until the next byte decides.
- **The terminator ints are the bytes**: `EolCrLf == 0x0D0A`. They come
  from linereader's `EolType`.
- **Byte transparency.** No charset handling anywhere: bytes in, bytes
  out, like every original gateway (CGI, ISAPI). Legacy apps typically
  live in CP1252 and declare it in their Content-Type header; encoding
  conversion, if ever needed, belongs to the caller.
- **Email is an opaque `[]byte`** (de facto RFC 822: `From:` first, one
  or more `To:`, CRLF endings — but the package does not parse it).
- **File transfers**: `OnOpenFile` returns the destination stream; the
  engine writes and closes it (also on abort). Opening a new file with
  one already open closes the previous one, like the C.
- **Never call git, never touch the filesystem** on the package's own
  behalf: the package does no disk I/O — `DirectWaaCtx` collects files
  in memory; where they land is the caller's decision.
