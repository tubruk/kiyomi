#!/usr/bin/env python3
"""Generate PNG fixtures for e2e tests. Run from e2e/fixtures/providers/ directory."""
import base64
import os

# Minimal 1x1 transparent PNG (base64)
PNG_1X1_TRANSPARENT = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="

def main():
    dir_path = os.path.dirname(os.path.abspath(__file__))
    for filename in ["cover-alpha.png", "cover-beta.png"]:
        filepath = os.path.join(dir_path, filename)
        with open(filepath, "wb") as f:
            f.write(base64.b64decode(PNG_1X1_TRANSPARENT))
        print(f"Created: {filepath}")

if __name__ == "__main__":
    main()
