package server

import (
	// Stdlib
	"net/http"

	// Vendor
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
)

func NewOAuth2HttpClient(token oauth2.Token) *http.Client {
	return oauth2.NewClient(context.Background(), token)
}
