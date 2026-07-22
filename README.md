# wui

A Go UI framework that renders the same component tree as a **terminal UI** (TUI) or as **semantic HTML** in the browser via WebAssembly. One codebase, two build targets, zero canvas.

```
make run-counter      # run counter example in terminal
make serve-counter    # build + serve counter in browser (auto-picks a free port)
make tui-counter      # run counter in terminal AND serve it in the browser
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

By default the WASM build also injects a small terminal-look stylesheet (monospace type, dark background, `[ Button ]` chrome, `[x]` checkboxes, `- ` list markers, bordered tables, preserved whitespace so column-aligned text lines up like in a terminal) so the browser rendering visually matches the TUI. Disable it with `wui.WithoutBaseCSS()` if the host page brings its own styles.

### Element → HTML mapping

| wui element | HTML output | Native behaviour |
|---|---|---|
| `Text` | `<span>` | — |
| `Box(Row)` | `<div style="display:flex;flex-direction:row">` | — |
| `Box(Column)` | `<div style="display:flex;flex-direction:column">` | — |
| `Button` | `<button>` | focusable, keyboard-activatable |
| `Input` | `<input type="text/password">` | native caret, IME, autofill |
| `Checkbox` | `<label><input type="checkbox">` | focusable, Space toggles, label click |
| `Card` | `<fieldset><legend>` | — |
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
make run-NAME        # run example NAME in terminal
make tui-NAME        # run example NAME in terminal + serve its web build (auto port)
make wasm-NAME       # build example NAME → example/NAME/web/
make serve-NAME      # build + serve example NAME in the browser (auto port)
make clean           # remove built WASM binaries
```

`NAME` is any of the examples: `counter`, `form`, `todo`, `timer`. Both serving targets auto-pick the first free port in 8765–8864 (loopback only); pin one with `make serve-counter PORT=9000` (binds all interfaces).

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

**Browser** — `make serve-counter`, then open the printed URL

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
| `wui.KeyMsg{Key, Rune}` | Key press on both platforms (key names: `"enter"`, `"ctrl+c"`, `"tab"`, `"backspace"`, `"up"`, `"down"`, etc.). In the browser, keys aimed at a focused input/textarea/select stay with that element and are not dispatched |
| `wui.ResizeMsg{Width, Height}` | Terminal or window resize |
| `wui.InputMsg{ID, Value}` | Input value changed with no `OnChange` callback set |
| `wui.ToggleMsg{ID, Checked}` | Checkbox toggled with no `OnToggle` callback set |
| `wui.SubmitMsg{FormValues}` | Form submitted with no `OnSubmit` callback set |
| `wui.ClickMsg{TargetID}` | Link clicked |
| `wui.NavigateMsg{Path}` | Browser URL hash names a path — initial load or back/forward (WASM only; see `wui.Pather`) |

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

// With an explicit identity (needed when several buttons share a label,
// e.g. a per-row "✕" delete button):
wui.Button("✕", onDelete, wui.WithID("del-"+itemID))
```

**TUI**: rendered as `[ Label ]`, focusable via Tab, activated via Enter or Space.
**HTML**: `<button onclick=…>`.

A button's focus identity is its `WithID` value when set, else its label. Buttons that share a label *and* have no ID share TUI focus and a DOM id — give repeated buttons IDs.

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

`OnChange` fires on every keystroke. `OnSubmit` fires on Enter, on both platforms.

Input values are **controlled with a twist**: in-progress typing is never clobbered by re-renders, but when your app changes `WithValue` programmatically (e.g. clearing a field after submit), the new value is applied. This works identically in TUI and HTML.

### Checkbox

```go
wui.Checkbox("done-42", "Buy milk", item.Done, func(checked bool) wui.Msg {
    return toggleMsg{id: 42, done: checked}
})

// Without a callback, toggling emits wui.ToggleMsg{ID, Checked}:
wui.Checkbox("opt-in", "Subscribe", m.optIn, nil)

// Disabled:
wui.Checkbox("locked", "Read-only", true, nil, wui.CheckboxDisabled())
```

**TUI**: rendered as `[x] Label` / `[ ] Label`, focusable via Tab, toggled via Enter or Space.
**HTML**: a real `<label><input type="checkbox">` — Space toggles, clicking the label toggles, screen readers announce it. The base CSS hides the native box and draws the same `[x]` mark as the TUI.

The `id` must be stable and unique per view (like `Input`).

### Card

A titled, bordered panel — the standard dashboard building block:

```go
wui.Card("Weather", body)

// With style (Border is implied; Padding, Margin, Width, BorderColor apply):
wui.Card("Weather", body, wui.WithCardStyle(wui.Style{Padding: [4]int{0, 1, 0, 1}}))
```

**TUI**: rounded border with the bold title embedded in the top border line:

```
╭ Weather ─────────╮
│ 21.3°C  clear    │
╰──────────────────╯
```

**HTML**: `<fieldset><legend>Weather</legend>…</fieldset>` — the semantic element for a titled group, styled by the base CSS to match.

### Form

Wraps inputs in a `<form>` (HTML) or a vertical container (TUI). The `OnSubmit` callback receives all input values by ID when the form is submitted.

```go
wui.Form(
    func(vals map[string]string) wui.Msg {
        return loginMsg{user: vals["user"], pass: vals["pass"]}
    },
    wui.Input("user", wui.WithPlaceholder("Username")),
    wui.Input("pass", wui.WithPlaceholder("Password"), wui.WithPassword()),
    wui.Button("Log in", nil), // nil OnClick inside a Form = submit button
)
```

Submission works the same on both platforms:
- A button with `nil` OnClick inside a Form acts as a **submit button** — like `<button type="submit">` in HTML; in TUI, Enter on the focused button submits the form.
- **Enter in an input** inside the form also submits it (unless the input has its own `OnSubmit`, which then takes precedence).

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
    20, // max height in terminal rows, both platforms
)
```

**TUI**: `lipgloss.MaxHeight`. **HTML**: `overflow:auto; max-height:Nlh` (an `em` fallback covers browsers without `lh`).

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
    Padding     [4]int   // top, right, bottom, left — cells
    Margin      [4]int   // top, right, bottom, left — cells
    Width       int      // 0 = auto — cells
    Height      int      // 0 = auto — cells
    Border      bool
    BorderColor Color
}
```

**Color values**:
- Hex strings (`"#ff0000"`) work on both platforms.
- Named CSS colors (`"red"`, `"cornflowerblue"`) work on both platforms.
- ANSI indices `"0"`–`"15"` are translated to hex for HTML, passed to lipgloss as-is for TUI.

**Units**: all sizing values (`Width`, `Height`, `Padding`, `Margin`, `Box` gap) are terminal cells. The HTML renderer translates them to the closest CSS analogues — `ch` horizontally and `lh` (line height, with an `em` fallback) vertically — so the same numbers produce visually similar layouts on both platforms.

---

## Keyboard navigation (TUI)

| Key | Action |
|---|---|
| `Tab` | Focus next focusable element |
| `Shift+Tab` | Focus previous focusable element |
| `Enter` / `Space` | Activate focused button, link, or checkbox; Enter submits a focused input (its `OnSubmit`, else the enclosing form) |
| `Ctrl+C` | Quit (intercepted by wui, not forwarded to `Update`) |

Mouse clicks are also wired in the TUI: clicking a button, link, checkbox, or input activates it and moves keyboard focus to it, like the browser.

Focusable elements are collected in tree order: text inputs, checkboxes, buttons, and links. Button focus keys derive from `WithID` when set, else from the label — give repeated buttons IDs.

Your `Update` receives `wui.KeyMsg` for any key that isn't consumed by focus management or input editing — in the TUI and in the browser alike (WASM builds listen on `document` and skip keys aimed at an editable element; alt/meta chords and unmapped named keys stay with the browser, and space, backspace, `'`, `/`, and tab have their page defaults suppressed so they behave like the TUI). Use it for global shortcuts:

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

## Running TUI and web together

A TUI program can serve the WASM build of the same app over HTTP while it runs:

```go
wui.NewProgram(model{}, wui.WithWebServer(":8765", "example/counter/web")).Run()

// Or pass "" to pick a port automatically: binds loopback only, on the
// first free port in 8765-8864 (OS-assigned as a last resort).
wui.NewProgram(model{}, wui.WithWebServer("", "example/counter/web")).Run()
```

The examples expose this as a `-serve` flag via the `wui.ServeFlag` helper:

```bash
make tui-counter                        # builds the WASM bundle, then:
go run ./example/counter -serve         # auto-pick a free port (loopback only)
go run ./example/counter -serve=:8765   # explicit address
```

Note the `=` in the explicit form — because bare `-serve` is allowed, the flag package treats it as boolean-style, so `-serve :8765` would not parse. In your own main:

```go
serve := wui.ServeFlag("serve", "also serve the web build")
flag.Parse()
var opts []wui.Option
if serve.Enabled {
    opts = append(opts, wui.WithWebServer(serve.Addr, "path/to/web"))
}
```

While serving, the TUI always shows a **status bar** pinned to the bottom of the screen with the URL of the equivalent web page:

```
 web ⇒ http://localhost:8765/
```

`WithWebServer` is a no-op in WASM builds, so shared `main` code can pass it unconditionally.

### Equivalent paths (Pather)

If your model implements `wui.Pather`, both platforms agree on *where* in the app you are:

```go
func (m model) Path() string {
    if m.state == stateResult {
        return "/result"
    }
    return "/"
}
```

- **TUI**: the status bar link includes the path — `http://localhost:8765/#/result`.
- **Browser**: `location.hash` is kept in sync with `Path()` after every update, and a `wui.NavigateMsg{Path}` is dispatched on initial load and on hash changes (back/forward), so deep links and history work:

```go
case wui.NavigateMsg:
    if v.Path == "/result" && m.message != "" {
        m.state = stateResult
    } else {
        m.state = stateForm
    }
```

---

## WASM build

### Quick start

```bash
make serve-counter   # builds WASM + copies wasm_exec.js + serves on a free port
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

Every example runs three ways:

```bash
make run-NAME     # terminal
make serve-NAME   # browser, auto-picked free port (printed on start)
make tui-NAME     # terminal + web server + status bar link
```

### Counter (`example/counter/`)

Minimal example: increment/decrement/reset buttons, styled title, row layout.

### Contact form (`example/form/`)

Two text inputs, validation, form submission via submit button or Enter, result view with a data table. Implements `wui.Pather` — the result screen lives at `#/result` in the browser and the TUI status bar links to it.

TUI interaction:
- `Tab` to cycle between Name input, Email input, Submit button
- `Enter` on the Submit button (or in an input) to submit
- `q` or Back button to return to the form

### Todo (`example/todo/`)

Add items via input + submit button (or Enter), remove items with per-item buttons, scrollable list, programmatic input clearing after submit.

### Timer (`example/timer/`)

A stopwatch: Cmd-driven self-rescheduling ticks, start/stop/reset, bordered display.

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
- **Button focus keys derive from labels by default** — two buttons with the same label and no `WithID` in one view share TUI focus; give repeated buttons IDs.
- **No theme system** — style is applied per-element inline; the WASM base stylesheet (see `WithoutBaseCSS`) is the only global styling hook.
- **ANSI color fidelity** — ANSI indices 0–15 map to fixed hex values in HTML; exact colours depend on the terminal's colour scheme in TUI.

---

## License

MIT
