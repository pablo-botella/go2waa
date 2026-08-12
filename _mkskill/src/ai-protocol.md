---
mkskill:
  pos: 230
  in: ai*
---

## The wire protocol

Message framing (little-endian, from Alaska's `MESSAGE.C`):

```
<msgLen int32><cmd int32><data>     msgLen = 4 (cmd) + len(data)
```

`GetMessage`/`PutMessage`/`PutMessageString` work over plain
`io.Reader`/`io.Writer` — any transport.

Commands (from `WAA1CMD.CH`; the gateway starts, the server terminates):

```
0   CmdConnect        gateway → server, first message
1   CmdDisconnect     server's last command (gateway ACKs)
2   CmdReady          server's answer to Connect
3   CmdAck            positive acknowledge
4   CmdNak            negative acknowledge
100 CmdGetEnv         server asks one env var    → 110 CmdPutEnv
200 CmdGetVar         server asks one form var   → 210 CmdPutVar
201 CmdPut            server sends output (CGI block) → Ack
300 CmdGetAllVars     server asks everything     → 310 CmdPutAllVars
400 CmdOpenFile       open file for writing      → Ack/Nak
401 CmdPutFileData    file data                  → Ack/Nak
402 CmdCloseFile      close file                 → Ack/Nak
500 CmdGetDocRoot     document root              → 210 CmdPutVar
600 CmdGetScriptName  script name                → 210 CmdPutVar
700 CmdEmail          send email (raw message)   → Ack/Nak
```

Quirks inherited on purpose: `CmdPut` is 201 (inside the GetVar range),
and DocRoot/ScriptName answer with `CmdPutVar`, not a code of their own.
The original C gateways (CGI W32/OS2/Linux and ISAPI) support neither
file upload (multipart) nor any charset conversion — and neither does
this package: fidelity first.
