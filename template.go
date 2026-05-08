package kardec

import (
	"bytes"
	"fmt"
	"text/template"
)

// Template is a Markdown source with `text/template` placeholders. Calling
// Render(data) executes the template against data, parses the resulting
// Markdown, and returns a fresh Document ready to render to PDF.
//
// Typical usage: hold one Template, generate many Documents from a slice
// of records (one invoice per customer, one report per month, etc.).
//
//	tpl, err := kardec.NewTemplate(`
//	# Invoice {{.ID}}
//
//	**Customer:** {{.Customer.Name}}
//	**Total:** R$ {{.Total}}
//	`)
//	if err != nil { return err }
//	doc, err := tpl.Render(invoice)
//	if err != nil { return err }
//	doc.Render("invoice.pdf")
//
// A Template is safe for concurrent use; an underlying *Document is not.
type Template struct {
	tpl  *template.Template
	opts templateOptions
}

// templateOptions holds the configurable bits of a Template. Surfaced
// through TemplateOption functional options so the constructor stays
// stable as new knobs land.
type templateOptions struct {
	pageSize PageSize
	margins  Margins
	name     string
}

// TemplateOption customizes a Template at construction time.
type TemplateOption func(*templateOptions)

// WithPageSize overrides the default page size (A4) used by documents
// produced from the template.
func WithPageSize(size PageSize) TemplateOption {
	return func(o *templateOptions) { o.pageSize = size }
}

// WithMargins overrides the default margins (Normal) used by documents
// produced from the template.
func WithMargins(m Margins) TemplateOption {
	return func(o *templateOptions) { o.margins = m }
}

// WithName labels the underlying text/template instance, surfacing the
// chosen name in error messages produced during Execute.
func WithName(name string) TemplateOption {
	return func(o *templateOptions) { o.name = name }
}

// NewTemplate parses src as a Markdown template. Any text/template syntax
// errors are returned eagerly; field references that do not resolve are
// only detected at Render time.
func NewTemplate(src string, opts ...TemplateOption) (*Template, error) {
	o := templateOptions{
		pageSize: PageA4,
		margins:  MarginsNormal,
		name:     "kardec.Template",
	}
	for _, opt := range opts {
		opt(&o)
	}
	tpl, err := template.New(o.name).Parse(src)
	if err != nil {
		return nil, fmt.Errorf("kardec: parse template: %w", err)
	}
	return &Template{tpl: tpl, opts: o}, nil
}

// MustNewTemplate is the panicking variant of NewTemplate, useful for
// templates declared at package init where a failure must be fatal.
func MustNewTemplate(src string, opts ...TemplateOption) *Template {
	t, err := NewTemplate(src, opts...)
	if err != nil {
		panic(err)
	}
	return t
}

// Render executes the template against data, then builds a Document by
// feeding the rendered Markdown to AppendMarkdown. Errors in template
// execution are returned; errors in Markdown parsing surface through
// Document.Err.
func (t *Template) Render(data any) (*Document, error) {
	var buf bytes.Buffer
	if err := t.tpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("kardec: execute template: %w", err)
	}
	doc := New(t.opts.pageSize, t.opts.margins).AppendMarkdown(buf.String())
	return doc, nil
}
