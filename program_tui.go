//go:build !js

package wui

import (
	"fmt"
	"net"
	"net/http"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type platformState struct {
	teaProgram *tea.Program
	adapter    *teaAdapter
}

func newProgram(m Model, cfg config) *Program {
	adapter := &teaAdapter{
		model:    m,
		renderer: newTUIRenderer(80, 24),
		focus:    newTUIFocusManager(),
	}
	return &Program{
		model: m,
		cfg:   cfg,
		platformState: platformState{
			adapter:    adapter,
			teaProgram: tea.NewProgram(adapter, tea.WithAltScreen()),
		},
	}
}

func (p *Program) run() error {
	if p.cfg.serveEnabled {
		ln, err := listenWeb(p.cfg.serveAddr)
		if err != nil {
			return fmt.Errorf("wui: web server: %w", err)
		}
		defer ln.Close()
		go http.Serve(ln, http.FileServer(http.Dir(p.cfg.webDir)))
		p.adapter.serveURL = serveURL(ln.Addr())
	}
	_, err := p.teaProgram.Run()
	return err
}

// Port range scanned when WithWebServer gets an empty address: an
// unprivileged, developer-conventional block starting at wui's
// default port.
const (
	autoPortMin = 8765
	autoPortMax = 8864
)

// listenWeb binds the given address, or — when addr is empty — picks a
// port automatically: loopback only, first free port in
// [autoPortMin, autoPortMax], falling back to an OS-assigned ephemeral
// port when the whole range is busy.
func listenWeb(addr string) (net.Listener, error) {
	if addr != "" {
		return net.Listen("tcp", addr)
	}
	for port := autoPortMin; port <= autoPortMax; port++ {
		ln, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
		if err == nil {
			return ln, nil
		}
	}
	return net.Listen("tcp", "localhost:0")
}

// serveURL derives a browser-openable URL from the bound listener
// address, mapping wildcard and loopback hosts to "localhost".
func serveURL(addr net.Addr) string {
	host, port, err := net.SplitHostPort(addr.String())
	if err != nil {
		return "http://" + addr.String() + "/"
	}
	if ip := net.ParseIP(host); ip == nil || ip.IsUnspecified() || ip.IsLoopback() {
		host = "localhost"
	}
	return "http://" + net.JoinHostPort(host, port) + "/"
}

// teaAdapter bridges a wui.Model into a tea.Model.
type teaAdapter struct {
	model    Model
	renderer *tuiRenderer
	focus    *tuiFocusManager
	serveURL string // non-empty when WithWebServer is active
}

func (a *teaAdapter) Init() tea.Cmd {
	return wuiCmdToTea(a.model.Init())
}

func (a *teaAdapter) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		a.renderer.Width = m.Width
		a.renderer.Height = m.Height
		newModel, cmd := a.model.Update(ResizeMsg{Width: m.Width, Height: m.Height})
		a.model = newModel
		return a, wuiCmdToTea(cmd)

	case tea.KeyMsg:
		return a.handleKey(m)

	default:
		// Anything else (including app-defined Msg values returned
		// from a Cmd and sent back via tea) goes straight to the
		// user's Update.
		newModel, cmd := a.model.Update(m)
		a.model = newModel
		return a, wuiCmdToTea(cmd)
	}
}

func (a *teaAdapter) handleKey(m tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := m.String()

	if key == "ctrl+c" {
		return a, tea.Quit
	}

	focusables := collectFocusables(a.model.View())

	if key == "tab" {
		a.focus.SetIDs(focusableIDs(focusables))
		a.focus.Next()
		a.renderer.FocusedID = a.focus.FocusedID()
		return a, nil
	}
	if key == "shift+tab" {
		a.focus.SetIDs(focusableIDs(focusables))
		a.focus.Prev()
		a.renderer.FocusedID = a.focus.FocusedID()
		return a, nil
	}

	focusedID := a.focus.FocusedID()
	if focusedID != "" {
		if newModel, cmd, handled := a.routeKeyToFocused(focusedID, focusables, m); handled {
			a.model = newModel
			return a, cmd
		}
	}

	newModel, cmd := a.model.Update(KeyMsg{Key: key, Rune: firstRune(key)})
	a.model = newModel
	return a, wuiCmdToTea(cmd)
}

// routeKeyToFocused dispatches a key event to whichever element holds
// focus: text inputs get the key forwarded to their bubbles model;
// buttons and links activate on Enter via their Activate closure;
// elements inside a Form fall back to submitting the form on Enter,
// matching native browser behaviour in the HTML renderer.
func (a *teaAdapter) routeKeyToFocused(focusedID string, focusables []focusable, m tea.KeyMsg) (Model, tea.Cmd, bool) {
	var target *focusable
	for i := range focusables {
		if focusables[i].ID == focusedID {
			target = &focusables[i]
			break
		}
	}
	if target == nil {
		return a.model, nil, false
	}

	if !target.IsInput {
		if m.String() == "enter" {
			if target.Activate != nil {
				newModel, cmd := a.model.Update(target.Activate())
				return newModel, wuiCmdToTea(cmd), true
			}
			if msg := a.submitForm(target.Form); msg != nil {
				newModel, cmd := a.model.Update(msg)
				return newModel, wuiCmdToTea(cmd), true
			}
		}
		return a.model, nil, true
	}

	el := findInputByID(a.model.View(), focusedID)
	if el == nil {
		return a.model, nil, false
	}

	in := a.renderer.ensureInput(*el)
	prevValue := in.Value()
	updated, _ := in.Update(m)
	a.renderer.setInput(focusedID, updated)

	if m.String() == "enter" {
		if el.OnSubmit != nil {
			newModel, cmd := a.model.Update(el.OnSubmit(updated.Value()))
			return newModel, wuiCmdToTea(cmd), true
		}
		if msg := a.submitForm(target.Form); msg != nil {
			newModel, cmd := a.model.Update(msg)
			return newModel, wuiCmdToTea(cmd), true
		}
		return a.model, nil, true
	}

	if updated.Value() != prevValue {
		if el.OnChange != nil {
			newModel, cmd := a.model.Update(el.OnChange(updated.Value()))
			return newModel, wuiCmdToTea(cmd), true
		}
		// No callback — emit generic InputMsg so app can still react.
		newModel, cmd := a.model.Update(InputMsg{ID: focusedID, Value: updated.Value()})
		return newModel, wuiCmdToTea(cmd), true
	}

	return a.model, nil, true
}

// submitForm collects the form's current input values and produces the
// form's OnSubmit Msg, or nil when there is no form or no handler.
func (a *teaAdapter) submitForm(form *FormEl) Msg {
	if form == nil || form.OnSubmit == nil {
		return nil
	}
	return form.OnSubmit(a.renderer.formValues(*form))
}

func (a *teaAdapter) View() string {
	a.focus.SetIDs(focusableIDs(collectFocusables(a.model.View())))
	a.renderer.FocusedID = a.focus.FocusedID()
	view := a.renderer.Render(a.model.View())
	if a.serveURL == "" {
		return view
	}
	return a.withStatusBar(view)
}

// withStatusBar pins a one-line bar to the bottom of the screen
// linking to the web rendering of the app. If the model implements
// Pather, the link targets the equivalent path via the URL hash.
func (a *teaAdapter) withStatusBar(view string) string {
	url := a.serveURL
	if p, ok := a.model.(Pather); ok {
		if path := p.Path(); path != "" && path != "/" {
			url += "#" + path
		}
	}
	bar := lipgloss.NewStyle().
		Reverse(true).
		Width(a.renderer.Width).
		Render(" web ⇒ " + url)
	body := lipgloss.NewStyle().MaxHeight(a.renderer.Height - 1).Render(view)
	if pad := a.renderer.Height - 1 - lipgloss.Height(body); pad > 0 {
		body += strings.Repeat("\n", pad)
	}
	return body + "\n" + bar
}

func wuiCmdToTea(cmd Cmd) tea.Cmd {
	if cmd == nil {
		return nil
	}
	return func() tea.Msg {
		return cmd()
	}
}

func firstRune(key string) rune {
	for _, r := range key {
		return r
	}
	return 0
}
