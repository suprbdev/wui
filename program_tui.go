//go:build !js

package wui

import (
	tea "github.com/charmbracelet/bubbletea"
)

type platformState struct {
	teaProgram *tea.Program
}

func newProgram(m Model) *Program {
	adapter := &teaAdapter{
		model:    m,
		renderer: newTUIRenderer(80, 24),
		focus:    newTUIFocusManager(),
	}
	return &Program{
		model:         m,
		platformState: platformState{teaProgram: tea.NewProgram(adapter, tea.WithAltScreen())},
	}
}

func (p *Program) run() error {
	_, err := p.platformState.teaProgram.Run()
	return err
}

// teaAdapter bridges a wui.Model into a tea.Model.
type teaAdapter struct {
	model    Model
	renderer *tuiRenderer
	focus    *tuiFocusManager
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
// buttons and links activate on Enter via their Activate closure.
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
		if m.String() == "enter" && target.Activate != nil {
			newModel, cmd := a.model.Update(target.Activate())
			return newModel, wuiCmdToTea(cmd), true
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
			newModel, _ := a.model.Update(SubmitMsg{FormValues: map[string]string{focusedID: updated.Value()}})
			return newModel, nil, true
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

func (a *teaAdapter) View() string {
	a.focus.SetIDs(focusableIDs(collectFocusables(a.model.View())))
	a.renderer.FocusedID = a.focus.FocusedID()
	return a.renderer.Render(a.model.View())
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
