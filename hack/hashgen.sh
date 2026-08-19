#!/bin/sh

# Emit a .sha256 alongside each built binary. Prefer sha256sum (Linux),
# fall back to shasum (macOS).
if command -v sha256sum >/dev/null 2>&1; then
    for f in bin/vzzn*; do sha256sum "$f" > "$f.sha256"; done
else
    for f in bin/vzzn*; do shasum -a 256 "$f" > "$f.sha256"; done
fi
