// Command todo demonstrates lists, scroll areas, per-item buttons,
// and clearing an input programmatically after submit — identical in
// TUI and WASM builds.
package main

import (
	"flag"
	"fmt"

	"github.com/suprbdev/wui"
)

type model struct {
	draft string
	items []string
}

func (m model) Init() wui.Cmd { return nil }

type draftChangedMsg struct{ val string }
type addMsg struct{ text string }
type removeMsg struct{ index int }
type clearMsg struct{}

func (m model) Update(msg wui.Msg) (wui.Model, wui.Cmd) {
	switch v := msg.(type) {
	case draftChangedMsg:
		m.draft = v.val
	case addMsg:
		if v.text != "" {
			m.items = append(m.items, v.text)
			m.draft = "" // clears the input on both platforms
		}
	case removeMsg:
		if v.index >= 0 && v.index < len(m.items) {
			m.items = append(m.items[:v.index], m.items[v.index+1:]...)
		}
	case clearMsg:
		m.items = nil
	}
	return m, nil
}

func (m model) View() wui.Element {
	rows := make([]wui.Element, len(m.items))
	for i, item := range m.items {
		// Button labels double as TUI focus keys, so each label
		// includes the item number to stay unique.
		rows[i] = wui.BoxStyled(wui.Row, 1, wui.Style{},
			wui.Button(fmt.Sprintf("✕ %d", i+1), func() wui.Msg { return removeMsg{index: i} }),
			wui.Text(item),
		)
	}

	list := wui.Element(wui.Text("Nothing to do. Add something above.",
		wui.WithTextStyle(wui.Style{Italic: true})))
	if len(rows) > 0 {
		list = wui.Scroll(wui.List(false, rows...), 10)
	}

	return wui.BoxStyled(wui.Column, 1, wui.Style{Padding: [4]int{1, 2, 1, 2}},
		wui.Text("Todo", wui.WithTextStyle(wui.Style{Bold: true})),
		wui.Form(
			func(vals map[string]string) wui.Msg { return addMsg{text: vals["new-item"]} },
			wui.BoxStyled(wui.Row, 1, wui.Style{},
				wui.Input("new-item",
					wui.WithValue(m.draft),
					wui.WithPlaceholder("What needs doing?"),
					wui.WithOnChange(func(val string) wui.Msg { return draftChangedMsg{val} }),
				),
				wui.Button("Add", nil), // submit button
			),
		),
		list,
		wui.Text(fmt.Sprintf("%d item(s)", len(m.items))),
		wui.Button("Clear all", func() wui.Msg { return clearMsg{} }),
	)
}

func main() {
	serve := wui.ServeFlag("serve", "also serve the web build; picks a free port, or bind explicitly with -serve=:8765")
	flag.Parse()

	var opts []wui.Option
	if serve.Enabled {
		opts = append(opts, wui.WithWebServer(serve.Addr, "example/todo/web"))
	}
	if err := wui.NewProgram(model{}, opts...).Run(); err != nil {
		panic(err)
	}
}
