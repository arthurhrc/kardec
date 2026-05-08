package kardec

import "time"

// SetCreationDate fixes the timestamp the renderer writes to the
// PDF's /Info /CreationDate entry. Two renders of the same Document
// using the same fixed timestamp produce byte-identical output —
// useful for reproducible builds, content-addressed caching, and
// hash-based diffing in CI.
//
// Without an explicit value the renderer falls back to time.Now at
// emission time, which preserves the conventional behaviour but
// breaks byte-equality between runs.
//
// SetCreationDate returns the document for fluent chaining.
func (d *Document) SetCreationDate(t time.Time) *Document {
	if d.err != nil {
		return d
	}
	stamp := t
	d.creationDate = &stamp
	return d
}

// CreationDate returns the timestamp configured via SetCreationDate
// plus a boolean indicating whether one was set. Render reads this
// to decide between the fixed stamp and time.Now.
func (d *Document) CreationDate() (time.Time, bool) {
	if d.creationDate == nil {
		return time.Time{}, false
	}
	return *d.creationDate, true
}
