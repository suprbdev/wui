//go:build !js

package wui

import (
	"strings"
	"testing"

	zone "github.com/lrstanley/bubblezone"
)

func newTestRenderer() *tuiRenderer {
	return newTUIRenderer(80, 24, zone.New())
}

// stripZones removes bubblezone's zero-width markers so assertions can
// match on visible text.
func stripZones(z *zone.Manager, s string) string {
	return z.Scan(s)
}

func TestRenderCardEmbedsTitleInTopBorder(t *testing.T) {
	r := newTestRenderer()
	out := r.Render(Card("Clock", Text("12:34:56")))
	lines := strings.Split(out, "\n")
	if len(lines) != 3 {
		t.Fatalf("want 3 lines, got %d:\n%s", len(lines), out)
	}
	if !strings.Contains(lines[0], "Clock") {
		t.Errorf("title not in top border: %q", lines[0])
	}
	if !strings.HasPrefix(lines[0], "╭") || !strings.HasSuffix(lines[0], "╮") {
		t.Errorf("top border corners missing: %q", lines[0])
	}
	if !strings.Contains(lines[1], "12:34:56") {
		t.Errorf("content missing: %q", lines[1])
	}
	if !strings.HasPrefix(lines[2], "╰") || !strings.HasSuffix(lines[2], "╯") {
		t.Errorf("bottom border corners missing: %q", lines[2])
	}
}

func TestRenderCardTitleWiderThanContent(t *testing.T) {
	r := newTestRenderer()
	out := r.Render(Card("A very long widget title", Text("x")))
	for i, line := range strings.Split(out, "\n") {
		// Border characters are single-cell; every line must be equally wide.
		if got, want := lineWidth(line), lineWidth(strings.Split(out, "\n")[0]); got != want {
			t.Errorf("line %d width %d != top border width %d", i, got, want)
		}
	}
}

func lineWidth(s string) int {
	// Strip ANSI (bold title) crudely: count runes outside escape sequences.
	w := 0
	inEsc := false
	for _, r := range s {
		switch {
		case inEsc:
			if r == 'm' {
				inEsc = false
			}
		case r == '\x1b':
			inEsc = true
		default:
			w++
		}
	}
	return w
}

func TestRenderCheckboxMarks(t *testing.T) {
	r := newTestRenderer()
	z := r.zones
	unchecked := stripZones(z, r.Render(Checkbox("cb1", "milk", false, nil)))
	if !strings.Contains(unchecked, "[ ] milk") {
		t.Errorf("unchecked render: %q", unchecked)
	}
	checked := stripZones(z, r.Render(Checkbox("cb1", "milk", true, nil)))
	if !strings.Contains(checked, "[x] milk") {
		t.Errorf("checked render: %q", checked)
	}
}

func TestCheckboxFocusableAndToggleMsg(t *testing.T) {
	tree := Box(Column,
		Checkbox("cb1", "milk", false, nil),
		Button("Save", func() Msg { return nil }),
	)
	fs := collectFocusables(tree)
	if len(fs) != 2 {
		t.Fatalf("want 2 focusables, got %d", len(fs))
	}
	if fs[0].ID != "cb1" {
		t.Errorf("checkbox focus id = %q", fs[0].ID)
	}
	msg, ok := fs[0].Activate().(ToggleMsg)
	if !ok {
		t.Fatalf("activate msg = %T, want ToggleMsg", fs[0].Activate())
	}
	if msg.ID != "cb1" || !msg.Checked {
		t.Errorf("ToggleMsg = %+v, want {cb1 true}", msg)
	}
}

func TestButtonFocusKeyPrefersID(t *testing.T) {
	if got := buttonFocusKey(ButtonEl{Label: "✕", ID: "del-5"}); got != "btn:del-5" {
		t.Errorf("with ID: %q", got)
	}
	if got := buttonFocusKey(ButtonEl{Label: "Save"}); got != "btn:Save" {
		t.Errorf("without ID: %q", got)
	}
}

func TestCardChildrenReachableByWalkers(t *testing.T) {
	tree := Card("T", Box(Column,
		Input("name"),
		Button("Go", func() Msg { return nil }),
	))
	fs := collectFocusables(tree)
	if len(fs) != 2 {
		t.Fatalf("focusables inside Card not collected: got %d, want 2", len(fs))
	}
	if findInputByID(tree, "name") == nil {
		t.Error("findInputByID does not descend into Card")
	}
}
