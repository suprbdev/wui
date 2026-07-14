// Command timer is a stopwatch demonstrating Cmd-driven async work
// (self-rescheduling ticks) — identical in TUI and WASM builds.
package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/suprbdev/wui"
)

const tickInterval = 100 * time.Millisecond

type model struct {
	elapsed time.Duration
	running bool
}

func (m model) Init() wui.Cmd { return nil }

type tickMsg struct{}
type toggleMsg struct{}
type resetMsg struct{}

func tick() wui.Cmd {
	return func() wui.Msg {
		time.Sleep(tickInterval)
		return tickMsg{}
	}
}

func (m model) Update(msg wui.Msg) (wui.Model, wui.Cmd) {
	switch msg.(type) {
	case toggleMsg:
		m.running = !m.running
		if m.running {
			return m, tick()
		}
	case tickMsg:
		// A tick scheduled before Stop was pressed may still arrive;
		// only count and reschedule while running.
		if m.running {
			m.elapsed += tickInterval
			return m, tick()
		}
	case resetMsg:
		m.elapsed = 0
	}
	return m, nil
}

func (m model) View() wui.Element {
	label := "Start"
	if m.running {
		label = "Stop"
	}
	return wui.BoxStyled(wui.Column, 1, wui.Style{Padding: [4]int{1, 2, 1, 2}},
		wui.Text("Stopwatch", wui.WithTextStyle(wui.Style{Bold: true})),
		wui.Text(fmt.Sprintf("%6.1fs", m.elapsed.Seconds()),
			wui.WithTextStyle(wui.Style{Border: true, Padding: [4]int{0, 1, 0, 1}})),
		wui.BoxStyled(wui.Row, 1, wui.Style{},
			wui.Button(label, func() wui.Msg { return toggleMsg{} }),
			wui.Button("Reset", func() wui.Msg { return resetMsg{} }),
		),
	)
}

func main() {
	serve := flag.String("serve", "", "also serve the web build at this address (e.g. :8765)")
	flag.Parse()

	var opts []wui.Option
	if *serve != "" {
		opts = append(opts, wui.WithWebServer(*serve, "example/timer/web"))
	}
	if err := wui.NewProgram(model{}, opts...).Run(); err != nil {
		panic(err)
	}
}
