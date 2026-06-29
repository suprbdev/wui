//go:build js

package wui

import (
	"sync"
	"syscall/js"
)

type platformState struct {
	renderer *wasmRenderer
	mu       sync.Mutex
	done     chan struct{}
}

func newProgram(m Model) *Program {
	return &Program{
		model:         m,
		platformState: platformState{done: make(chan struct{})},
	}
}

func (p *Program) run() error {
	doc := js.Global().Get("document")
	root := doc.Call("createElement", "div")
	root.Set("id", "wui-root")
	doc.Get("body").Call("appendChild", root)

	p.platformState.renderer = newWASMRenderer(root, p.dispatch)

	if initCmd := p.model.Init(); initCmd != nil {
		go func() {
			p.dispatch(initCmd())
		}()
	}

	p.platformState.renderer.Render(p.model.View())

	<-p.platformState.done
	return nil
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

	if cmd != nil {
		go func() {
			p.dispatch(cmd())
		}()
	}
}
