// Invoice demonstrates Kardec's templating layer: a single Markdown
// template renders one PDF per record. Useful for invoices, certificates,
// monthly statements — anywhere the structure repeats and only the data
// changes.
//
// Run from the repository root:
//
//	go run ./examples/invoice
//
// Three invoices are produced (invoice-A-1001.pdf, invoice-A-1002.pdf,
// invoice-A-1003.pdf) in the working directory.
package main

import (
	"fmt"
	"log"

	"github.com/arthurhrc/kardec"
	_ "github.com/arthurhrc/kardec/render"
)

const tpl = `# Invoice {{.ID}}

**Customer:** {{.Customer}}
**Date:** {{.Date}}

## Line items
{{range .Items}}
- {{.Name}} — R$ {{.Price}}
{{end}}

---

**Subtotal:** R$ {{.Subtotal}}
**Tax (10%):** R$ {{.Tax}}
**Total:** R$ {{.Total}}

*Thanks for your business.*
`

type item struct {
	Name  string
	Price int
}

type invoice struct {
	ID       string
	Customer string
	Date     string
	Items    []item
	Subtotal int
	Tax      int
	Total    int
}

func main() {
	t, err := kardec.NewTemplate(tpl)
	if err != nil {
		log.Fatalf("parse template: %v", err)
	}

	invoices := []invoice{
		newInvoice("A-1001", "ACME Corporation", "2026-05-01",
			item{"Cloud subscription", 1200},
			item{"Support hours (10)", 800},
		),
		newInvoice("A-1002", "Globex S.A.", "2026-05-03",
			item{"Annual licence", 4500},
		),
		newInvoice("A-1003", "Initech Ltda.", "2026-05-05",
			item{"Onsite training", 2200},
			item{"Travel allowance", 600},
			item{"Custom integration", 3000},
		),
	}

	for _, inv := range invoices {
		doc, err := t.Render(inv)
		if err != nil {
			log.Fatalf("render template for %s: %v", inv.ID, err)
		}
		path := fmt.Sprintf("invoice-%s.pdf", inv.ID)
		if err := doc.Render(path); err != nil {
			log.Fatalf("write %s: %v", path, err)
		}
		fmt.Printf("rendered %s\n", path)
	}
}

func newInvoice(id, customer, date string, items ...item) invoice {
	subtotal := 0
	for _, it := range items {
		subtotal += it.Price
	}
	tax := subtotal / 10
	return invoice{
		ID:       id,
		Customer: customer,
		Date:     date,
		Items:    items,
		Subtotal: subtotal,
		Tax:      tax,
		Total:    subtotal + tax,
	}
}
