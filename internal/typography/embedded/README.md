# Bundled fonts

Kardec ships with four OFL-licensed font families that are embedded into the
final binary via `embed.FS`. The selection mirrors what most desktop word
processors expose by default and provides metric-compatible substitutes for
the proprietary Microsoft / Apple defaults.

| Family            | Faces                                  | License | Source                                                  | Substitute for |
|-------------------|----------------------------------------|---------|---------------------------------------------------------|----------------|
| Liberation Sans   | Regular, Bold, Italic, BoldItalic      | OFL 1.1 | https://github.com/liberationfonts/liberation-fonts     | Arial          |
| Liberation Serif  | Regular, Bold, Italic, BoldItalic      | OFL 1.1 | https://github.com/liberationfonts/liberation-fonts     | Times New Roman|
| Carlito           | Regular, Bold, Italic, BoldItalic      | OFL 1.1 | https://github.com/googlefonts/carlito                  | Calibri        |
| JetBrains Mono    | Regular, Bold, Italic, BoldItalic      | OFL 1.1 | https://github.com/JetBrains/JetBrainsMono              | Code / monospace |

Total payload is roughly 7 MB uncompressed; gzip-compressed object code
adds a few MB to the final binary.

## Math typography

The `MathFont` interface (see `../math.go`) is served by **Latin Modern
Math** — a GUST-OFL math companion to Latin Modern Roman that ships
every Greek letter, big operator, and relation the math layout track
needs. Its ~500 KB OTF is **not** stored in this directory: the bytes
come from the upstream `github.com/go-fonts/latin-modern/lmmath` Go
module via `lmmath.TTF` (an `//go:embed`-backed `[]byte`). That keeps
the dependency graph honest — Kardec re-exports the upstream artifact
rather than shadow-copying it — while still satisfying the "no disk
I/O at runtime" constraint shared with the OFL families above.

| Family            | Faces   | License  | Source                                                                                                       |
|-------------------|---------|----------|--------------------------------------------------------------------------------------------------------------|
| Latin Modern Math | Regular | GUST-OFL | https://www.gust.org.pl/projects/e-foundry/lm-math (vendored via `github.com/go-fonts/latin-modern/lmmath`)  |

Total math-font payload: ~500 KB OTF, kept off the binary's own
`embed.FS` and pulled instead from the upstream module.

## Naming convention

Files MUST be named `<Family>-<Weight><Italic>.ttf`, e.g.
`LiberationSans-BoldItalic.ttf`. The list in `embedded.go` (`builtinFaces`)
is the source of truth; add a new face by appending to that table and
dropping the matching TTF file in this directory.

## License notes

The SIL Open Font License 1.1 permits embedding, redistribution, and use
in commercial products without attribution in compiled output, provided the
license text accompanies any redistribution of the font files themselves.
The verbatim license texts are mirrored alongside each font's upstream
repository linked above. Distributors of Kardec source/release artifacts
SHOULD include the relevant `LICENSE`/`OFL.txt` files when shipping these
TTFs.

## Replacing a font

1. Drop the new TTF in this directory using the naming convention above.
2. Update or extend the `builtinFaces` slice in `../embedded.go`.
3. Run `go test ./internal/typography/...` to verify metrics smoke tests.
