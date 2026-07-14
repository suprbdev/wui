#!/usr/bin/env python3
"""Serve a wui example's web/ directory with the correct WASM MIME type.

Usage: serve.py DIR [PORT]

With no PORT, mirrors wui's WithWebServer auto-pick: binds loopback
only, on the first free port in 8765-8864, falling back to an
OS-assigned port when the whole range is busy. An explicit PORT binds
all interfaces.
"""
import http.server
import mimetypes
import os
import sys

mimetypes.add_type("application/wasm", ".wasm")
os.chdir(sys.argv[1])
port = int(sys.argv[2]) if len(sys.argv) > 2 and sys.argv[2] else 0

if port:
    candidates = [("", port)]
else:
    candidates = [("127.0.0.1", p) for p in range(8765, 8865)] + [("127.0.0.1", 0)]

httpd = None
for addr in candidates:
    try:
        httpd = http.server.ThreadingHTTPServer(
            addr, http.server.SimpleHTTPRequestHandler
        )
        break
    except OSError:
        continue
if httpd is None:
    sys.exit("serve.py: no free port")

print(
    f"Serving {os.getcwd()} at http://localhost:{httpd.server_address[1]}",
    flush=True,
)
try:
    httpd.serve_forever()
except KeyboardInterrupt:
    pass
