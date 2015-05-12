package main

import (
	// Stdlib
	"flag"

	// Vendor
	"github.com/codegangsta/negroni"
)

func main() {
	flagAddress := flag.String("addr", ":3000", "network address")

	n := negroni.Classic()
	n.Run(*flagAddress)
}
