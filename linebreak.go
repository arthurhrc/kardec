package kardec

// LineBreakAlgorithm selects the paragraph line-breaking strategy
// the renderer uses when laying out text blocks. Pass to
// Document.SetLineBreakAlgorithm.
//
// The default (LineBreakFirstFit) is the v0.1-era greedy first-fit
// breaker plus the v0.4 Knuth-Liang hyphenation fallback. It is
// fast, predictable, and produces output identical to every prior
// release.
//
// LineBreakOptimal switches to the Knuth-Plass DP optimum-fit
// algorithm: lines are chosen to minimise summed badness² across
// the paragraph, distributing whitespace more evenly and reducing
// rivers in justified text. The algorithm is O(n²) in the number
// of break candidates but stays sub-millisecond for paragraph-size
// inputs.
//
// v0.18.0 introduces the algorithm behind a per-document feature
// flag so callers can opt in without disturbing existing layout
// fixtures. Hyphenation candidates inside long words are not yet
// woven into the optimal-mode DP — overflowing paragraphs fall
// back to the greedy breaker per-paragraph; v0.18.x will add the
// hyphen penalty path.
type LineBreakAlgorithm uint8

const (
	// LineBreakFirstFit keeps the legacy greedy breaker. Default.
	LineBreakFirstFit LineBreakAlgorithm = iota
	// LineBreakOptimal switches to Knuth-Plass DP optimum-fit.
	LineBreakOptimal
)

// SetLineBreakAlgorithm picks the paragraph line-breaking strategy
// the renderer uses for this document. Calling with the default
// LineBreakFirstFit is equivalent to never calling — the legacy
// greedy breaker stays in effect.
func (d *Document) SetLineBreakAlgorithm(a LineBreakAlgorithm) *Document {
	if d.err != nil {
		return d
	}
	d.lineBreakAlgo = a
	return d
}

// LineBreakAlgorithm reports the currently configured strategy.
// Read by the layout engine to dispatch breakLines calls.
func (d *Document) LineBreakAlgorithm() LineBreakAlgorithm {
	return d.lineBreakAlgo
}
