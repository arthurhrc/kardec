package layout

import "math"

// breakLinesOptimal runs the Knuth-Plass optimum-fit algorithm
// against the same token stream the greedy breakLines consumes.
//
// Tokens are mapped to the classic box / glue / penalty model:
//
//   - non-whitespace token  → box of fixed width
//   - whitespace token      → glue with stretch and shrink
//   - end of stream         → forced breakpoint
//
// A dynamic-programming sweep finds the breakpoint sequence that
// minimises Σ (linePenalty + badness)² across all line counts.
// Compared to first-fit, the result distributes whitespace more
// evenly, eliminates last-word "loose lines", and reduces rivers in
// justified paragraphs — at the cost of an O(n²) worst case (n =
// space count) which stays well below 1 ms for paragraph-size
// inputs.
//
// Hyphenation candidates inside boxes are NOT considered in
// v0.18.0. Optimal-mode paragraphs whose longest unbreakable run
// exceeds the column width fall back to the greedy breaker so the
// existing tryHyphenate path still applies. v0.18.x will weave
// hyphen penalties into the DP frontier so optimal mode can break
// inside long words too.
func breakLinesOptimal(tokens []token, available float64) []line {
	n := len(tokens)
	if n == 0 {
		return nil
	}
	// If any single box is wider than the column, the DP cannot
	// produce a feasible path through it. Greedy mode carries the
	// hyphenation fallback that may rescue these inputs; defer.
	for _, t := range tokens {
		if !t.isSpace && t.width > available {
			return breakLines(tokens, available)
		}
	}

	const (
		stretchFactor = 1.0  // glue.stretch = width × this
		shrinkFactor  = 0.33 // glue.shrink  = width × this
		maxRatio      = 1.0  // reject lines stretched more than 100 %
		linePenalty   = 10.0 // demerit baseline → (badness + this)²
		hugeDemerits  = 1e18
	)

	// Cumulative width / stretch / shrink up to (exclusive) index i.
	cumW := make([]float64, n+1)
	cumY := make([]float64, n+1)
	cumZ := make([]float64, n+1)
	for i, t := range tokens {
		cumW[i+1] = cumW[i] + t.width
		if t.isSpace {
			cumY[i+1] = cumY[i] + t.width*stretchFactor
			cumZ[i+1] = cumZ[i] + t.width*shrinkFactor
		} else {
			cumY[i+1] = cumY[i]
			cumZ[i+1] = cumZ[i]
		}
	}

	// Feasible breakpoints expressed as "line ends just before this
	// index". Index 0 stands for the start; whitespace indices are
	// candidate breaks (the glue is discarded at the end of the
	// line, mirroring breakLines.flush); index n is the forced final
	// break at end of paragraph.
	breaks := []int{0}
	for i, t := range tokens {
		if t.isSpace && i > 0 {
			breaks = append(breaks, i)
		}
	}
	if breaks[len(breaks)-1] != n {
		breaks = append(breaks, n)
	}

	// bestDemerits[i] is the lowest cumulative cost reaching breakpoint
	// breaks[i]; bestPrev[i] is the predecessor index achieving that
	// minimum (used to trace back the chosen sequence).
	bestDemerits := make([]float64, len(breaks))
	bestPrev := make([]int, len(breaks))
	for i := range bestDemerits {
		bestDemerits[i] = hugeDemerits
		bestPrev[i] = -1
	}
	bestDemerits[0] = 0

	for j := 1; j < len(breaks); j++ {
		bj := breaks[j]
		// Last entry is the forced end-of-paragraph break: tolerate
		// underfull lines (large positive ratio) without penalty so
		// the natural short last line costs nothing.
		forced := j == len(breaks)-1
		for i := 0; i < j; i++ {
			bi := breaks[i]
			if bestDemerits[i] >= hugeDemerits/2 {
				continue
			}
			// A line starts after the previous break, skipping the
			// leading glue (which the line breaker traditionally
			// discards). bi == 0 is the start of the paragraph and
			// has no preceding glue to skip.
			start := bi
			if bi > 0 && tokens[bi].isSpace {
				start++
			}
			// Trailing glue at bj-1 is also discarded for line-
			// width accounting (the spec talks about ending at a
			// glue, not extending up to it).
			endTrim := bj
			trimmedTrailingGlue := bj > start && tokens[bj-1].isSpace
			if trimmedTrailingGlue {
				endTrim = bj - 1
			}
			w := cumW[endTrim] - cumW[start]
			y := cumY[endTrim] - cumY[start]
			z := cumZ[endTrim] - cumZ[start]

			diff := available - w
			var ratio float64
			switch {
			case diff > 0:
				if y > 0 {
					ratio = diff / y
				} else {
					ratio = math.Inf(1)
				}
			case diff < 0:
				if z > 0 {
					ratio = diff / z
				} else {
					ratio = math.Inf(-1)
				}
			}
			if ratio < -1 {
				continue
			}
			if !forced && ratio > maxRatio {
				continue
			}
			badness := 0.0
			if !forced {
				badness = 100.0 * math.Abs(ratio) * math.Abs(ratio) * math.Abs(ratio)
			}
			cost := linePenalty + badness
			d := bestDemerits[i] + cost*cost
			if d < bestDemerits[j] {
				bestDemerits[j] = d
				bestPrev[j] = i
			}
		}
	}

	// No feasible path — fall back to greedy. Happens when stretch
	// budget cannot cover unusually long unbreakable boxes (defended
	// against above) or when the available width is degenerate.
	if bestDemerits[len(breaks)-1] >= hugeDemerits/2 {
		return breakLines(tokens, available)
	}

	// Trace back: collect the chosen breakpoint indices in order.
	var path []int
	for j := len(breaks) - 1; j > 0; j = bestPrev[j] {
		path = append([]int{breaks[j]}, path...)
		if bestPrev[j] <= 0 {
			break
		}
	}

	// Materialise lines from the breakpoint path, mirroring the
	// flush() behaviour of breakLines: skip leading glue, drop
	// trailing glue width.
	var out []line
	start := 0
	for _, end := range path {
		s := start
		for s < end && tokens[s].isSpace {
			s++
		}
		ln := line{}
		for k := s; k < end; k++ {
			t := tokens[k]
			ln.tokens = append(ln.tokens, t)
			ln.width += t.width
			if t.ascentPt > ln.ascent {
				ln.ascent = t.ascentPt
			}
			if t.descentPt > ln.descent {
				ln.descent = t.descentPt
			}
		}
		for len(ln.tokens) > 0 && ln.tokens[len(ln.tokens)-1].isSpace {
			ln.width -= ln.tokens[len(ln.tokens)-1].width
			ln.tokens = ln.tokens[:len(ln.tokens)-1]
		}
		if len(ln.tokens) > 0 {
			out = append(out, ln)
		}
		// Next line starts at end + 1 to skip the glue that was the
		// breakpoint. End-of-stream (end == n) needs no skip.
		if end < n {
			start = end + 1
		} else {
			start = end
		}
	}
	return out
}
