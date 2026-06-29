package wui

// Color is a color value. In TUI it may be a hex string ("#ff0000"), a
// named ANSI color, or an ANSI index ("9"). In WASM it is passed through
// as CSS, with ANSI indices 0-15 translated to hex.
type Color string

// Style holds presentation properties shared by both renderers.
// Width/Height are terminal cells in TUI and pixels in WASM.
type Style struct {
	FG          Color
	BG          Color
	Bold        bool
	Italic      bool
	Underline   bool
	Padding     [4]int // top, right, bottom, left
	Margin      [4]int // top, right, bottom, left
	Width       int    // 0 = auto
	Height      int    // 0 = auto
	Border      bool
	BorderColor Color
}
