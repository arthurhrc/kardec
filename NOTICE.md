# Third-party notices

Kardec is licensed under the MIT License (see [LICENSE](LICENSE)). The
distributions below are bundled with the source tree under their own
licenses; users redistributing Kardec must preserve the notices listed
here.

## Bundled fonts (SIL Open Font License 1.1)

The TrueType files under `internal/typography/embedded/` ship under the
SIL Open Font License 1.1. Full attribution and the OFL text live next
to the files at
[`internal/typography/embedded/README.md`](internal/typography/embedded/README.md).

| Family            | Source                                                     |
|-------------------|------------------------------------------------------------|
| Liberation Sans   | https://github.com/liberationfonts/liberation-fonts        |
| Liberation Serif  | https://github.com/liberationfonts/liberation-fonts        |
| Carlito           | https://github.com/googlefonts/carlito                     |
| JetBrains Mono    | https://github.com/JetBrains/JetBrainsMono                 |

## Latin Modern Math (GUST Font License)

`github.com/go-fonts/latin-modern` ships the Latin Modern Math OTF used
by `Document.MathFont()`. The GUST Font License is a free, OFL-derived
license; full text is bundled with the upstream module.

## Direct Go dependencies

See `go.mod` for the full graph. Notable runtime dependencies:

- `github.com/tdewolff/canvas` — MIT
- `github.com/yuin/goldmark` — MIT
- `github.com/go-fonts/latin-modern` — GUST Font License
