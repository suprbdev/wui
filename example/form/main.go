// Command form demonstrates wui text inputs, form submission, and
// rendering a results table — works identically as TUI or WASM.
package main

import (
	"fmt"

	"github.com/suprbdev/wui"
)

type state int

const (
	stateForm state = iota
	stateResult
)

type model struct {
	name    string
	email   string
	state   state
	message string
}

func (m model) Init() wui.Cmd { return nil }

type submitMsg struct{ name, email string }
type nameChangedMsg struct{ val string }
type emailChangedMsg struct{ val string }

func (m model) Update(msg wui.Msg) (wui.Model, wui.Cmd) {
	switch v := msg.(type) {
	case nameChangedMsg:
		m.name = v.val
	case emailChangedMsg:
		m.email = v.val
	case submitMsg:
		if v.name == "" || v.email == "" {
			m.message = "Name and email are required."
		} else {
			m.message = fmt.Sprintf("Hello, %s (%s)!", v.name, v.email)
			m.state = stateResult
		}
	case wui.KeyMsg:
		if v.Key == "q" && m.state == stateResult {
			m.state = stateForm
			m.name = ""
			m.email = ""
			m.message = ""
		}
	}
	return m, nil
}

func (m model) View() wui.Element {
	if m.state == stateResult {
		return wui.Box(wui.Column,
			wui.Text(m.message),
			wui.Text(""),
			wui.Table(
				[]string{"Field", "Value"},
				[][]string{
					{"Name", m.name},
					{"Email", m.email},
				},
			),
			wui.Text(""),
			wui.Button("Back", func() wui.Msg { return wui.KeyMsg{Key: "q"} }),
			wui.Text("(press q to go back)"),
		)
	}

	return wui.Form(
		func(vals map[string]string) wui.Msg {
			return submitMsg{name: vals["name"], email: vals["email"]}
		},
		wui.Text("Contact Form"),
		wui.Text(""),
		wui.Text("Name:"),
		wui.Input("name",
			wui.WithValue(m.name),
			wui.WithPlaceholder("Your name"),
			wui.WithOnChange(func(val string) wui.Msg { return nameChangedMsg{val} }),
		),
		wui.Text(""),
		wui.Text("Email:"),
		wui.Input("email",
			wui.WithValue(m.email),
			wui.WithPlaceholder("you@example.com"),
			wui.WithOnChange(func(val string) wui.Msg { return emailChangedMsg{val} }),
		),
		wui.Text(""),
		wui.Button("Submit", func() wui.Msg {
			return submitMsg{name: m.name, email: m.email}
		}),
		wui.Text(m.message),
	)
}

func main() {
	if err := wui.NewProgram(model{}).Run(); err != nil {
		panic(err)
	}
}
