// ffe, short of Femto Front-End is a minimalistic frontend for serving static
// contents from given directories.
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/honr/vulcan/static"
)

var (
	port       = flag.String("port", ":8000", "Port, and maybe hostname, to listen to")
	devMode    = flag.Bool("dev", false, "Dev mode")
	staticDirs = flag.String("dirs", "static", "Colon-separated directories containing static resources such as html, javascript, and css files.  Latter directories win when there are duplicate files.")
	index      = flag.String("index", "/index.htl", "Default file, for instance /index.html")
)

func main() {
	flag.Parse()
	m, err := static.HandlersFromDirs(strings.Split(*staticDirs, ":"), *devMode)
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

	fmt.Println("listening on", *port)
	http.ListenAndServe(*port, nil)
}
