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
	ID       string // optional; focus key falls back to the label
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

type CheckboxEl struct {
	ID       string
	Label    string
	Checked  bool
	OnToggle func(checked bool) Msg
	Style    Style
	Disabled bool
}

type CardEl struct {
	Title string
	Child Element
	Style Style
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
func (CheckboxEl) isElement()   {}
func (CardEl) isElement()       {}
func (ListEl) isElement()       {}
func (ScrollAreaEl) isElement() {}
func (LinkEl) isElement()       {}
func (TableEl) isElement()      {}

// buttonFocusKey derives a button's stable identity — used as the TUI
// focus-ring key and the DOM id: the explicit ID when set, else the
// label. Buttons sharing a label (and no ID) in one view share a key.
func buttonFocusKey(e ButtonEl) string {
	if e.ID != "" {
		return "btn:" + e.ID
	}
	return "btn:" + e.Label
}

// checkboxToggleMsg produces the Msg for toggling a checkbox to the
// given state: the OnToggle callback when set, else a generic ToggleMsg.
func checkboxToggleMsg(e CheckboxEl, checked bool) Msg {
	if e.OnToggle != nil {
		return e.OnToggle(checked)
	}
	return ToggleMsg{ID: e.ID, Checked: checked}
}

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

// WithID gives a button an explicit stable identity. Without it the
// focus key derives from the label, so two buttons sharing a label in
// one view would share focus; an ID also lets the browser restore
// focus to the button across re-renders.
func WithID(id string) func(*ButtonEl) {
	return func(e *ButtonEl) { e.ID = id }
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

// Checkbox creates a toggleable checkbox with a text label. id must be
// stable and unique — it identifies the checkbox in the focus ring and
// the DOM. onToggle receives the new checked state; pass nil to get a
// generic ToggleMsg instead.
func Checkbox(id, label string, checked bool, onToggle func(bool) Msg, opts ...func(*CheckboxEl)) Element {
	e := CheckboxEl{ID: id, Label: label, Checked: checked, OnToggle: onToggle}
	for _, opt := range opts {
		opt(&e)
	}
	return e
}

// WithCheckboxStyle sets style on a CheckboxEl.
func WithCheckboxStyle(s Style) func(*CheckboxEl) {
	return func(e *CheckboxEl) { e.Style = s }
}

// CheckboxDisabled marks a CheckboxEl as disabled.
func CheckboxDisabled() func(*CheckboxEl) {
	return func(e *CheckboxEl) { e.Disabled = true }
}

// Card wraps a child in a titled, bordered panel. The TUI draws the
// title embedded in the top border; HTML renders <fieldset><legend>.
func Card(title string, child Element, opts ...func(*CardEl)) Element {
	e := CardEl{Title: title, Child: child}
	for _, opt := range opts {
		opt(&e)
	}
	return e
}

// WithCardStyle sets style on a CardEl. Border is implied; Padding,
// Margin, Width and BorderColor apply.
func WithCardStyle(s Style) func(*CardEl) {
	return func(e *CardEl) { e.Style = s }
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
