#!/usr/bin/env sh
# build the wasm and inline it into one self-contained html page (web/dist/index.html)
set -eu
cd "$(dirname "$0")/.."
zig build wasm
mkdir -p web/dist
python3 - <<'PY'
import base64
t = open("web/index.template.html").read()
w = base64.b64encode(open("web/fluids.wasm", "rb").read()).decode()
open("web/dist/index.html", "w").write(t.replace("__WASM__", w))
PY
echo "wrote web/dist/index.html"
