// Command counter is a minimal wui example: buttons that adjust a
// count, rendering identically as a TUI or in a WASM build.
//
// Run with -serve :8765 (TUI build) to also serve the web build and
// show a status bar linking to it.
package main

import (
	"flag"
	"fmt"

	"github.com/suprbdev/wui"
)

type model struct{ count int }

func (m model) Init() wui.Cmd { return nil }

type deltaMsg struct{ by int }
type resetMsg struct{}

func (m model) Update(msg wui.Msg) (wui.Model, wui.Cmd) {
	switch v := msg.(type) {
	case deltaMsg:
		m.count += v.by
	case resetMsg:
		m.count = 0
	}
	return m, nil
}

func (m model) View() wui.Element {
	return wui.BoxStyled(wui.Column, 1, wui.Style{Padding: [4]int{1, 2, 1, 2}},
		wui.Text("Counter", wui.WithTextStyle(wui.Style{Bold: true})),
		wui.Text(fmt.Sprintf("Count: %d", m.count)),
		wui.BoxStyled(wui.Row, 1, wui.Style{},
			wui.Button("Increment", func() wui.Msg { return deltaMsg{by: 1} }),
			wui.Button("Decrement", func() wui.Msg { return deltaMsg{by: -1} }),
			wui.Button("Reset", func() wui.Msg { return resetMsg{} }),
		),
	)
}

func main() {
	serve := flag.String("serve", "", "also serve the web build at this address (e.g. :8765)")
	flag.Parse()

	var opts []wui.Option
	if *serve != "" {
		opts = append(opts, wui.WithWebServer(*serve, "example/counter/web"))
	}
	if err := wui.NewProgram(model{}, opts...).Run(); err != nil {
		panic(err)
	}
}
