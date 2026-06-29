package wui

// Element is the renderable unit. Every concrete element type satisfies
// it; renderers type-switch on the concrete type.
type Element interface {
	isElement()
}

// Direction controls Box layout axis.
type Direction int

const (
	Row Direction = iota
	Column
)

// Align values for flex-like alignment.
type Align int

const (
	AlignStart Align = iota
	AlignCenter
	AlignEnd
	AlignStretch
)

type TextEl struct {
	Content string
	Style   Style
}

type BoxEl struct {
	Direction Direction
	Gap       int
	Align     Align
	Children  []Element
	Style     Style
}

type ButtonEl struct {
	Label    string
	OnClick  func() Msg
	Style    Style
	Disabled bool
}

type TextInputEl struct {
	ID          string
	Value       string
	Placeholder string
	Password    bool
	OnChange    func(string) Msg
	OnSubmit    func(string) Msg
	Style       Style
	Disabled    bool
}

type FormEl struct {
	Children []Element
	OnSubmit func(values map[string]string) Msg
	Style    Style
}

type ListEl struct {
	Items   []Element
	Ordered bool
	Style   Style
}

type ScrollAreaEl struct {
	Child     Element
	MaxHeight int
	Style     Style
}

type LinkEl struct {
	Label string
	Href  string
	Style Style
}

type TableEl struct {
	Columns []string
	Rows    [][]string
	Style   Style
}

func (TextEl) isElement()       {}
func (BoxEl) isElement()        {}
func (ButtonEl) isElement()     {}
func (TextInputEl) isElement()  {}
func (FormEl) isElement()       {}
func (ListEl) isElement()       {}
func (ScrollAreaEl) isElement() {}
func (LinkEl) isElement()       {}
func (TableEl) isElement()      {}

// Text creates a text node.
func Text(s string, opts ...func(*TextEl)) Element {
	e := TextEl{Content: s}
	for _, opt := range opts {
		opt(&e)
	}
	return e
}

// WithTextStyle sets style on a TextEl built via Text.
func WithTextStyle(s Style) func(*TextEl) {
	return func(e *TextEl) { e.Style = s }
}

// Box creates a flex container laid out along dir.
func Box(dir Direction, children ...Element) Element {
	return BoxEl{Direction: dir, Children: children}
}

// BoxStyled creates a flex container with explicit style and gap.
func BoxStyled(dir Direction, gap int, style Style, children ...Element) Element {
	return BoxEl{Direction: dir, Gap: gap, Style: style, Children: children}
}

// Button creates a clickable button.
func Button(label string, onClick func() Msg, opts ...func(*ButtonEl)) Element {
	e := ButtonEl{Label: label, OnClick: onClick}
	for _, opt := range opts {
		opt(&e)
	}
	return e
}

// WithButtonStyle sets style on a ButtonEl.
func WithButtonStyle(s Style) func(*ButtonEl) {
	return func(e *ButtonEl) { e.Style = s }
}

// Disabled marks a ButtonEl as disabled.
func Disabled() func(*ButtonEl) {
	return func(e *ButtonEl) { e.Disabled = true }
}

// Input creates a text input field identified by id.
func Input(id string, opts ...func(*TextInputEl)) Element {
	e := TextInputEl{ID: id}
	for _, opt := range opts {
		opt(&e)
	}
	return e
}

// WithValue sets the current value of a TextInputEl.
func WithValue(v string) func(*TextInputEl) {
	return func(e *TextInputEl) { e.Value = v }
}

// WithPlaceholder sets placeholder text on a TextInputEl.
func WithPlaceholder(p string) func(*TextInputEl) {
	return func(e *TextInputEl) { e.Placeholder = p }
}

// WithPassword marks a TextInputEl as a password field.
func WithPassword() func(*TextInputEl) {
	return func(e *TextInputEl) { e.Password = true }
}

// WithOnChange sets the change handler on a TextInputEl.
func WithOnChange(f func(string) Msg) func(*TextInputEl) {
	return func(e *TextInputEl) { e.OnChange = f }
}

// WithOnSubmit sets the submit handler on a TextInputEl (Enter key).
func WithOnSubmit(f func(string) Msg) func(*TextInputEl) {
	return func(e *TextInputEl) { e.OnSubmit = f }
}

// Form creates a form container that collects input values on submit.
func Form(onSubmit func(map[string]string) Msg, children ...Element) Element {
	return FormEl{OnSubmit: onSubmit, Children: children}
}

// List creates a list of items, ordered or unordered.
func List(ordered bool, items ...Element) Element {
	return ListEl{Ordered: ordered, Items: items}
}

// Scroll wraps a child in a scrollable area capped at maxHeight.
func Scroll(child Element, maxHeight int) Element {
	return ScrollAreaEl{Child: child, MaxHeight: maxHeight}
}

// Link creates a navigable link.
func Link(label, href string) Element {
	return LinkEl{Label: label, Href: href}
}

// Table creates a tabular display.
func Table(columns []string, rows [][]string) Element {
	return TableEl{Columns: columns, Rows: rows}
}
