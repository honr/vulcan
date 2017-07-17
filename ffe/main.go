// ffe, short of Femto Front-End is a minimalistic frontend for serving static
// contents from given directories.  Examples of usage:
//   1. Serve the current directory and anything under it.
//   $ ffe --addr=:8000
//   2. Serve files from ./tmp/hello-world, ./web/common, and under them.
//   $ ffe --addr=localhost:8000 web/common tmp/hello-world
//   3. Serve files in non-dev-mode (read each file only once, serve from
//   memory).
//   $ ffe --addr=:8011 --dev=false #
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/honr/vulcan/static"
)

var (
	addr    = flag.String("addr", "", "addr is the port and maybe hostname to listen to.  E.g., :8000 or localhost:8000")
	devMode = flag.Bool("dev-mode", true, "Whether run in dev mode, where *registered* resources will be reread on each refresh.  If you add a new resource file, you need to restart the server for it to take effect.")
	index   = flag.String("index", "/index.htl", "Default file, for instance /index.html")
)

func main() {
	flag.Parse()
	// staticDirs is the colon-separated list of directories containing static
	// resources such as html, javascript, and css files.  Latter directories
	// win when there are duplicate files.  When not specfied, current directory
	// is read and served.
	staticDirs := flag.Args()
	if len(staticDirs) == 0 {
		staticDirs = []string{"."}
	}
	if *addr == "" {
		log.Fatal("Must provide a port to listen to, such as :8000")
	}

	m, err := static.HandlersFromDirs(staticDirs, *devMode)
	if err != nil {
		log.Fatal(err)
	}
	for p, h := range m {
		fmt.Println("registered path:", p)
		http.HandleFunc(p, h)
		if p == *index {
			http.HandleFunc("/", h)
		}
	}

	fmt.Println("listening on", *addr)
	err = http.ListenAndServe(*addr, nil)
	if err != nil {
		log.Fatal(err)
	}
}
