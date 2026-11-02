package tui

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/de-hv/dehv/internal/ui"
)

// column of a table. Columns with a higher drop give way first when the pane
// is narrow; 0 never does. The last column takes what is left and is cut.
type column struct {
	title string
	drop  int
}

// row is one line: a key that survives a reload, the cells, already styled,
// and the text a filter matches.
type row struct {
	key    string
	cells  []string
	search string
}

// table is a pane's list: a selection that follows the key across reloads,
// a / filter, scrolling, and columns that fold away when it is narrow.
type table struct {
	cols      []column
	rows      []row
	view      []int // indexes of rows the filter keeps
	cur       int   // in view
	off       int
	filter    textinput.Model
	filtering bool
}

func newTable(cols ...column) table {
	f := newInput("/ ", "filter")
	return table{cols: cols, filter: f}
}

// setRows replaces the rows, keeping the selected one when it is still there.
func (t *table) setRows(rows []row) {
	key := ""
	if r, ok := t.selected(); ok {
		key = r.key
	}
	t.rows = rows
	t.refilter()
	for i, v := range t.view {
		if t.rows[v].key == key {
			t.cur = i
			return
		}
	}
	t.cur = min(t.cur, max(0, len(t.view)-1))
}

func (t *table) refilter() {
	q := strings.ToLower(strings.TrimSpace(t.filter.Value()))
	t.view = t.view[:0]
	for i, r := range t.rows {
		if q == "" || strings.Contains(strings.ToLower(r.search), q) {
			t.view = append(t.view, i)
		}
	}
	t.cur = min(t.cur, max(0, len(t.view)-1))
}

func (t *table) selected() (row, bool) {
	if t.cur < 0 || t.cur >= len(t.view) {
		return row{}, false
	}
	return t.rows[t.view[t.cur]], true
}

// filtered says a filter is narrowing the rows.
func (t *table) filtered() bool { return t.filter.Value() != "" }

// key moves the selection or edits the filter; false when it is not a key
// the table uses.
func (t *table) key(msg tea.KeyPressMsg) (bool, tea.Cmd) {
	if t.filtering {
		switch msg.String() {
		case "esc":
			t.filtering = false
			t.filter.SetValue("")
			t.filter.Blur()
			t.refilter()
			return true, nil
		case "enter", "down", "up":
			t.filtering = false
			t.filter.Blur()
			if msg.String() == "enter" {
				return true, nil
			}
		default:
			var cmd tea.Cmd
			t.filter, cmd = t.filter.Update(msg)
			t.refilter()
			return true, cmd
		}
	}
	switch msg.String() {
	case "up", "k":
		t.cur = max(0, t.cur-1)
	case "down", "j":
		t.cur = min(max(0, len(t.view)-1), t.cur+1)
	case "pgup", "ctrl+u":
		t.cur = max(0, t.cur-10)
	case "pgdown", "ctrl+d":
		t.cur = min(max(0, len(t.view)-1), t.cur+10)
	case "home", "g":
		t.cur = 0
	case "end", "G":
		t.cur = max(0, len(t.view)-1)
	case "/":
		t.filtering = true
		return true, t.filter.Focus()
	case "esc":
		if t.filtered() {
			t.filter.SetValue("")
			t.refilter()
			return true, nil
		}
		return false, nil
	default:
		return false, nil
	}
	return true, nil
}

// render draws it in w×h cells: the header, the rows, and the filter line
// while there is one. Without the focus, the selected row keeps only a dim
// bar.
func (t *table) render(w, h int, focused bool) string {
	showFilter := t.filtering || t.filtered()
	if showFilter {
		h--
	}
	// Each column as wide as its widest cell; drop the most droppable ones
	// until they fit, the last one is cut to what is left.
	keep := make([]bool, len(t.cols))
	widths := make([]int, len(t.cols))
	for i, c := range t.cols {
		keep[i] = true
		widths[i] = lipgloss.Width(c.title)
	}
	for _, r := range t.rows {
		for i := range t.cols {
			if i < len(r.cells) {
				widths[i] = max(widths[i], lipgloss.Width(r.cells[i]))
			}
		}
	}
	avail := w - 2 // the selection bar and a space
	// The last column is cut to what is left, so it only asks for a little
	// room: a long description never pushes the others out.
	const lastMin = 24
	total := func() int {
		n, last := 0, -1
		for i := range t.cols {
			if keep[i] {
				n += widths[i] + 2
				last = i
			}
		}
		if last >= 0 && widths[last] > lastMin {
			n -= widths[last] - lastMin
		}
		return n - 2
	}
	for total() > avail {
		worst := -1
		for i, c := range t.cols {
			// Among equals the rightmost goes first.
			if keep[i] && c.drop > 0 && (worst < 0 || c.drop >= t.cols[worst].drop) {
				worst = i
			}
		}
		if worst < 0 {
			break
		}
		keep[worst] = false
	}
	line := func(cells []string) string {
		var b strings.Builder
		last := -1
		for i := range t.cols {
			if keep[i] {
				last = i
			}
		}
		for i := range t.cols {
			if !keep[i] {
				continue
			}
			c := ""
			if i < len(cells) {
				c = cells[i]
			}
			if i == last {
				b.WriteString(c)
			} else {
				b.WriteString(ui.Pad(c, widths[i]) + "  ")
			}
		}
		return ui.Cut(b.String(), avail)
	}

	var out []string
	titles := make([]string, len(t.cols))
	for i, c := range t.cols {
		titles[i] = c.title
	}
	out = append(out, "  "+ui.Dim.Render(line(titles)))

	rowsH := max(0, h-1)
	if t.cur < t.off {
		t.off = t.cur
	}
	if t.cur >= t.off+rowsH {
		t.off = t.cur - rowsH + 1
	}
	t.off = max(0, min(t.off, max(0, len(t.view)-rowsH)))
	for i := t.off; i < len(t.view) && i < t.off+rowsH; i++ {
		r := t.rows[t.view[i]]
		l := line(r.cells)
		if i == t.cur && !focused {
			out = append(out, ui.Dim.Render("▌")+" "+l)
		} else if i == t.cur {
			plain := ui.Pad(ansi.Strip(l), avail)
			out = append(out, barStyle.Render("▌")+" "+selStyle.Render(plain))
		} else {
			out = append(out, "  "+l)
		}
	}
	if len(t.view) == 0 && len(t.rows) > 0 {
		out = append(out, "  "+ui.Dim.Render("nothing matches · esc clears the filter"))
	}
	for len(out) < h {
		out = append(out, "")
	}
	if showFilter {
		out = append(out, t.filter.View())
	}
	return strings.Join(out, "\n")
}
