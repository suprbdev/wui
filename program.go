package wui

// Model is the user-implemented Elm Architecture model.
type Model interface {
	Init() Cmd
	Update(Msg) (Model, Cmd)
	View() Element
}

// Program is the wui runtime. Construct with NewProgram and call Run.
type Program struct {
	model Model
	platformState
}

// NewProgram creates a Program for the given model.
func NewProgram(m Model) *Program {
	return newProgram(m)
}

// Run starts the program loop, blocking until the program exits.
func (p *Program) Run() error {
	return p.run()
}
