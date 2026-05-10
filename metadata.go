package kardec

// Document metadata setters. Each writes one entry into the PDF's
// /Info dictionary plus, when PDFA is enabled, the matching XMP
// metadata key (dc:title, dc:creator, dc:description, pdf:Keywords).
// Empty strings clear the field.
//
// Fluent chaining returns *Document; the setters are inert once a
// builder error has been captured (matching every other Document
// setter in the package).

// SetTitle sets the document title written to /Info /Title and to
// dc:title in the PDF/A XMP metadata. The title is what most PDF
// readers display in their window chrome and tab labels.
func (d *Document) SetTitle(title string) *Document {
	if d.err != nil {
		return d
	}
	d.title = title
	return d
}

// SetAuthor sets the document author written to /Info /Author and
// dc:creator in the PDF/A XMP metadata.
func (d *Document) SetAuthor(author string) *Document {
	if d.err != nil {
		return d
	}
	d.author = author
	return d
}

// SetSubject sets the document subject (a short one-line description
// of the topic) written to /Info /Subject and dc:description in the
// PDF/A XMP metadata.
func (d *Document) SetSubject(subject string) *Document {
	if d.err != nil {
		return d
	}
	d.subject = subject
	return d
}

// SetKeywords sets a comma- or semicolon-separated list of search
// keywords written to /Info /Keywords and pdf:Keywords in the PDF/A
// XMP metadata. The exact format is not constrained by the PDF spec;
// "comma-separated, lowercase" is the de-facto convention.
func (d *Document) SetKeywords(keywords string) *Document {
	if d.err != nil {
		return d
	}
	d.keywords = keywords
	return d
}

// Title returns the document title configured via SetTitle, or the
// empty string when none was set. Read-only access for layout and
// renderer integrations.
func (d *Document) Title() string { return d.title }

// Author returns the document author configured via SetAuthor.
func (d *Document) Author() string { return d.author }

// Subject returns the document subject configured via SetSubject.
func (d *Document) Subject() string { return d.subject }

// Keywords returns the keyword string configured via SetKeywords.
func (d *Document) Keywords() string { return d.keywords }
