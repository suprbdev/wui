package wui

// Model is the user-implemented Elm Architecture model.
type Model interface {
	Init() Cmd
	Update(Msg) (Model, Cmd)
	View() Element
}

// Pather is an optional interface a Model can implement to expose its
// current logical location as a path such as "/" or "/settings".
//
// When the TUI runs with WithWebServer, the status bar link points at
// the equivalent web location ("http://host/#/settings"). In the
// browser, the path is mirrored into location.hash after every update,
// and a NavigateMsg is dispatched on initial load and whenever the
// hash changes (e.g. back/forward navigation), so both platforms stay
// addressable at the same locations.
type Pather interface {
	Path() string
}

type config struct {
	serveAddr string
	webDir    string
	noBaseCSS bool
}

// Option configures a Program at construction time.
type Option func(*config)

// WithWebServer serves dir — a WASM build of the same app (index.html,
// main.wasm, wasm_exec.js) — over HTTP at addr (e.g. ":8765") for as
// long as the TUI runs, and displays a status bar at the bottom of the
// TUI linking to the equivalent web page. It has no effect in WASM
// builds, so it is safe to pass unconditionally from shared code.
func WithWebServer(addr, dir string) Option {
	return func(c *config) {
		c.serveAddr = addr
		c.webDir = dir
	}
}

// WithoutBaseCSS disables injection of wui's default terminal-like
// stylesheet in WASM builds. Use it when the host page provides its
// own styling. It has no effect in TUI builds.
func WithoutBaseCSS() Option {
	return func(c *config) { c.noBaseCSS = true }
}

// Program is the wui runtime. Construct with NewProgram and call Run.
type Program struct {
	model Model
	cfg   config
	platformState
}

// NewProgram creates a Program for the given model.
func NewProgram(m Model, opts ...Option) *Program {
	var cfg config
	for _, opt := range opts {
		opt(&cfg)
	}
	return newProgram(m, cfg)
}

// Run starts the program loop, blocking until the program exits.
func (p *Program) Run() error {
	return p.run()
}
