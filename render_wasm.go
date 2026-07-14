//go:build js

package wui

import (
	"fmt"
	"strconv"
	"strings"
	"syscall/js"
)

// ansiPalette maps ANSI color indices 0-15 to hex colors for use in
// CSS, since ANSI indices are meaningless to the browser.
var ansiPalette = [16]string{
	"#000000", "#800000", "#008000", "#808000",
	"#000080", "#800080", "#008080", "#c0c0c0",
	"#808080", "#ff0000", "#00ff00", "#ffff00",
	"#0000ff", "#ff00ff", "#00ffff", "#ffffff",
}

// wasmRenderer performs a full-replace render of an Element tree into
// a root DOM node. v1 does no VDOM diffing: each render clears and
// rebuilds the subtree, preserving focus and in-progress input values
// across the replace.
//
// Elements carry their own callbacks (ButtonEl.OnClick, etc.); the
// renderer wires each DOM event directly to the matching callback and
// forwards the resulting Msg to dispatch.
type wasmRenderer struct {
	root     js.Value
	dispatch func(Msg)
	funcs    []js.Func

	// prevValues snapshots live <input> values at the start of each
	// render; lastSpec records the spec Value each input was last
	// rendered with. Together they let renderTextInput distinguish
	// programmatic value changes (respect the new spec) from
	// in-progress typing (preserve the live value) — mirroring the
	// TUI renderer's ensureInput.
	prevValues map[string]string
	lastSpec   map[string]string
}

func newWASMRenderer(root js.Value, dispatch func(Msg)) *wasmRenderer {
	return &wasmRenderer{
		root:     root,
		dispatch: dispatch,
		lastSpec: make(map[string]string),
	}
}

func (r *wasmRenderer) Render(el Element) {
	doc := js.Global().Get("document")

	activeID := ""
	if active := doc.Get("activeElement"); !active.IsNull() && !active.IsUndefined() {
		activeID = active.Get("id").String()
	}
	r.prevValues = r.collectInputValues(doc)

	for _, f := range r.funcs {
		f.Release()
	}
	r.funcs = r.funcs[:0]

	r.root.Set("innerHTML", "")
	node := r.renderEl(el, doc)
	r.root.Call("appendChild", node)

	if activeID != "" {
		if el := doc.Call("getElementById", activeID); !el.IsNull() {
			el.Call("focus")
		}
	}
}

func (r *wasmRenderer) collectInputValues(doc js.Value) map[string]string {
	values := make(map[string]string)
	nodeList := r.root.Call("querySelectorAll", "input")
	length := nodeList.Get("length").Int()
	for i := 0; i < length; i++ {
		n := nodeList.Call("item", i)
		id := n.Get("id").String()
		if id != "" {
			values[id] = n.Get("value").String()
		}
	}
	return values
}

func (r *wasmRenderer) addFunc(f js.Func) js.Func {
	r.funcs = append(r.funcs, f)
	return f
}

func (r *wasmRenderer) renderEl(el Element, doc js.Value) js.Value {
	switch e := el.(type) {
	case TextEl:
		return r.renderText(e, doc)
	case BoxEl:
		return r.renderBox(e, doc)
	case ButtonEl:
		return r.renderButton(e, doc)
	case TextInputEl:
		return r.renderTextInput(e, doc)
	case FormEl:
		return r.renderForm(e, doc)
	case ListEl:
		return r.renderList(e, doc)
	case ScrollAreaEl:
		return r.renderScroll(e, doc)
	case LinkEl:
		return r.renderLink(e, doc)
	case TableEl:
		return r.renderTable(e, doc)
	default:
		return doc.Call("createElement", "span")
	}
}

func (r *wasmRenderer) renderText(e TextEl, doc js.Value) js.Value {
	n := doc.Call("createElement", "span")
	n.Set("textContent", e.Content)
	applyStyle(n, e.Style)
	return n
}

func (r *wasmRenderer) renderBox(e BoxEl, doc js.Value) js.Value {
	n := doc.Call("createElement", "div")
	dir := "row"
	if e.Direction == Column {
		dir = "column"
	}
	css := "display:flex;flex-direction:" + dir + ";"
	if e.Gap > 0 {
		// Gap is terminal cells in the TUI; ch/lh (with an em
		// fallback) are the closest CSS analogues.
		css += fmt.Sprintf("column-gap:%dch;row-gap:%dem;row-gap:%dlh;", e.Gap, e.Gap, e.Gap)
	}
	if a := alignItemsCSS(e.Align); a != "" {
		css += "align-items:" + a + ";"
	}
	n.Set("style", css+styleToCSS(e.Style))
	for _, c := range e.Children {
		n.Call("appendChild", r.renderEl(c, doc))
	}
	return n
}

func (r *wasmRenderer) renderButton(e ButtonEl, doc js.Value) js.Value {
	n := doc.Call("createElement", "button")
	n.Set("textContent", e.Label)
	if e.Disabled {
		n.Set("disabled", true)
	}
	applyStyle(n, e.Style)
	if e.OnClick != nil {
		onClick := e.OnClick
		f := js.FuncOf(func(this js.Value, args []js.Value) any {
			r.dispatch(onClick())
			return nil
		})
		r.addFunc(f)
		n.Call("addEventListener", "click", f)
	}
	return n
}

func (r *wasmRenderer) renderTextInput(e TextInputEl, doc js.Value) js.Value {
	n := doc.Call("createElement", "input")
	n.Set("id", e.ID)
	if e.Password {
		n.Set("type", "password")
	} else {
		n.Set("type", "text")
	}
	// Keep in-progress typing across the full-tree replace unless the
	// app changed the spec value programmatically since last render.
	val := e.Value
	if prev, ok := r.prevValues[e.ID]; ok && r.lastSpec[e.ID] == e.Value {
		val = prev
	}
	r.lastSpec[e.ID] = e.Value
	n.Set("value", val)
	n.Set("placeholder", e.Placeholder)
	if e.Disabled {
		n.Set("disabled", true)
	}
	applyStyle(n, e.Style)

	if e.OnChange != nil {
		onChange := e.OnChange
		inputFn := js.FuncOf(func(this js.Value, args []js.Value) any {
			val := args[0].Get("target").Get("value").String()
			r.dispatch(onChange(val))
			return nil
		})
		r.addFunc(inputFn)
		n.Call("addEventListener", "input", inputFn)
	}
	if e.OnSubmit != nil {
		onSubmit := e.OnSubmit
		keydownFn := js.FuncOf(func(this js.Value, args []js.Value) any {
			if args[0].Get("key").String() == "Enter" {
				// Stop the enclosing form (if any) from also
				// submitting — the input's own handler wins,
				// matching TUI dispatch order.
				args[0].Call("preventDefault")
				val := args[0].Get("target").Get("value").String()
				r.dispatch(onSubmit(val))
			}
			return nil
		})
		r.addFunc(keydownFn)
		n.Call("addEventListener", "keydown", keydownFn)
	}
	return n
}

func (r *wasmRenderer) renderForm(e FormEl, doc js.Value) js.Value {
	n := doc.Call("createElement", "form")
	applyStyle(n, e.Style)
	for _, c := range e.Children {
		n.Call("appendChild", r.renderEl(c, doc))
	}
	if e.OnSubmit != nil {
		formNode := n
		onSubmit := e.OnSubmit
		submitFn := js.FuncOf(func(this js.Value, args []js.Value) any {
			args[0].Call("preventDefault")
			values := collectFormValues(formNode)
			r.dispatch(onSubmit(values))
			return nil
		})
		r.addFunc(submitFn)
		n.Call("addEventListener", "submit", submitFn)
	}
	return n
}

func collectFormValues(formNode js.Value) map[string]string {
	values := make(map[string]string)
	nodeList := formNode.Call("querySelectorAll", "input")
	length := nodeList.Get("length").Int()
	for i := 0; i < length; i++ {
		input := nodeList.Call("item", i)
		id := input.Get("id").String()
		if id != "" {
			values[id] = input.Get("value").String()
		}
	}
	return values
}

func (r *wasmRenderer) renderList(e ListEl, doc js.Value) js.Value {
	tag := "ul"
	if e.Ordered {
		tag = "ol"
	}
	n := doc.Call("createElement", tag)
	applyStyle(n, e.Style)
	for _, item := range e.Items {
		li := doc.Call("createElement", "li")
		li.Call("appendChild", r.renderEl(item, doc))
		n.Call("appendChild", li)
	}
	return n
}

func (r *wasmRenderer) renderScroll(e ScrollAreaEl, doc js.Value) js.Value {
	n := doc.Call("createElement", "div")
	css := "overflow:auto;"
	if e.MaxHeight > 0 {
		css += fmt.Sprintf("max-height:%dpx;", e.MaxHeight)
	}
	n.Set("style", css+styleToCSS(e.Style))
	n.Call("appendChild", r.renderEl(e.Child, doc))
	return n
}

func (r *wasmRenderer) renderLink(e LinkEl, doc js.Value) js.Value {
	n := doc.Call("createElement", "a")
	n.Set("textContent", e.Label)
	n.Set("href", e.Href)
	applyStyle(n, e.Style)
	return n
}

func (r *wasmRenderer) renderTable(e TableEl, doc js.Value) js.Value {
	table := doc.Call("createElement", "table")
	applyStyle(table, e.Style)

	thead := doc.Call("createElement", "thead")
	headRow := doc.Call("createElement", "tr")
	for _, col := range e.Columns {
		th := doc.Call("createElement", "th")
		th.Set("textContent", col)
		headRow.Call("appendChild", th)
	}
	thead.Call("appendChild", headRow)
	table.Call("appendChild", thead)

	tbody := doc.Call("createElement", "tbody")
	for _, row := range e.Rows {
		tr := doc.Call("createElement", "tr")
		for _, cell := range row {
			td := doc.Call("createElement", "td")
			td.Set("textContent", cell)
			tr.Call("appendChild", td)
		}
		tbody.Call("appendChild", tr)
	}
	table.Call("appendChild", tbody)
	return table
}

func applyStyle(n js.Value, s Style) {
	css := styleToCSS(s)
	if css != "" {
		existing := n.Get("style").Get("cssText").String()
		n.Set("style", existing+css)
	}
}

func styleToCSS(s Style) string {
	var parts []string
	if s.FG != "" {
		parts = append(parts, "color:"+resolveCSSColor(s.FG))
	}
	if s.BG != "" {
		parts = append(parts, "background-color:"+resolveCSSColor(s.BG))
	}
	if s.Bold {
		parts = append(parts, "font-weight:bold")
	}
	if s.Italic {
		parts = append(parts, "font-style:italic")
	}
	if s.Underline {
		parts = append(parts, "text-decoration:underline")
	}
	// Sizing units mirror terminal cells: ch horizontally, lh (line
	// height) vertically with an em fallback for older browsers.
	if s.Width > 0 {
		parts = append(parts, fmt.Sprintf("width:%dch", s.Width))
	}
	if s.Height > 0 {
		parts = append(parts, fmt.Sprintf("height:%dem;height:%dlh", s.Height, s.Height))
	}
	if s.Padding != [4]int{} {
		parts = append(parts, fmt.Sprintf("padding:%dem %dch %dem %dch",
			s.Padding[0], s.Padding[1], s.Padding[2], s.Padding[3]))
		parts = append(parts, fmt.Sprintf("padding:%dlh %dch %dlh %dch",
			s.Padding[0], s.Padding[1], s.Padding[2], s.Padding[3]))
	}
	if s.Margin != [4]int{} {
		parts = append(parts, fmt.Sprintf("margin:%dem %dch %dem %dch",
			s.Margin[0], s.Margin[1], s.Margin[2], s.Margin[3]))
		parts = append(parts, fmt.Sprintf("margin:%dlh %dch %dlh %dch",
			s.Margin[0], s.Margin[1], s.Margin[2], s.Margin[3]))
	}
	if s.Border {
		col := "currentColor"
		if s.BorderColor != "" {
			col = resolveCSSColor(s.BorderColor)
		}
		parts = append(parts, "border:1px solid "+col)
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, ";") + ";"
}

func resolveCSSColor(c Color) string {
	if idx, err := strconv.Atoi(string(c)); err == nil && idx >= 0 && idx < 16 {
		return ansiPalette[idx]
	}
	return string(c)
}

// alignItemsCSS maps Align to a CSS align-items value. AlignStart
// returns "" since flex-start is the default.
func alignItemsCSS(a Align) string {
	switch a {
	case AlignCenter:
		return "center"
	case AlignEnd:
		return "flex-end"
	case AlignStretch:
		return "stretch"
	default:
		return ""
	}
}

// baseCSS is the default stylesheet injected by WASM builds (disable
// with WithoutBaseCSS). It styles the semantic HTML output to visually
// match the TUI renderer: monospace type on a dark background, buttons
// drawn as "[ Label ]" that reverse-video on focus, unordered lists
// with "- " markers, and bordered tables.
const baseCSS = `
:root { color-scheme: dark; }
html, body { height: 100%; }
body {
  margin: 0;
  background: #14151a;
  color: #e6e6e6;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, "Liberation Mono", monospace;
  font-size: 14px;
  line-height: 1.4;
}
#wui-root { padding: 1em 1ch; }
#wui-root button {
  font: inherit;
  color: inherit;
  background: none;
  border: none;
  padding: 0;
  cursor: pointer;
  width: fit-content;
}
#wui-root button::before { content: "[ "; }
#wui-root button::after { content: " ]"; }
#wui-root button:hover:not(:disabled),
#wui-root button:focus-visible {
  background: #e6e6e6;
  color: #14151a;
  outline: none;
}
#wui-root button:disabled { opacity: 0.45; cursor: default; }
#wui-root input {
  font: inherit;
  color: inherit;
  background: #1f2128;
  border: 1px solid #3a3d46;
  border-radius: 3px;
  padding: 0 1ch;
}
#wui-root input:focus { outline: none; border-color: #7aa2f7; }
#wui-root a { color: inherit; }
#wui-root a:focus-visible {
  background: #e6e6e6;
  color: #14151a;
  outline: none;
}
#wui-root ul { list-style: none; margin: 0; padding: 0; }
#wui-root ul > li::before { content: "- "; }
#wui-root ol { margin: 0; padding: 0; list-style-position: inside; }
#wui-root table { border-collapse: collapse; }
#wui-root th, #wui-root td {
  border: 1px solid #3a3d46;
  padding: 0 1ch;
  text-align: left;
}
`

// injectBaseCSS appends the default stylesheet to <head>, once.
func injectBaseCSS(doc js.Value) {
	if existing := doc.Call("getElementById", "wui-base-css"); !existing.IsNull() {
		return
	}
	style := doc.Call("createElement", "style")
	style.Set("id", "wui-base-css")
	style.Set("textContent", baseCSS)
	doc.Get("head").Call("appendChild", style)
}
