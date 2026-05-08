package math

import (
	"reflect"
	"testing"
)

// TestParseSimple covers the leaf token kinds: identifiers, numbers and
// operators in isolation.
func TestParseSimple(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want Expr
	}{
		{"identifier", "x", Identifier{Name: "x"}},
		{"number int", "42", Number{Value: "42"}},
		{"number decimal", "3.14", Number{Value: "3.14"}},
		{"operator plus", "+", Op{Symbol: "+"}},
		{"operator equals", "=", Op{Symbol: "="}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Parse(tc.src)
			if err != nil {
				t.Fatalf("Parse(%q) err = %v", tc.src, err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("Parse(%q) = %#v, want %#v", tc.src, got, tc.want)
			}
		})
	}
}

// TestParseExpression covers a multi-term sequence flowing through Group
// when not wrapped in braces.
func TestParseExpression(t *testing.T) {
	got, err := Parse("a + b")
	if err != nil {
		t.Fatalf("Parse err = %v", err)
	}
	want := Group{Children: []Expr{
		Identifier{Name: "a"},
		Op{Symbol: "+"},
		Identifier{Name: "b"},
	}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Parse(\"a + b\") = %#v, want %#v", got, want)
	}
}

// TestParseGroup covers brace-delimited groups.
func TestParseGroup(t *testing.T) {
	got, err := Parse("{abc}")
	if err != nil {
		t.Fatalf("Parse err = %v", err)
	}
	want := Group{Children: []Expr{
		Identifier{Name: "a"},
		Identifier{Name: "b"},
		Identifier{Name: "c"},
	}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Parse(\"{abc}\") = %#v, want %#v", got, want)
	}
}

// TestParseFrac covers \frac and \dfrac, including a nested fraction.
func TestParseFrac(t *testing.T) {
	got, err := Parse("\\frac{1}{2}")
	if err != nil {
		t.Fatalf("Parse err = %v", err)
	}
	want := Frac{Numerator: Number{Value: "1"}, Denominator: Number{Value: "2"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("\\frac{1}{2} = %#v, want %#v", got, want)
	}

	got, err = Parse("\\dfrac{a+b}{c}")
	if err != nil {
		t.Fatalf("Parse err = %v", err)
	}
	want = Frac{
		Numerator: Group{Children: []Expr{
			Identifier{Name: "a"},
			Op{Symbol: "+"},
			Identifier{Name: "b"},
		}},
		Denominator: Identifier{Name: "c"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("\\dfrac{a+b}{c} = %#v, want %#v", got, want)
	}

	got, err = Parse("\\frac{\\frac{a}{b}}{c}")
	if err != nil {
		t.Fatalf("Parse err = %v", err)
	}
	want = Frac{
		Numerator:   Frac{Numerator: Identifier{Name: "a"}, Denominator: Identifier{Name: "b"}},
		Denominator: Identifier{Name: "c"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("nested frac = %#v, want %#v", got, want)
	}
}

// TestParseScripts covers subscript, superscript and combined scripts in
// either source order.
func TestParseScripts(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want Expr
	}{
		{"sup only", "x^2", SubSup{Base: Identifier{Name: "x"}, Sup: Number{Value: "2"}}},
		{"sub only", "x_i", SubSup{Base: Identifier{Name: "x"}, Sub: Identifier{Name: "i"}}},
		{"sub then sup", "x_i^2", SubSup{
			Base: Identifier{Name: "x"},
			Sub:  Identifier{Name: "i"},
			Sup:  Number{Value: "2"},
		}},
		{"sup then sub", "x^2_i", SubSup{
			Base: Identifier{Name: "x"},
			Sub:  Identifier{Name: "i"},
			Sup:  Number{Value: "2"},
		}},
		{"sup with brace", "x^{n+1}", SubSup{
			Base: Identifier{Name: "x"},
			Sup: Group{Children: []Expr{
				Identifier{Name: "n"},
				Op{Symbol: "+"},
				Number{Value: "1"},
			}},
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Parse(tc.src)
			if err != nil {
				t.Fatalf("Parse(%q) err = %v", tc.src, err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("Parse(%q) = %#v, want %#v", tc.src, got, tc.want)
			}
		})
	}
}

// TestParseGreek covers a small expression mixing greek letters and a
// relational operator.
func TestParseGreek(t *testing.T) {
	got, err := Parse("\\alpha + \\beta = \\gamma")
	if err != nil {
		t.Fatalf("Parse err = %v", err)
	}
	want := Group{Children: []Expr{
		Atom{Symbol: "\\alpha"},
		Op{Symbol: "+"},
		Atom{Symbol: "\\beta"},
		Op{Symbol: "="},
		Atom{Symbol: "\\gamma"},
	}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("greek expression = %#v, want %#v", got, want)
	}
}

// TestParseSqrt covers \sqrt{...} and \sqrt[n]{...}.
func TestParseSqrt(t *testing.T) {
	got, err := Parse("\\sqrt{2}")
	if err != nil {
		t.Fatalf("Parse err = %v", err)
	}
	want := Sqrt{Body: Number{Value: "2"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("\\sqrt{2} = %#v, want %#v", got, want)
	}

	got, err = Parse("\\sqrt[3]{x}")
	if err != nil {
		t.Fatalf("Parse err = %v", err)
	}
	want2 := NthRoot{Index: Number{Value: "3"}, Body: Identifier{Name: "x"}}
	if !reflect.DeepEqual(got, want2) {
		t.Fatalf("\\sqrt[3]{x} = %#v, want %#v", got, want2)
	}
}

// TestParseBigOpSum covers \sum_{i=0}^{n} a_i with both limits and a body.
func TestParseBigOpSum(t *testing.T) {
	got, err := Parse("\\sum_{i=0}^{n} a_i")
	if err != nil {
		t.Fatalf("Parse err = %v", err)
	}
	wantLower := Group{Children: []Expr{
		Identifier{Name: "i"},
		Op{Symbol: "="},
		Number{Value: "0"},
	}}
	wantBody := SubSup{Base: Identifier{Name: "a"}, Sub: Identifier{Name: "i"}}
	want := BigOp{
		Symbol: "\\sum",
		Lower:  wantLower,
		Upper:  Identifier{Name: "n"},
		Body:   wantBody,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("\\sum_... = %#v, want %#v", got, want)
	}
}

// TestParseBigOpInt covers \int_0^1 f(x). The body picks up the next single
// term (the identifier f), with the parenthesised argument trailing as part
// of the surrounding group.
func TestParseBigOpInt(t *testing.T) {
	got, err := Parse("\\int_0^1 f(x)")
	if err != nil {
		t.Fatalf("Parse err = %v", err)
	}
	bigOp := BigOp{
		Symbol: "\\int",
		Lower:  Number{Value: "0"},
		Upper:  Number{Value: "1"},
		Body:   Identifier{Name: "f"},
	}
	want := Group{Children: []Expr{
		bigOp,
		Atom{Symbol: "("},
		Identifier{Name: "x"},
		Atom{Symbol: ")"},
	}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("\\int_0^1 f(x) = %#v, want %#v", got, want)
	}
}

// TestParseUnknownCommand asserts an unknown command yields an Atom with the
// literal command string, so renderers can fall back to plain text.
func TestParseUnknownCommand(t *testing.T) {
	got, err := Parse("\\foobar")
	if err != nil {
		t.Fatalf("Parse err = %v", err)
	}
	want := Atom{Symbol: "\\foobar"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Parse(%q) = %#v, want %#v", "\\foobar", got, want)
	}
}

// TestParseEmpty asserts that an empty source returns an empty Group with no
// error, matching the package contract.
func TestParseEmpty(t *testing.T) {
	got, err := Parse("")
	if err != nil {
		t.Fatalf("Parse err = %v", err)
	}
	want := Group{}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Parse(\"\") = %#v, want %#v", got, want)
	}
}

// TestParseFracError asserts a malformed fraction produces an error.
func TestParseFracError(t *testing.T) {
	if _, err := Parse("\\frac{1}"); err == nil {
		t.Fatalf("Parse(\\frac{1}) err = nil, want error")
	}
}

// TestKindHelper asserts Kind dispatches to the unexported sentinel and
// returns 0 for nil.
func TestKindHelper(t *testing.T) {
	if Kind(nil) != 0 {
		t.Fatalf("Kind(nil) = %d, want 0", Kind(nil))
	}
	if Kind(Number{Value: "1"}) != KindNumber {
		t.Fatalf("Kind(Number) = %d, want KindNumber", Kind(Number{Value: "1"}))
	}
	if Kind(Frac{}) != KindFrac {
		t.Fatalf("Kind(Frac) = %d, want KindFrac", Kind(Frac{}))
	}
}
