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

// ResizeMsg is sent when the terminal or window is resized.
type ResizeMsg struct {
	Width  int
	Height int
}

// NoneMsg is a no-op message.
type NoneMsg struct{}
