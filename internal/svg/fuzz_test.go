package svg

import "testing"

// FuzzConvert drives the SVG → PDF Form converter with random and
// corpus-derived input. Conversion may return errors but must not
// panic — SVG inputs come from user templates / external assets,
// so a panic is a denial-of-service vector for any deployment that
// renders user-supplied SVG.
func FuzzConvert(f *testing.F) {
	for _, seed := range []string{
		`<svg width="50" height="50"><rect x="5" y="5" width="40" height="40" fill="black"/></svg>`,
		`<svg viewBox="0 0 10 10"><circle cx="5" cy="5" r="3"/></svg>`,
		`<svg width="1" height="1"></svg>`,
		`<svg><line x1="0" y1="0" x2="10" y2="10"/></svg>`,
		`<svg><path d="M 0 0 L 10 10 Z"/></svg>`,
		`<svg><g fill="red"><rect width="10" height="10"/></g></svg>`,
		``,
		`not xml`,
		`<svg`,                                // malformed
		`<svg>` + repeat(`<rect/>`, 200) + `</svg>`, // many shapes
		`<svg><path d="M ` + repeat("0 ", 1000) + `Z"/></svg>`, // long path
	} {
		f.Add([]byte(seed))
	}
	f.Fuzz(func(t *testing.T, src []byte) {
		// Cap input size — fuzz engines can grow inputs into the MB
		// range, and the converter is allowed to reject (return
		// err) anything pathological. We're guarding against panic,
		// not against ill-shaped input.
		if len(src) > 64*1024 {
			return
		}
		_, _, _, _ = Convert(src)
	})
}

func repeat(s string, n int) string {
	b := make([]byte, 0, len(s)*n)
	for i := 0; i < n; i++ {
		b = append(b, s...)
	}
	return string(b)
}
