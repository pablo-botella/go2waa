---
mkskill:
  pos: 60
  in: readme
---

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
