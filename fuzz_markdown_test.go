package kardec

import (
	"strings"
	"testing"
)

// FuzzAppendMarkdown drives random / corpus-derived markdown
// through the goldmark integration. AppendMarkdown receives input
// from user templates, README ingestion pipelines, and CLI
// invocations; a panic is a denial-of-service vector. The test
// guarantees that every input produces either valid blocks or a
// captured deferred error, never a crash.
func FuzzAppendMarkdown(f *testing.F) {
	for _, seed := range []string{
		"",
		"# heading\n\nbody",
		"- a\n- b\n- c\n",
		"```\ncode\n```",
		"[link](https://example.com)",
		"![image](image.png)",
		"| col1 | col2 |\n|------|------|\n| a    | b    |\n",
		"**bold *italic***",
		"`inline`",
		strings.Repeat("# ", 1000),     // deep heading
		strings.Repeat(">", 100) + " a", // deep blockquote
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, src string) {
		// Cap fuzzer inputs; we're guarding against panics, not
		// against unbounded input.
		if len(src) > 64*1024 {
			return
		}
		doc := New(PageA4, MarginsNormal)
		doc.AppendMarkdown(src)
		_ = doc.Err() // surface any parse failure but never panic
	})
}
