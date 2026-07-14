package wui

import "flag"

// AddrFlag is a flag.Value for optional-address flags such as -serve.
// It accepts three forms:
//
//	(absent)        Enabled=false — no web server
//	-serve          Enabled=true, Addr=""  — pick a free port automatically
//	-serve=:8765    Enabled=true, Addr=":8765"
//
// Because the bare form is allowed, an explicit address must use the
// "=" syntax ("-serve=:8765", not "-serve :8765").
type AddrFlag struct {
	Enabled bool
	Addr    string
}

// ServeFlag registers an AddrFlag named name on flag.CommandLine and
// returns it. Call before flag.Parse; afterwards, pass the result to
// WithWebServer when Enabled:
//
//	serve := wui.ServeFlag("serve", "also serve the web build (optionally -serve=:8765)")
//	flag.Parse()
//	var opts []wui.Option
//	if serve.Enabled {
//		opts = append(opts, wui.WithWebServer(serve.Addr, "path/to/web"))
//	}
func ServeFlag(name, usage string) *AddrFlag {
	f := &AddrFlag{}
	flag.Var(f, name, usage)
	return f
}

func (f *AddrFlag) String() string {
	if f == nil || !f.Enabled {
		return ""
	}
	if f.Addr == "" {
		return "auto"
	}
	return f.Addr
}

func (f *AddrFlag) Set(s string) error {
	switch s {
	// The flag package passes "true" for a bare boolean-style flag;
	// accept "false" too for -serve=false symmetry.
	case "true":
		f.Enabled = true
		f.Addr = ""
	case "false":
		f.Enabled = false
		f.Addr = ""
	default:
		f.Enabled = true
		f.Addr = s
	}
	return nil
}

// IsBoolFlag lets the flag package accept the bare form (-serve) with
// no value.
func (f *AddrFlag) IsBoolFlag() bool { return true }
