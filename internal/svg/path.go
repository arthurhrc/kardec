package svg

import (
	"fmt"
	"strconv"
	"strings"
)

// pathCmd is one parsed SVG path-data command. op is the canonical
// uppercase command letter (we resolve relative→absolute during
// parse, so the converter only sees absolute coordinates).
type pathCmd struct {
	op     byte
	coords []float64
}

// parsePath tokenises and normalises an SVG <path d="..."> string.
// Supported commands: M/m, L/l, H/h, V/v, C/c, Q/q, Z/z. Other
// commands (A, S, T) are silently dropped — adding them is a
// follow-up if real-world content needs them.
//
// Relative variants (lowercase letters) are resolved into absolute
// coordinates so emitPath in svg.go only handles M/L/H/V/C/Q/Z.
func parsePath(d string) ([]pathCmd, error) {
	tokens := tokenisePath(d)
	if len(tokens) == 0 {
		return nil, nil
	}
	var out []pathCmd
	var cx, cy float64        // current point in absolute coords
	var subStartX, subStartY float64 // start of current subpath (for Z)
	var lastCmd byte
	i := 0
	for i < len(tokens) {
		tk := tokens[i]
		if tk.kind != tkCmd {
			// Implicit command repeat: spec says repeated coordinate
			// sets after a command continue with the same command,
			// except after M which becomes implicit L (or l after m).
			if lastCmd == 'M' {
				lastCmd = 'L'
			} else if lastCmd == 'm' {
				lastCmd = 'l'
			}
			tk = pathToken{kind: tkCmd, op: lastCmd}
		} else {
			i++
		}
		op := tk.op
		lastCmd = op
		isRel := op >= 'a' && op <= 'z'
		upper := op
		if isRel {
			upper = op - 32
		}
		argc := pathArgc(upper)
		if argc < 0 {
			// Unknown command — skip its position-numbers until we
			// hit another letter. Defensive against rare commands
			// (A, S, T) so the rest of the path still renders.
			for i < len(tokens) && tokens[i].kind != tkCmd {
				i++
			}
			continue
		}
		coords := make([]float64, argc)
		for j := 0; j < argc; j++ {
			if i >= len(tokens) || tokens[i].kind != tkNum {
				return nil, fmt.Errorf("svg: path %q expected coord, got end of stream", d)
			}
			coords[j] = tokens[i].num
			i++
		}
		// Resolve relative → absolute. The math depends on which
		// command — H is 1D in x, V is 1D in y, others are pairs.
		switch upper {
		case 'M', 'L', 'C', 'Q':
			if isRel {
				for k := 0; k < len(coords); k += 2 {
					coords[k] += cx
					coords[k+1] += cy
				}
			}
			// New current point = last coord pair.
			cx = coords[len(coords)-2]
			cy = coords[len(coords)-1]
			if upper == 'M' {
				subStartX, subStartY = cx, cy
			}
		case 'H':
			if isRel {
				coords[0] += cx
			}
			cx = coords[0]
		case 'V':
			if isRel {
				coords[0] += cy
			}
			cy = coords[0]
		case 'Z':
			cx, cy = subStartX, subStartY
		}
		out = append(out, pathCmd{op: upper, coords: coords})
	}
	return out, nil
}

// pathArgc returns the coordinate count consumed per absolute
// command instance (0 for Z).
func pathArgc(op byte) int {
	switch op {
	case 'M', 'L':
		return 2
	case 'H', 'V':
		return 1
	case 'C':
		return 6
	case 'Q':
		return 4
	case 'Z':
		return 0
	}
	return -1
}

const (
	tkCmd = iota
	tkNum
)

type pathToken struct {
	kind int
	op   byte
	num  float64
}

// tokenisePath splits the path-data string into a flat stream of
// command letters and numbers. Numbers can be embedded back-to-back
// without separators ("10-5" = [10, -5]); the lexer handles that
// the same way splitNums does for points/polygons.
func tokenisePath(d string) []pathToken {
	var out []pathToken
	var cur strings.Builder
	flushNum := func() {
		t := strings.TrimSpace(cur.String())
		cur.Reset()
		if t == "" {
			return
		}
		if v, err := strconv.ParseFloat(t, 64); err == nil {
			out = append(out, pathToken{kind: tkNum, num: v})
		}
	}
	for i := 0; i < len(d); i++ {
		c := d[i]
		switch {
		case (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z'):
			flushNum()
			out = append(out, pathToken{kind: tkCmd, op: c})
		case c == ',' || c == ' ' || c == '\t' || c == '\n' || c == '\r':
			flushNum()
		case c == '-' || c == '+':
			s := cur.String()
			// Sign starts a new number unless we're inside an
			// exponent ("1e-3").
			if cur.Len() > 0 && s[len(s)-1] != 'e' && s[len(s)-1] != 'E' {
				flushNum()
			}
			cur.WriteByte(c)
		case c == '.':
			// Two dots in a row signal a new number ("1.5.6" =
			// [1.5, 0.6]).
			s := cur.String()
			if strings.Contains(s, ".") {
				flushNum()
			}
			cur.WriteByte(c)
		default:
			cur.WriteByte(c)
		}
	}
	flushNum()
	return out
}
