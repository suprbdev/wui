// Command form demonstrates wui text inputs, form submission, and
// rendering a results table — works identically as TUI or WASM. It
// also implements wui.Pather, so the TUI status bar (with -serve) and
// the browser URL hash agree on the current location.
package main

import (
	"flag"
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

// Path exposes the current screen as a web-equivalent location.
func (m model) Path() string {
	if m.state == stateResult {
		return "/result"
	}
	return "/"
}

type submitMsg struct{ name, email string }
type nameChangedMsg struct{ val string }
type emailChangedMsg struct{ val string }
type backMsg struct{}

func (m model) Update(msg wui.Msg) (wui.Model, wui.Cmd) {
	switch v := msg.(type) {
	case nameChangedMsg:
		m.name = v.val
	case emailChangedMsg:
		m.email = v.val
	case submitMsg:
		m.name = v.name
		m.email = v.email
		if v.name == "" || v.email == "" {
			m.message = "Name and email are required."
		} else {
			m.message = fmt.Sprintf("Hello, %s (%s)!", v.name, v.email)
			m.state = stateResult
		}
	case backMsg:
		m = model{}
	case wui.KeyMsg:
		if v.Key == "q" && m.state == stateResult {
			m = model{}
		}
	case wui.NavigateMsg:
		// Browser back/forward or a shared "#/result" link. Only show
		// the result screen when there is a result to show.
		if v.Path == "/result" && m.message != "" {
			m.state = stateResult
		} else {
			m.state = stateForm
		}
	}
	return m, nil
}

func (m model) View() wui.Element {
	if m.state == stateResult {
		return wui.BoxStyled(wui.Column, 1, wui.Style{Padding: [4]int{1, 2, 1, 2}},
			wui.Text(m.message, wui.WithTextStyle(wui.Style{Bold: true})),
			wui.Table(
				[]string{"Field", "Value"},
				[][]string{
					{"Name", m.name},
					{"Email", m.email},
				},
			),
			wui.Button("Back", func() wui.Msg { return backMsg{} }),
			wui.Text("(press q to go back)", wui.WithTextStyle(wui.Style{Italic: true})),
		)
	}

	return wui.Form(
		func(vals map[string]string) wui.Msg {
			return submitMsg{name: vals["name"], email: vals["email"]}
		},
		wui.BoxStyled(wui.Column, 1, wui.Style{Padding: [4]int{1, 2, 1, 2}},
			wui.Text("Contact Form", wui.WithTextStyle(wui.Style{Bold: true})),
			wui.Text("Name:"),
			wui.Input("name",
				wui.WithValue(m.name),
				wui.WithPlaceholder("Your name"),
				wui.WithOnChange(func(val string) wui.Msg { return nameChangedMsg{val} }),
			),
			wui.Text("Email:"),
			wui.Input("email",
				wui.WithValue(m.email),
				wui.WithPlaceholder("you@example.com"),
				wui.WithOnChange(func(val string) wui.Msg { return emailChangedMsg{val} }),
			),
			// nil OnClick inside a Form = submit button on both
			// platforms, like <button type="submit">.
			wui.Button("Submit", nil),
			wui.Text(m.message, wui.WithTextStyle(wui.Style{FG: "9"})),
		),
	)
}

func main() {
	serve := flag.String("serve", "", "also serve the web build at this address (e.g. :8765)")
	flag.Parse()

	var opts []wui.Option
	if *serve != "" {
		opts = append(opts, wui.WithWebServer(*serve, "example/form/web"))
	}
	if err := wui.NewProgram(model{}, opts...).Run(); err != nil {
		panic(err)
	}
}
