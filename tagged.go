package kardec

// SetTagged opts the document into PDF/UA-1 lite tagging.
//
// When enabled, the renderer:
//
//   - declares /MarkInfo << /Marked true >> on the catalog
//   - emits a /StructTreeRoot with one /P StructElem per page
//   - wraps each page's content stream in a marked-content sequence
//     (BDC/EMC) bound to the matching StructElem via MCID
//   - sets /Tabs /S on every page so assistive tech walks
//     annotations in structure order
//   - writes /Lang lang on the catalog when lang is non-empty
//
// lang is a BCP-47 language code (e.g. "en", "pt-BR"). Strict
// PDF/UA-1 conformance requires the language to be declared; pass
// "" only when language is genuinely unknown.
//
// v0.17.0 ships the scaffolding: every page maps to one /P element.
// Heading classification, figure alt text, list and table semantics
// land in v0.17.x as the per-block mapping is wired through the
// layout engine.
//
// Calling SetTagged("") with the document already untagged is a
// no-op; calling it on a previously tagged document with lang ==
// "" disables tagging entirely.
func (d *Document) SetTagged(lang string) *Document {
	if d.err != nil {
		return d
	}
	if lang == "" && !d.tagged {
		return d
	}
	d.tagged = true
	d.taggedLang = lang
	return d
}

// DisableTagging removes any prior SetTagged opt-in. Convenience for
// the rare callers that build documents through a templating layer
// that conditionally tags.
func (d *Document) DisableTagging() *Document {
	if d.err != nil {
		return d
	}
	d.tagged = false
	d.taggedLang = ""
	return d
}

// Tagged reports whether SetTagged was called and returns the
// configured language code. The render bridge consults this to
// populate pdf.Document.Tagged / pdf.Document.Lang.
func (d *Document) Tagged() (lang string, ok bool) {
	if !d.tagged {
		return "", false
	}
	return d.taggedLang, true
}
