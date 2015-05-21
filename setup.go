// +build !heroku

package main

import (
	// Stdlib
	"flag"

	// Internal
	"github.com/salsaflow/salsaflow-server/server"

	// Vendor
	oauth2 "github.com/goincremental/negroni-oauth2"
)

func NewServer() *server.Server {
	flagAddress := flag.String("addr", server.DefaultAddress, "network address to listen on")
	flagTimeout := flag.Duration("timeout", server.DefaultTimeout, "server shutdown timeout")
	flag.Parse()

	oauth2Config := &oauth2.Config{
		ClientID:     "someID",
		ClientSecret: "someSecret",
		RedirectURL:  "http://localhost:3000",
		Scopes:       []string{"email"},
	}

	return server.New(
		oauth2Config,
		server.SetAddress(*flagAddress),
		server.SetShutdownTimeout(*flagTimeout))
}
