package kardec

// Clause appends a hierarchically numbered paragraph at the given
// level. Level 1 produces top-level numbers (1, 2, 3, ...), level 2
// produces children (1.1, 1.2, 2.1, ...), and so on. The numbering
// follows the standard "deeper levels reset when a shallower level
// advances" rule — calling Clause(2) twice then Clause(1) produces
// 1.1 / 1.2 / 2.
//
// Levels below 1 are clamped to 1. Skipping a level (calling Clause(3)
// before any Clause(2) ran) is allowed and treated as if zeroed
// intermediate counters were initialised on demand, mirroring most
// legal-document numbering tools.
//
// The numbered text is emitted as a Paragraph with the supplied
// runs prepended by "N.N.N " (no period after the deepest number for
// level 1, period for sub-levels — matches Word's "1." / "1.1" style).
func (d *Document) Clause(level int, runs ...Run) *Document {
	if level < 1 {
		level = 1
	}
	d.advanceClauseCounter(level)
	prefix := formatClauseNumber(d.clauseCounters) + " "
	all := make([]Run, 0, len(runs)+1)
	all = append(all, Bold(prefix))
	all = append(all, runs...)
	return d.append(Paragraph{runs: all})
}

// ClauseAt appends a paragraph numbered with an explicit dotted
// label, bypassing the auto-counter. Useful for legal documents that
// reference a fixed clause numbering scheme not aligned with the
// build order.
func (d *Document) ClauseAt(number string, runs ...Run) *Document {
	all := make([]Run, 0, len(runs)+1)
	all = append(all, Bold(number+" "))
	all = append(all, runs...)
	return d.append(Paragraph{runs: all})
}

// advanceClauseCounter applies the level-N increment rule to the
// document's counter stack. The slice grows when a deeper level
// advances and truncates when a shallower one does.
func (d *Document) advanceClauseCounter(level int) {
	for len(d.clauseCounters) < level {
		d.clauseCounters = append(d.clauseCounters, 0)
	}
	if len(d.clauseCounters) > level {
		d.clauseCounters = d.clauseCounters[:level]
	}
	d.clauseCounters[level-1]++
}

// formatClauseNumber composes the dotted label from the counter
// stack: [1] → "1.", [1, 2] → "1.2", [2, 1, 3] → "2.1.3". Top-level
// numbers gain a trailing dot so they read like "1. Definitions";
// deeper levels do not, matching the conventional "1.2 Term" style.
func formatClauseNumber(counters []int) string {
	if len(counters) == 0 {
		return ""
	}
	out := itoaSmall(counters[0])
	if len(counters) == 1 {
		return out + "."
	}
	for i := 1; i < len(counters); i++ {
		out += "." + itoaSmall(counters[i])
	}
	return out
}
