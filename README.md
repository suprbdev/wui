# wui

A Go UI framework that renders the same component tree as a **terminal UI** (TUI) or as **semantic HTML** in the browser via WebAssembly. One codebase, two build targets, zero canvas.

```
make run-counter      # run counter example in terminal
make serve-counter    # build + serve counter in browser at :8765
```

---

## How it works

wui follows the [Elm Architecture](https://guide.elm-lang.org/architecture/): your application defines a **Model** (state), an **Update** function (events → new state), and a **View** function (state → element tree). wui drives the loop on the appropriate platform.

```
          ┌──────────────────────────────────────────────────────────┐
          │  Your application (shared)                               │
          │                                                          │
          │   Model ──Update(Msg)──► (Model, Cmd) ──► View() ──►    │
          │                                              Element     │
          └──────────────────────┬───────────────────────┬──────────┘
                                 │                       │
                    ┌────────────▼──────┐   ┌────────────▼──────────┐
                    │   TUI Renderer    │   │   HTML Renderer        │
                    │  (bubbletea +     │   │  (syscall/js DOM)      │
                    │   lipgloss)       │   │                        │
                    │                  │   │  <button>, <input>,    │
                    │  [ Increment ]   │   │  <form>, <table>, <a>  │
                    └───────────────────┘   └────────────────────────┘
```

The HTML renderer maps every element to the correct semantic HTML tag — not canvas, not `<div>` soup. This means native browser behaviours work out of the box: tab focus, form submission, scroll, accessibility, link navigation.

### Element → HTML mapping

| wui element | HTML output | Native behaviour |
|---|---|---|
| `Text` | `<span>` | — |
| `Box(Row)` | `<div style="display:flex;flex-direction:row">` | — |
| `Box(Column)` | `<div style="display:flex;flex-direction:column">` | — |
| `Button` | `<button>` | focusable, keyboard-activatable |
| `Input` | `<input type="text/password">` | native caret, IME, autofill |
| `Form` | `<form>` | submit on Enter, `preventDefault` wired |
| `List(false)` | `<ul><li>…</li></ul>` | — |
| `List(true)` | `<ol><li>…</li></ul>` | — |
| `Scroll` | `<div style="overflow:auto">` | native scroll |
| `Link` | `<a href="…">` | right-click, open-in-new-tab, history |
| `Table` | `<table><thead><tbody>…` | — |

---

## Installation

```bash
go get github.com/suprbdev/wui
```

Requires Go 1.24+. No CGO.

---

## Make targets

```
make help            # list all targets
make build           # build wui package (TUI)
make build-wasm      # build wui package (WASM)
make vet             # go vet both TUI and WASM targets
make test            # run tests
make run-counter     # run counter example in terminal
make run-form        # run form example in terminal
make wasm-counter    # build counter → example/counter/web/
make wasm-form       # build form    → example/form/web/
make serve-counter   # build + serve counter at http://localhost:8765
make serve-form      # build + serve form    at http://localhost:8765
make clean           # remove built WASM binaries
```

---

## Quick start

```go
package main

import (
    "fmt"
    "github.com/suprbdev/wui"
)

// 1. Define your state.
type model struct{ count int }

// 2. Define your messages.
type incrementMsg struct{}

// 3. Implement wui.Model.
func (m model) Init() wui.Cmd { return nil }

func (m model) Update(msg wui.Msg) (wui.Model, wui.Cmd) {
    if _, ok := msg.(incrementMsg); ok {
        m.count++
    }
    return m, nil
}

func (m model) View() wui.Element {
    return wui.Box(wui.Column,
        wui.Text(fmt.Sprintf("Count: %d", m.count)),
        wui.Button("Increment", func() wui.Msg { return incrementMsg{} }),
    )
}

func main() {
    wui.NewProgram(model{}).Run()
}
```

**TUI** — `make run-counter`

**Browser** — `make serve-counter` then open `http://localhost:8765`

Renders as:
```html
<div style="display:flex;flex-direction:column;…">
  <span>Count: 0</span>
  <button>Increment</button>
</div>
```

---

## Core concepts

### Model

Your application state. Must be a value type (struct). `wui.Model` requires three methods:

```go
type Model interface {
    Init()           Cmd          // runs once on startup; return nil or a Cmd
    Update(Msg)      (Model, Cmd) // pure: old state + message → new state + optional Cmd
    View()           Element      // pure: state → element tree
}
```

`Init` and `Update` must be pure — no side effects. Side effects (HTTP calls, timers, file I/O) belong in a `Cmd`.

### Msg

Any Go value. Define your own types:

```go
type tickMsg time.Time
type loadedMsg struct{ data []Item }
type errMsg struct{ err error }
```

wui also emits built-in messages your `Update` can handle:

| Type | When |
|---|---|
| `wui.KeyMsg{Key, Rune}` | Key press in TUI (key names: `"enter"`, `"ctrl+c"`, `"tab"`, `"backspace"`, `"up"`, `"down"`, etc.) |
| `wui.ResizeMsg{Width, Height}` | Terminal or window resize |
| `wui.InputMsg{ID, Value}` | Input value changed with no `OnChange` callback set |
| `wui.SubmitMsg{FormValues}` | Form submitted with no `OnSubmit` callback set |
| `wui.ClickMsg{TargetID}` | Link clicked |

### Cmd

A `func() Msg` run asynchronously. Use it for side effects:

```go
func fetchData() wui.Cmd {
    return func() wui.Msg {
        resp, err := http.Get("https://api.example.com/data")
        if err != nil {
            return errMsg{err}
        }
        // decode resp...
        return loadedMsg{data: items}
    }
}

func (m model) Init() wui.Cmd {
    return fetchData()
}
```

Return a `Cmd` from `Init` or `Update`. wui runs it in a goroutine and dispatches the resulting `Msg` back through `Update`.

---

## Elements

### Text

```go
wui.Text("Hello, world!")

// With style:
wui.Text("Warning", wui.WithTextStyle(wui.Style{FG: "#ff0000", Bold: true}))
```

Renders as `<span>` in HTML.

### Box

Flex container — `wui.Row` or `wui.Column`:

```go
wui.Box(wui.Row,
    wui.Text("Left"),
    wui.Text("Right"),
)

wui.Box(wui.Column,
    wui.Text("Top"),
    wui.Text("Bottom"),
)

// With gap and style:
wui.BoxStyled(wui.Column, 8, wui.Style{Border: true},
    wui.Text("Item 1"),
    wui.Text("Item 2"),
)
```

### Button

```go
wui.Button("Save", func() wui.Msg { return saveMsg{} })

// Disabled:
wui.Button("Save", nil, wui.Disabled())

// With style:
wui.Button("Delete", onDelete, wui.WithButtonStyle(wui.Style{FG: "#ff0000"}))
```

**TUI**: rendered as `[ Label ]`, focusable via Tab, activated via Enter.
**HTML**: `<button onclick=…>`.

### Input

Text inputs require a stable `id` string — wui uses it to maintain cursor/blink state in TUI and to identify the DOM element in HTML.

```go
wui.Input("username",
    wui.WithValue(m.username),
    wui.WithPlaceholder("Username"),
    wui.WithOnChange(func(val string) wui.Msg {
        return usernameChanged{val}
    }),
)

// Password field:
wui.Input("password",
    wui.WithPlaceholder("Password"),
    wui.WithPassword(),
    wui.WithOnSubmit(func(val string) wui.Msg {
        return loginMsg{password: val}
    }),
)
```

`OnChange` fires on every keystroke. `OnSubmit` fires on Enter (TUI) or Enter key in the input (HTML).

### Form

Wraps inputs in a `<form>` (HTML) or a vertical container (TUI). The `OnSubmit` callback receives all input values by ID when the form is submitted — via the Submit button or Enter in the last field.

```go
wui.Form(
    func(vals map[string]string) wui.Msg {
        return loginMsg{user: vals["user"], pass: vals["pass"]}
    },
    wui.Input("user", wui.WithPlaceholder("Username")),
    wui.Input("pass", wui.WithPlaceholder("Password"), wui.WithPassword()),
    wui.Button("Log in", func() wui.Msg { return nil }),
)
```

In TUI, form submission happens via the button's `OnClick`; `FormEl.OnSubmit` is used in HTML.

### List

```go
// Unordered:
wui.List(false,
    wui.Text("First item"),
    wui.Text("Second item"),
    wui.Button("Clickable item", onClick),
)

// Ordered:
wui.List(true,
    wui.Text("Step one"),
    wui.Text("Step two"),
)
```

Items can be any `Element`, not just text. Renders as `<ul>`/`<ol>` + `<li>` in HTML.

### ScrollArea

```go
wui.Scroll(
    wui.Box(wui.Column, items...),
    20, // max height: terminal rows (TUI) or pixels (HTML)
)
```

**TUI**: `lipgloss.MaxHeight`. **HTML**: `overflow:auto; max-height:Npx`.

### Link

```go
wui.Link("Docs", "https://pkg.go.dev/github.com/suprbdev/wui")
```

**HTML**: real `<a href="…">` — right-click works, opens in new tab, browser history, screen readers. **TUI**: underlined text, focusable via Tab.

### Table

```go
wui.Table(
    []string{"Name", "Version", "Status"},
    [][]string{
        {"bubbletea", "v1.3.10", "✓"},
        {"lipgloss",  "v1.1.0",  "✓"},
    },
)
```

**TUI**: lipgloss table with rounded borders. **HTML**: `<table><thead><tbody>`.

---

## Styling

`wui.Style` is a platform-agnostic style struct:

```go
type Style struct {
    FG, BG      Color    // hex "#ff0000", CSS color name "red", or ANSI index "9"
    Bold        bool
    Italic      bool
    Underline   bool
    Padding     [4]int   // top, right, bottom, left
    Margin      [4]int   // top, right, bottom, left
    Width       int      // 0 = auto; terminal cells (TUI) or px (HTML)
    Height      int      // 0 = auto; terminal cells (TUI) or px (HTML)
    Border      bool
    BorderColor Color
}
```

**Color values**:
- Hex strings (`"#ff0000"`) work on both platforms.
- Named CSS colors (`"red"`, `"cornflowerblue"`) work on both platforms.
- ANSI indices `"0"`–`"15"` are translated to hex for HTML, passed to lipgloss as-is for TUI.

> **Note on Width/Height**: these are terminal cell columns/rows in TUI and pixels in HTML. Choose values that make sense for each context, or omit them (`0` = auto).

---

## Keyboard navigation (TUI)

| Key | Action |
|---|---|
| `Tab` | Focus next focusable element |
| `Shift+Tab` | Focus previous focusable element |
| `Enter` | Activate focused button or link; submit focused input |
| `Ctrl+C` | Quit (intercepted by wui, not forwarded to `Update`) |

Focusable elements in tree order: **TextInput → Button → Link**.

Your `Update` receives `wui.KeyMsg` for any key that isn't consumed by focus management or input editing. Use it for global shortcuts:

```go
case wui.KeyMsg:
    if v.Key == "q" {
        // return a Cmd that signals quit, or transition to a done state
    }
```

---

## Commands (async work)

```go
// A Cmd that fires after a 1s delay:
func tick() wui.Cmd {
    return func() wui.Msg {
        time.Sleep(time.Second)
        return tickMsg{}
    }
}

func (m model) Init() wui.Cmd {
    return tick()
}

func (m model) Update(msg wui.Msg) (wui.Model, wui.Cmd) {
    switch msg.(type) {
    case tickMsg:
        m.elapsed++
        return m, tick() // reschedule
    }
    return m, nil
}
```

Each `Cmd` runs in its own goroutine. The returned `Msg` is dispatched back through `Update` on the main loop.

---

## WASM build

### Quick start

```bash
make serve-counter   # builds WASM + copies wasm_exec.js + serves at :8765
make serve-form
```

Or manually:

```bash
GOOS=js GOARCH=wasm go build -o web/main.wasm .
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" web/
```

### HTML harness

```html
<!DOCTYPE html>
<html>
<head><meta charset="utf-8"></head>
<body>
<script src="wasm_exec.js"></script>
<script>
  const go = new Go();
  WebAssembly.instantiateStreaming(fetch("main.wasm"), go.importObject)
    .then(r => go.run(r.instance));
</script>
</body>
</html>
```

wui mounts a `<div id="wui-root">` into `document.body` automatically.

### Serving

`make serve-counter` and `make serve-form` handle the WASM MIME type automatically. For production or custom setups, ensure the server sets `Content-Type: application/wasm` for `.wasm` files — most servers (Caddy, nginx, Go's `net/http`) do this by default. For ad-hoc local serving:

```bash
# npx serve web
# caddy file-server --root web
```

### Build constraints

All bubbletea/lipgloss/bubbles code is behind `//go:build !js`. All `syscall/js` DOM code is behind `//go:build js`. Standard Go build tags select the right renderer automatically — no manual configuration needed.

---

## Architecture

```
wui/
├── element.go        Element interface + all concrete types + constructors
├── style.go          Style struct (shared, no platform deps)
├── msg.go            Msg, Cmd, and built-in message types
├── program.go        Model interface + Program (NewProgram / Run)
├── program_tui.go    //go:build !js — bubbletea adapter + key routing
├── program_wasm.go   //go:build js  — DOM event loop + dispatch
├── render_tui.go     //go:build !js — element tree → lipgloss string
├── render_wasm.go    //go:build js  — element tree → DOM nodes
├── focus_tui.go      //go:build !js — Tab ring (index-based)
└── focus_wasm.go     //go:build js  — Tab ring (delegates to browser)
```

The `element.go`, `style.go`, `msg.go`, and `program.go` files have **no build tags and no platform dependencies** — they are the shared contract between your app and both renderers.

### Re-render strategy

**TUI**: bubbletea calls `View()` on every state change and diffs the output string to minimise redraws.

**HTML**: v1 does a full replace of `#wui-root` on every state change. Before clearing the DOM, wui saves:
1. `document.activeElement.id` — restored after the new tree is built, preserving keyboard focus.
2. All `<input>` values by id — restored after the new tree, preventing in-progress text from being wiped by unrelated state changes.

This is efficient enough for typical app trees. VDOM diffing is a future enhancement.

---

## Examples

### Counter (`example/counter/`)

Minimal example: one button, one counter.

```bash
make run-counter      # terminal
make serve-counter    # browser → http://localhost:8765
```

### Contact form (`example/form/`)

Two text inputs, validation, result view with a data table.

```bash
make run-form         # terminal
make serve-form       # browser → http://localhost:8765
```

TUI interaction:
- `Tab` to cycle between Name input, Email input, Submit button
- Type in each input field
- `Enter` on Submit button to submit
- `q` or Back button to return to the form

---

## Dependencies

| Package | Used by | Purpose |
|---|---|---|
| `github.com/charmbracelet/bubbletea` | TUI only | Event loop, alt-screen, key input |
| `github.com/charmbracelet/lipgloss` | TUI only | Layout, styling, borders, tables |
| `github.com/charmbracelet/bubbles` | TUI only | Text input widget (cursor, blink, echo modes) |
| `syscall/js` | WASM only | DOM manipulation (Go standard library) |

The WASM binary includes none of the charmbracelet packages. The TUI binary includes no `syscall/js` code. Both are enforced by `//go:build` constraints.

---

## Limitations (v1)

- **No VDOM diffing** — HTML renderer does a full tree replace on each update. Works well for most apps; large frequently-updating trees may flicker.
- **Width/Height units differ** — `Style.Width`/`Height` are terminal cells in TUI and pixels in HTML. Use `0` (auto) when building cross-platform.
- **No mouse support in TUI** — buttons activate via Tab+Enter only; mouse clicks are not wired.
- **No styled components** — style is applied per-element inline; there is no global stylesheet or theme system.
- **ANSI color fidelity** — ANSI indices 0–15 map to fixed hex values in HTML; exact colours depend on the terminal's colour scheme in TUI.

---

## License

MIT
