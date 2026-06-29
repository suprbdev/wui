//go:build !js

package wui

// tuiFocusManager is an index-based focus ring over a list of
// focusable element IDs, used because the terminal has no native tab
// order.
// tuiFocusManager is an index-based focus ring over a list of
// focusable element IDs. cursor=-1 means nothing is focused yet;
// first Next() advances to 0.
type tuiFocusManager struct {
	ids    []string
	cursor int
}

func newTUIFocusManager() *tuiFocusManager {
	return &tuiFocusManager{cursor: -1}
}

func (m *tuiFocusManager) SetIDs(ids []string) {
	m.ids = ids
	if m.cursor >= len(ids) {
		m.cursor = -1
	}
}

func (m *tuiFocusManager) Next() {
	if len(m.ids) == 0 {
		return
	}
	m.cursor = (m.cursor + 1) % len(m.ids)
}

func (m *tuiFocusManager) Prev() {
	if len(m.ids) == 0 {
		return
	}
	if m.cursor <= 0 {
		m.cursor = len(m.ids) - 1
	} else {
		m.cursor--
	}
}

func (m *tuiFocusManager) FocusedID() string {
	if len(m.ids) == 0 || m.cursor < 0 {
		return ""
	}
	return m.ids[m.cursor]
}
