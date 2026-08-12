---
mkskill:
  pos: 240
  in: ai*
---

## Tests

`test/` is just the test suite — it carries no mkskill unit of its own
(nothing to document separately: this section covers it), and it
exercises the module strictly through the public API (`run` stays
unexported).

- `go2waa_test.go` — `TestProtocol` (framing over `net.Pipe`),
  `TestCallVirtual` and `TestCallNak` (`WaaClient.Call` against a
  scripted mock WAA server on a loopback port; the `script` helper turns
  any mismatch into a closed connection, which fails the `Call`). The
  `virtualContext` there is the reference in-memory `WaaCtx`
  implementation, embedding `NullWaaCtx`.
- `waa_call_test.go` — `TestWaaCall`: one real call against a live WAA
  server, driven by `test/test.json` (the Go flavor of the Xbase
  `ao_waa_call` request file: a `server` block — host, port, output
  files, `use_process_environment` — instead of the in-process
  `application` one; payload values may be strings or arrays). It
  requires `test/waa/echo.dll` loaded in a running WAA server (the
  request calls `WAA_PACKAGE=echo`, `WAA_FORM=echo`). `WAA_HOST` /
  `WAA_PORT` override the JSON; `WAA_CALL_REQUEST` points elsewhere.
  Opt-in behind the `live` build tag — plain `go test ./...` never runs
  it; invoke with `go test -tags live -run TestWaaCall ./test`, and once
  invoked a missing server is a FAILURE, never a skip.
- `test/waa/` — the echo fixture: `echo.prg` (Xbase++, registers form
  `echo`) dumps every variable received into an HTML table, nested for
  multi-values; `build.bat` compiles it, `echo.dll` is the built package.
  Run a live round trip with the server loaded with echo, then check
  `output.txt` (headers) and `output.html` (body) in `test/`.

Run everything: `go test ./...` from the module root.
