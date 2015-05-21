// +build heroku

package main

import (
	// Stdlib
	"os"
	"strings"

	// Internal
	"github.com/salsaflow/salsaflow-server/server"

	// Vendor
	oauth2 "github.com/goincremental/negroni-oauth2"
)

func NewServer() *server.Server {
	var (
		addr = ":" + os.Getenv("PORT")

		canonicalURL = os.Getenv("CANONICAL_URL")

		clientId     = os.Getenv("OAUTH2_CLIENT_ID")
		clientSecret = os.Getenv("OAUTH2_CLIENT_SECRET")
	)

	if !strings.HasSuffix(canonicalURL, "/") {
		canonicalURL += "/"
	}

	oauth2Config := &oauth2.Config{
		ClientID:     clientId,
		ClientSecret: clientSecret,
		RedirectURL:  canonicalURL + "oauth2callback",
		Scopes:       []string{"email"},
	}

	return server.New(oauth2Config, server.SetAddress(addr))
}
