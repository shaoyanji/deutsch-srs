#!/usr/bin/env python3
"""Render README.md into the single page GitHub Pages serves.

deutsch-srs is a terminal application, so there is no web build to deploy. Rather
than maintain a second copy of the documentation, the published page is the README
itself with a stylesheet attached: one source, one render.

Exit 0 with dist/index.html written, 2 when the README is missing.
"""

import pathlib
import sys

import markdown

TITLE = "deutsch-srs — a keyboard-first German SRS in the terminal"

STYLE = """
:root { color-scheme: light dark; }
body { max-width: 46rem; margin: 3rem auto; padding: 0 1.25rem;
       font: 16px/1.65 -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; }
h1, h2, h3 { line-height: 1.25; margin-top: 2rem; }
pre { overflow-x: auto; padding: .9rem 1rem; border-radius: 6px; background: #f4f4f5; }
code { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: .9em; }
pre code { font-size: .85em; }
table { border-collapse: collapse; margin: 1rem 0; }
th, td { border: 1px solid #d4d4d8; padding: .35rem .6rem; text-align: left; }
img { max-width: 100%; }
blockquote { margin: 1rem 0; padding-left: 1rem; border-left: 3px solid #d4d4d8; color: #52525b; }
a { color: #0b6bcb; }
@media (prefers-color-scheme: dark) {
  body { background: #111113; color: #e4e4e7; }
  pre { background: #1c1c1f; }
  th, td { border-color: #3f3f46; }
  blockquote { border-color: #3f3f46; color: #a1a1aa; }
  a { color: #7dd3fc; }
}
"""

PAGE = """<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>__TITLE__</title>
<style>__STYLE__</style>
</head>
<body>
__BODY__
</body>
</html>
"""


def main() -> int:
    root = pathlib.Path(__file__).resolve().parents[1]
    readme = root / "README.md"
    if not readme.is_file():
        print(f"render-readme: no README.md at {readme}", file=sys.stderr)
        return 2

    body = markdown.markdown(
        readme.read_text(encoding="utf-8"),
        extensions=["fenced_code", "tables", "sane_lists"],
    )
    page = PAGE.replace("__TITLE__", TITLE).replace("__STYLE__", STYLE).replace("__BODY__", body)

    out = root / "dist" / "index.html"
    out.parent.mkdir(parents=True, exist_ok=True)
    out.write_text(page, encoding="utf-8")
    print(f"render-readme: wrote {out} ({out.stat().st_size} bytes)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
