// +build heroku

package main

import (
	// Stdlib
	"fmt"
	"os"
	"strings"

	// Internal
	"github.com/salsaflow/salsaflow-server/server"

	// Vendor
	oauth2 "github.com/goincremental/negroni-oauth2"
)

func NewServer() (*server.Server, error) {
	var (
		addr = ":" + os.Getenv("PORT")

		canonicalHostname = os.Getenv("CANONICAL_HOSTNAME")

		clientId     = os.Getenv("OAUTH2_CLIENT_ID")
		clientSecret = os.Getenv("OAUTH2_CLIENT_SECRET")
	)

	canonicalURL, err := url.Parse(canonicalHostname)
	if err != nil {
		return nil, err
	}
	canonicalURL.Scheme = "https"
	canonicalURL.Path = path.Join(u.Path, "oauth2callback")

	oauth2Config := &oauth2.Config{
		ClientID:     clientId,
		ClientSecret: clientSecret,
		RedirectURL:  canonicalURL.String(),
		Scopes:       []string{"email"},
	}

	return server.New(oauth2Config, server.SetAddress(addr)), nil
}
