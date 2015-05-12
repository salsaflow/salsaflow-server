package main

import (
	// Stdlib
	"flag"
	"time"

	// Vendor
	"github.com/codegangsta/negroni"
	"gopkg.in/tylerb/graceful.v1"
)

func main() {
	flagAddress := flag.String("addr", ":3000", "network address")

	n := negroni.Classic()
	graceful.Run(*flagAddress, 3*time.Second, n)
}
