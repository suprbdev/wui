// Command serve is a static file server for wui WASM web builds.
//
// Usage: serve DIR [PORT]
//
// With no PORT, it mirrors wui's WithWebServer auto-pick: binds
// loopback only, on the first free port in the webserver package's
// auto range, falling back to an OS-assigned port when the whole range
// is busy. An explicit PORT binds all interfaces.
//
// Apps using wui as a framework can run it without checking out this
// repo:
//
//	go run github.com/suprbdev/wui/cmd/serve@latest path/to/web
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/suprbdev/wui/webserver"
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("serve: ")
	if len(os.Args) < 2 || len(os.Args) > 3 {
		log.Fatal("usage: serve DIR [PORT]")
	}
	dir := os.Args[1]
	if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
		log.Fatalf("not a directory: %s", dir)
	}

	addr := ""
	if len(os.Args) == 3 && os.Args[2] != "" {
		addr = ":" + os.Args[2]
	}
	ln, err := webserver.Listen(addr)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Serving %s at %s\n", dir, webserver.URL(ln.Addr()))
	log.Fatal(http.Serve(ln, http.FileServer(http.Dir(dir))))
}
