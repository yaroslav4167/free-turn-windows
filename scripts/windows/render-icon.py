#!/usr/bin/env python3
"""Render the canonical SVG into the ICO embedded by Wails (needs rsvg-convert)."""
from pathlib import Path
import struct
import subprocess

root = Path(__file__).resolve().parents[2]
sizes = (16, 24, 32, 48, 64, 128, 256)
images = [subprocess.check_output([
    "rsvg-convert", "-w", str(size), "-h", str(size),
    str(root / "desktop/assets/icon.svg"),
]) for size in sizes]
offset = 6 + 16 * len(sizes)
entries = []
for size, image in zip(sizes, images):
    entries.append(struct.pack("<BBBBHHII", size % 256, size % 256,
                               0, 0, 1, 32, len(image), offset))
    offset += len(image)
(root / "desktop/build/windows/icon.ico").write_bytes(
    struct.pack("<HHH", 0, 1, len(sizes)) + b"".join(entries) + b"".join(images)
)
