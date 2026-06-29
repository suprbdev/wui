// Command counter is a minimal wui example: a button that increments
// a count, rendering identically as a TUI or in a WASM build.
package main

import (
	"fmt"

	"github.com/suprbdev/wui"
)

type model struct{ count int }

func (m model) Init() wui.Cmd { return nil }

type incrementMsg struct{}

func (m model) Update(msg wui.Msg) (wui.Model, wui.Cmd) {
	switch msg.(type) {
	case incrementMsg:
		m.count++
	}
	return m, nil
}

func (m model) View() wui.Element {
	return wui.Box(wui.Column,
		wui.Text(fmt.Sprintf("Count: %d", m.count)),
		wui.Button("Increment", func() wui.Msg { return incrementMsg{} }),
	)
}

func main() {
	if err := wui.NewProgram(model{}).Run(); err != nil {
		panic(err)
	}
}
