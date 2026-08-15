---
mkskill:
  pos: 40
  in: readme
---

## Limitations

- **No file upload** (multipart) from the browser to the application.
- **No charset conversion** — bytes in, bytes out: whatever codepage the
  application lives in travels untouched, and any conversion, if ever
  needed, belongs to the caller.
