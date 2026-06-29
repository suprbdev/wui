//go:build js

package wui

import "syscall/js"

// wasmFocusManager delegates focus tracking to the browser's native
// tab order; interactive HTML elements (button, input, a) are natively
// focusable so there is little for wui to manage here.
type wasmFocusManager struct{}

func newWASMFocusManager() *wasmFocusManager {
	return &wasmFocusManager{}
}

func (m *wasmFocusManager) SetIDs(_ []string) {}
func (m *wasmFocusManager) Next()             {}
func (m *wasmFocusManager) Prev()             {}

func (m *wasmFocusManager) FocusedID() string {
	active := js.Global().Get("document").Get("activeElement")
	if active.IsNull() || active.IsUndefined() {
		return ""
	}
	return active.Get("id").String()
}
