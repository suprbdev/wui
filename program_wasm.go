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

	// Global key events, mirroring the TUI: keys not aimed at an
	// editable element reach the app as KeyMsg. Editable elements keep
	// their own listeners (input/keydown wired by the renderer), so
	// typing in an Input never double-dispatches.
	keyFn := js.FuncOf(func(this js.Value, args []js.Value) any {
		event := args[0]
		if activeElementIsEditable(doc) {
			return nil
		}
		key, ok := normalizeKeyEvent(event)
		if !ok {
			return nil
		}
		if shouldPreventDefault(key) {
			event.Call("preventDefault")
		}
		p.dispatch(KeyMsg{Key: key, Rune: firstRune(key)})
		return nil
	})
	doc.Call("addEventListener", "keydown", keyFn)

	<-p.platformState.done
	return nil
}

// activeElementIsEditable reports whether keyboard input is currently
// aimed at a text-editing or otherwise key-consuming element.
func activeElementIsEditable(doc js.Value) bool {
	active := doc.Get("activeElement")
	if active.IsNull() || active.IsUndefined() {
		return false
	}
	switch active.Get("tagName").String() {
	case "INPUT", "TEXTAREA", "SELECT":
		return true
	}
	return active.Get("isContentEditable").Truthy()
}

// jsKeyNames maps browser KeyboardEvent.key values to the normalized
// names the TUI produces (bubbletea key names).
var jsKeyNames = map[string]string{
	"Backspace":  "backspace",
	"Enter":      "enter",
	"Escape":     "esc",
	"Tab":        "tab",
	"ArrowUp":    "up",
	"ArrowDown":  "down",
	"ArrowLeft":  "left",
	"ArrowRight": "right",
	"Home":       "home",
	"End":        "end",
	"PageUp":     "pgup",
	"PageDown":   "pgdown",
	"Delete":     "delete",
}

// normalizeKeyEvent translates a browser keydown event into the TUI's
// normalized key name. ok is false for events that should not reach the
// app: bare modifier presses, alt/meta chords (browser and OS
// shortcuts), and named keys wui does not model.
func normalizeKeyEvent(event js.Value) (key string, ok bool) {
	k := event.Get("key").String()
	switch k {
	case "Shift", "Control", "Alt", "Meta", "CapsLock", "NumLock":
		return "", false
	}
	if event.Get("altKey").Bool() || event.Get("metaKey").Bool() {
		return "", false
	}

	if name, found := jsKeyNames[k]; found {
		k = name
	} else if len([]rune(k)) != 1 {
		// Unmapped named key (F5, Insert, media keys, …).
		return "", false
	}

	if event.Get("ctrlKey").Bool() {
		return "ctrl+" + k, true
	}
	return k, true
}

// shouldPreventDefault reports whether a dispatched key must have its
// browser default suppressed: keys whose defaults would disrupt the app
// when no editable element has focus — space scrolls, backspace
// navigates history, "'" and "/" open Firefox quick-find, tab moves
// focus out of step with the app's own handling, and ctrl+backspace
// navigates in some browsers. Other ctrl chords keep their browser
// behaviour.
func shouldPreventDefault(key string) bool {
	switch key {
	case " ", "backspace", "'", "/", "tab", "ctrl+backspace":
		return true
	}
	return false
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
