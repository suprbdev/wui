//go:build !js

package wui

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	ltable "github.com/charmbracelet/lipgloss/table"
	zone "github.com/lrstanley/bubblezone"
)

// ansiPalette maps ANSI color indices 0-15 to lipgloss colors.
var ansiPalette = [16]string{
	"0", "1", "2", "3", "4", "5", "6", "7",
	"8", "9", "10", "11", "12", "13", "14", "15",
}

// tuiRenderer walks an Element tree and produces an ANSI string. It is
// long-lived across renders so that stateful sub-widgets (text inputs)
// retain cursor position and blink state between frames.
type tuiRenderer struct {
	Width, Height int
	FocusedID     string

	inputs map[string]*tuiInputState
	zones  *zone.Manager
}

// tuiInputState pairs a live bubbles text input with the spec Value it
// was last synced from, so programmatic value changes by the app (e.g.
// clearing a field after submit) can be told apart from user typing.
type tuiInputState struct {
	model    textinput.Model
	lastSpec string
}

func newTUIRenderer(width, height int, zones *zone.Manager) *tuiRenderer {
	return &tuiRenderer{
		Width:  width,
		Height: height,
		inputs: make(map[string]*tuiInputState),
		zones:  zones,
	}
}

func (r *tuiRenderer) Render(el Element) string {
	return r.renderEl(el)
}

// focusable describes one element the Tab ring can land on. Inputs
// are edited in place by routeKeyToFocused; buttons and links activate
// via their Activate closure when Enter is pressed. Form points at the
// nearest enclosing FormEl (nil outside forms) so that Enter can fall
// back to submitting the form — a button with no OnClick inside a form
// acts as a submit button, mirroring `<button type="submit">` in HTML.
type focusable struct {
	ID       string
	IsInput  bool
	Activate func() Msg // nil for inputs and submit buttons
	Form     *FormEl
}

// collectFocusables walks the tree and returns focus-ring entries in
// tree order: text inputs, buttons, and links.
func collectFocusables(el Element) []focusable {
	var out []focusable
	var walk func(Element, *FormEl)
	walk = func(e Element, form *FormEl) {
		switch v := e.(type) {
		case TextInputEl:
			if v.ID != "" {
				out = append(out, focusable{ID: v.ID, IsInput: true, Form: form})
			}
		case ButtonEl:
			if v.Disabled {
				return
			}
			if v.OnClick != nil {
				out = append(out, focusable{ID: buttonFocusKey(v), Activate: v.OnClick, Form: form})
			} else if form != nil && form.OnSubmit != nil {
				out = append(out, focusable{ID: buttonFocusKey(v), Form: form})
			}
		case CheckboxEl:
			if v.Disabled {
				return
			}
			cb := v
			out = append(out, focusable{ID: cb.ID, Activate: func() Msg { return checkboxToggleMsg(cb, !cb.Checked) }, Form: form})
		case LinkEl:
			id := "link:" + v.Href
			out = append(out, focusable{ID: id, Activate: func() Msg { return ClickMsg{TargetID: v.Href} }})
		case CardEl:
			walk(v.Child, form)
		case BoxEl:
			for _, c := range v.Children {
				walk(c, form)
			}
		case FormEl:
			f := v
			for _, c := range v.Children {
				walk(c, &f)
			}
		case ListEl:
			for _, c := range v.Items {
				walk(c, form)
			}
		case ScrollAreaEl:
			walk(v.Child, form)
		}
	}
	walk(el, nil)
	return out
}

// formValues collects the current value of every text input inside the
// form, preferring the live bubbles model (in-progress typing) over
// the spec value — the TUI analogue of reading <input> values on
// form submission in the browser.
func (r *tuiRenderer) formValues(form FormEl) map[string]string {
	values := make(map[string]string)
	var walk func(Element)
	walk = func(e Element) {
		switch v := e.(type) {
		case TextInputEl:
			if v.ID == "" {
				return
			}
			if st, ok := r.inputs[v.ID]; ok {
				values[v.ID] = st.model.Value()
			} else {
				values[v.ID] = v.Value
			}
		case BoxEl:
			for _, c := range v.Children {
				walk(c)
			}
		case FormEl:
			for _, c := range v.Children {
				walk(c)
			}
		case ListEl:
			for _, c := range v.Items {
				walk(c)
			}
		case ScrollAreaEl:
			walk(v.Child)
		case CardEl:
			walk(v.Child)
		}
	}
	for _, c := range form.Children {
		walk(c)
	}
	return values
}

func focusableIDs(items []focusable) []string {
	ids := make([]string, len(items))
	for i, it := range items {
		ids[i] = it.ID
	}
	return ids
}

func findInputByID(el Element, id string) *TextInputEl {
	switch v := el.(type) {
	case TextInputEl:
		if v.ID == id {
			return &v
		}
	case BoxEl:
		for _, c := range v.Children {
			if found := findInputByID(c, id); found != nil {
				return found
			}
		}
	case FormEl:
		for _, c := range v.Children {
			if found := findInputByID(c, id); found != nil {
				return found
			}
		}
	case ListEl:
		for _, c := range v.Items {
			if found := findInputByID(c, id); found != nil {
				return found
			}
		}
	case ScrollAreaEl:
		return findInputByID(v.Child, id)
	case CardEl:
		return findInputByID(v.Child, id)
	}
	return nil
}

// ensureInput creates (if absent) and syncs placeholder/value for the
// input with the given spec, returning the live bubbles model. The
// value is re-synced only when the spec value differs from the spec
// value last seen — i.e. when the app changed it programmatically —
// so in-progress typing is never clobbered by re-renders.
func (r *tuiRenderer) ensureInput(spec TextInputEl) *textinput.Model {
	st, ok := r.inputs[spec.ID]
	if !ok {
		m := textinput.New()
		// No "> " prompt: the HTML renderer draws a bare <input>, so a
		// TUI-only prompt would make the two platforms diverge.
		m.Prompt = ""
		m.Placeholder = spec.Placeholder
		if spec.Password {
			m.EchoMode = textinput.EchoPassword
		}
		m.SetValue(spec.Value)
		st = &tuiInputState{model: m, lastSpec: spec.Value}
		r.inputs[spec.ID] = st
		return &st.model
	}
	st.model.Placeholder = spec.Placeholder
	if spec.Value != st.lastSpec {
		// Avoid SetValue when the live value already matches (the
		// common OnChange round-trip) — it would move the cursor to
		// the end of the line mid-edit.
		if spec.Value != st.model.Value() {
			st.model.SetValue(spec.Value)
		}
		st.lastSpec = spec.Value
	}
	return &st.model
}

// setInput replaces the stored bubbles model for the given input ID.
// textinput.Model.Update returns an updated value rather than mutating
// in place, so callers must write the result back via this method.
func (r *tuiRenderer) setInput(id string, m textinput.Model) {
	if st, ok := r.inputs[id]; ok {
		st.model = m
		return
	}
	r.inputs[id] = &tuiInputState{model: m}
}

func (r *tuiRenderer) renderEl(el Element) string {
	switch e := el.(type) {
	case TextEl:
		return r.renderText(e)
	case BoxEl:
		return r.renderBox(e)
	case ButtonEl:
		return r.renderButton(e)
	case TextInputEl:
		return r.renderTextInput(e)
	case CheckboxEl:
		return r.renderCheckbox(e)
	case CardEl:
		return r.renderCard(e)
	case FormEl:
		return r.renderForm(e)
	case ListEl:
		return r.renderList(e)
	case ScrollAreaEl:
		return r.renderScroll(e)
	case LinkEl:
		return r.renderLink(e)
	case TableEl:
		return r.renderTable(e)
	default:
		return ""
	}
}

func (r *tuiRenderer) renderText(e TextEl) string {
	return styleToLipgloss(e.Style).Render(e.Content)
}

func (r *tuiRenderer) renderBox(e BoxEl) string {
	parts := make([]string, 0, len(e.Children))
	for _, c := range e.Children {
		parts = append(parts, r.renderEl(c))
	}
	if e.Gap > 0 {
		gap := strings.Repeat(" ", e.Gap)
		if e.Direction == Column {
			gap = strings.Repeat("\n", e.Gap)
		}
		joined := make([]string, 0, len(parts)*2-1)
		for i, p := range parts {
			if i > 0 {
				joined = append(joined, gap)
			}
			joined = append(joined, p)
		}
		parts = joined
	}
	var out string
	if e.Direction == Row {
		out = lipgloss.JoinHorizontal(alignToPosition(e.Align), parts...)
	} else {
		out = lipgloss.JoinVertical(alignToPosition(e.Align), parts...)
	}
	return styleToLipgloss(e.Style).Render(out)
}

// alignToPosition maps Align onto a lipgloss join position: the
// cross-axis placement of children (top/left for AlignStart, etc.).
// AlignStretch has no terminal equivalent and behaves as AlignStart.
func alignToPosition(a Align) lipgloss.Position {
	switch a {
	case AlignCenter:
		return lipgloss.Center
	case AlignEnd:
		return lipgloss.Bottom // == lipgloss.Right (both 1.0)
	default:
		return lipgloss.Top // == lipgloss.Left (both 0.0)
	}
}

func (r *tuiRenderer) renderButton(e ButtonEl) string {
	label := "[ " + e.Label + " ]"
	style := styleToLipgloss(e.Style)
	if e.Disabled {
		style = style.Faint(true)
		return style.Render(label)
	}
	if r.FocusedID == buttonFocusKey(e) {
		style = style.Reverse(true)
	}
	return r.zones.Mark(buttonFocusKey(e), style.Render(label))
}

func (r *tuiRenderer) renderTextInput(e TextInputEl) string {
	in := r.ensureInput(e)
	focused := r.FocusedID == e.ID
	if focused && !in.Focused() {
		in.Focus()
	} else if !focused && in.Focused() {
		in.Blur()
	}
	style := styleToLipgloss(e.Style)
	out := style.Render(in.View())
	if e.ID != "" {
		out = r.zones.Mark(e.ID, out)
	}
	return out
}

func (r *tuiRenderer) renderCheckbox(e CheckboxEl) string {
	mark := "[ ]"
	if e.Checked {
		mark = "[x]"
	}
	label := mark
	if e.Label != "" {
		label += " " + e.Label
	}
	style := styleToLipgloss(e.Style)
	if e.Disabled {
		return style.Faint(true).Render(label)
	}
	if r.FocusedID == e.ID {
		style = style.Reverse(true)
	}
	return r.zones.Mark(e.ID, style.Render(label))
}

// renderCard draws the child inside a rounded border with the title
// embedded in the top border line:
//
//	╭ Title ────╮
//	│ content   │
//	╰───────────╯
//
// The border is drawn by hand (lipgloss borders cannot host a title).
// From e.Style: Padding and Width shape the interior, Margin wraps the
// finished box, BorderColor colors the frame. Title text is bold.
func (r *tuiRenderer) renderCard(e CardEl) string {
	inner := r.renderEl(e.Child)
	innerStyle := lipgloss.NewStyle().
		Padding(e.Style.Padding[0], e.Style.Padding[1], e.Style.Padding[2], e.Style.Padding[3])
	if e.Style.Width > 2 {
		innerStyle = innerStyle.Width(e.Style.Width - 2)
	}
	inner = innerStyle.Render(inner)

	title := ""
	if e.Title != "" {
		title = lipgloss.NewStyle().Bold(true).Render(" " + e.Title + " ")
	}
	tw := lipgloss.Width(title)
	w := lipgloss.Width(inner)
	if w < tw+2 {
		w = tw + 2
	}
	// Pad every line to the box width so the right border aligns.
	inner = lipgloss.NewStyle().Width(w).Render(inner)

	bs := lipgloss.NewStyle()
	if e.Style.BorderColor != "" {
		bs = bs.Foreground(lipgloss.Color(resolveColor(e.Style.BorderColor)))
	}
	b := lipgloss.RoundedBorder()

	var sb strings.Builder
	sb.WriteString(bs.Render(b.TopLeft + b.Top))
	sb.WriteString(title)
	sb.WriteString(bs.Render(strings.Repeat(b.Top, w-tw-1) + b.TopRight))
	for _, line := range strings.Split(inner, "\n") {
		sb.WriteString("\n" + bs.Render(b.Left) + line + bs.Render(b.Right))
	}
	sb.WriteString("\n" + bs.Render(b.BottomLeft+strings.Repeat(b.Bottom, w)+b.BottomRight))

	out := sb.String()
	if e.Style.Margin != [4]int{} {
		out = lipgloss.NewStyle().
			Margin(e.Style.Margin[0], e.Style.Margin[1], e.Style.Margin[2], e.Style.Margin[3]).
			Render(out)
	}
	return out
}

func (r *tuiRenderer) renderForm(e FormEl) string {
	parts := make([]string, 0, len(e.Children))
	for _, c := range e.Children {
		parts = append(parts, r.renderEl(c))
	}
	out := lipgloss.JoinVertical(lipgloss.Left, parts...)
	return styleToLipgloss(e.Style).Render(out)
}

func (r *tuiRenderer) renderList(e ListEl) string {
	lines := make([]string, 0, len(e.Items))
	for i, item := range e.Items {
		prefix := "- "
		if e.Ordered {
			prefix = strconv.Itoa(i+1) + ". "
		}
		lines = append(lines, prefix+r.renderEl(item))
	}
	out := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return styleToLipgloss(e.Style).Render(out)
}

func (r *tuiRenderer) renderScroll(e ScrollAreaEl) string {
	inner := r.renderEl(e.Child)
	style := styleToLipgloss(e.Style)
	if e.MaxHeight > 0 {
		style = style.MaxHeight(e.MaxHeight)
	}
	return style.Render(inner)
}

func (r *tuiRenderer) renderLink(e LinkEl) string {
	style := styleToLipgloss(e.Style).Underline(true)
	if r.FocusedID == "link:"+e.Href {
		style = style.Reverse(true)
	}
	return r.zones.Mark("link:"+e.Href, style.Render(e.Label))
}

func (r *tuiRenderer) renderTable(e TableEl) string {
	t := ltable.New().Headers(e.Columns...).Rows(e.Rows...)
	return t.Render()
}

func styleToLipgloss(s Style) lipgloss.Style {
	ls := lipgloss.NewStyle()
	if s.FG != "" {
		ls = ls.Foreground(lipgloss.Color(resolveColor(s.FG)))
	}
	if s.BG != "" {
		ls = ls.Background(lipgloss.Color(resolveColor(s.BG)))
	}
	if s.Bold {
		ls = ls.Bold(true)
	}
	if s.Italic {
		ls = ls.Italic(true)
	}
	if s.Underline {
		ls = ls.Underline(true)
	}
	if s.Width > 0 {
		ls = ls.Width(s.Width)
	}
	if s.Height > 0 {
		ls = ls.Height(s.Height)
	}
	ls = ls.Padding(s.Padding[0], s.Padding[1], s.Padding[2], s.Padding[3])
	ls = ls.Margin(s.Margin[0], s.Margin[1], s.Margin[2], s.Margin[3])
	if s.Border {
		ls = ls.Border(lipgloss.RoundedBorder())
		if s.BorderColor != "" {
			ls = ls.BorderForeground(lipgloss.Color(resolveColor(s.BorderColor)))
		}
	}
	return ls
}

func resolveColor(c Color) string {
	if idx, err := strconv.Atoi(string(c)); err == nil && idx >= 0 && idx < 16 {
		return ansiPalette[idx]
	}
	return string(c)
}
