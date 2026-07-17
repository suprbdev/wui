//go:build !js

package wui

import zone "github.com/lrstanley/bubblezone"

// RenderTUI renders an element tree to the string the TUI renderer
// would draw, without running a Program: no focus, fresh input state,
// mouse-zone markers stripped. Intended for snapshot tests of Views.
func RenderTUI(el Element) string {
	z := zone.New()
	defer z.Close()
	r := newTUIRenderer(80, 24, z)
	return z.Scan(r.Render(el))
}
