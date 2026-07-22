package wui

// Msg is the message type for the Elm Update loop. Any value can be a
// Msg, including application-defined types — analogous to tea.Msg.
type Msg interface{}

// Cmd is a function that produces a Msg asynchronously.
type Cmd func() Msg

// KeyMsg represents a key press. Key is a normalized name such as
// "enter", "ctrl+c", "tab", "backspace", or a single printable rune.
type KeyMsg struct {
	Key  string
	Rune rune
}

// ClickMsg is sent when a Link is activated. Buttons instead invoke
// their own OnClick callback directly.
type ClickMsg struct {
	TargetID string
}

// ToggleMsg is sent when a Checkbox is toggled and it has no OnToggle
// callback of its own. Checked is the new state.
type ToggleMsg struct {
	ID      string
	Checked bool
}

// InputMsg is sent when a TextInput's value changes and it has no
// OnChange callback of its own.
type InputMsg struct {
	ID    string
	Value string
}

// SubmitMsg is sent when a Form is submitted and it has no OnSubmit
// callback of its own.
type SubmitMsg struct {
	FormValues map[string]string
}

// NavigateMsg is sent in WASM builds when the browser URL hash names a
// path — on initial load with a "#/path" fragment and on every
// hashchange (back/forward navigation). TUI programs never receive it.
// See Pather.
type NavigateMsg struct {
	Path string
}

// ResizeMsg is sent when the terminal or window is resized.
type ResizeMsg struct {
	Width  int
	Height int
}

// NoneMsg is a no-op message.
type NoneMsg struct{}

// firstRune returns the first rune of a normalized key name — the
// printable character for single-rune keys, or the first letter of a
// named key ("enter" → 'e'); callers should treat Rune as meaningful
// only for single-rune keys.
func firstRune(key string) rune {
	for _, r := range key {
		return r
	}
	return 0
}
