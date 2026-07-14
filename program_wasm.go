//go:build js

package wui

import (
	"strings"
	"sync"
	"syscall/js"
)

type platformState struct {
	renderer *wasmRenderer
	mu       sync.Mutex
	done     chan struct{}
}

func newProgram(m Model, cfg config) *Program {
	return &Program{
		model:         m,
		cfg:           cfg,
		platformState: platformState{done: make(chan struct{})},
	}
}

func (p *Program) run() error {
	doc := js.Global().Get("document")
	root := doc.Call("getElementById", "wui-root")
	if root.IsNull() || root.IsUndefined() {
		root = doc.Call("createElement", "div")
		root.Set("id", "wui-root")
		doc.Get("body").Call("appendChild", root)
	}
	if !p.cfg.noBaseCSS {
		injectBaseCSS(doc)
	}

	p.platformState.renderer = newWASMRenderer(root, p.dispatch)

	if initCmd := p.model.Init(); initCmd != nil {
		go func() {
			p.dispatch(initCmd())
		}()
	}

	p.platformState.renderer.Render(p.model.View())
	p.syncHash()

	// Route the initial "#/path" fragment and subsequent hash changes
	// (back/forward navigation) to the app as NavigateMsg.
	if path := currentHashPath(); path != "" {
		p.dispatch(NavigateMsg{Path: path})
	}
	hashFn := js.FuncOf(func(this js.Value, args []js.Value) any {
		p.dispatch(NavigateMsg{Path: currentHashPath()})
		return nil
	})
	js.Global().Call("addEventListener", "hashchange", hashFn)

	<-p.platformState.done
	return nil
}

func currentHashPath() string {
	hash := js.Global().Get("location").Get("hash").String()
	return strings.TrimPrefix(hash, "#")
}

// syncHash mirrors the model's Path into location.hash so the browser
// URL always matches the location the TUI status bar links to.
// replaceState is used so the sync neither pollutes history nor fires
// a hashchange event (which would echo a NavigateMsg back).
func (p *Program) syncHash() {
	pather, ok := p.model.(Pather)
	if !ok {
		return
	}
	target := ""
	if path := pather.Path(); path != "" && path != "/" {
		target = "#" + path
	}
	loc := js.Global().Get("location")
	if loc.Get("hash").String() == target {
		return
	}
	url := target
	if url == "" {
		url = loc.Get("pathname").String() + loc.Get("search").String()
	}
	js.Global().Get("history").Call("replaceState", js.Null(), "", url)
}

// dispatch runs the Elm update step and re-renders. It is called from
// js.FuncOf event callbacks, which the syscall/js runtime invokes on
// their own goroutines, so access is serialized with a mutex.
func (p *Program) dispatch(msg Msg) {
	if msg == nil {
		return
	}
	p.platformState.mu.Lock()
	defer p.platformState.mu.Unlock()

	newModel, cmd := p.model.Update(msg)
	p.model = newModel
	p.platformState.renderer.Render(p.model.View())
	p.syncHash()

	if cmd != nil {
		go func() {
			p.dispatch(cmd())
		}()
	}
}
