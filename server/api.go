package server

import (
	// Stdlib
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	// Vendor
	"github.com/gorilla/mux"
)

const TokenByteLen = 16

const TokenHeader = "X-SalsaFlow-Token"

type API struct {
	store DataStore
}

// GetMe handles GET /me.
//
// It is possible to authenticate with a session here, no need for a token.
func (api *API) GetMe(rw http.ResponseWriter, r *http.Request) {
	// Get the requesting user record.
	user, err := getRequester(r, api.store)
	if err != nil {
		httpError(rw, r, err)
		return
	}
	if user == nil {
		httpStatus(rw, http.StatusForbidden)
		return
	}

	// Write the response.
	body, err := json.Marshal(user)
	if err != nil {
		httpError(rw, r, err)
		return
	}

	rw.Header().Set("Content-Type", "application/json")
	io.Copy(rw, bytes.NewReader(body))
}

// GetGenerateToken handles GET /users/{userId}/generateToken
func (api *API) GetGenerateToken(rw http.ResponseWriter, r *http.Request) {
	// Get the requesting user record.
	user, err := getRequester(r, api.store)
	if err != nil {
		httpError(rw, r, err)
		return
	}
	if user == nil {
		httpStatus(rw, http.StatusForbidden)
		return
	}

	// Get user ID.
	userId := mux.Vars(r)["userId"]

	// Make sure the user to be modified matches the requester.
	// We don't want the users to be modifying each other.
	if user.Id != userId {
		httpStatus(rw, http.StatusForbidden)
		return
	}

	// Generate new token.
	token, err := generateAccessToken()
	if err != nil {
		httpError(rw, r, err)
		return
	}

	// Save the new token.
	user.Token = token
	if err := api.store.SaveUser(user); err != nil {
		httpError(rw, r, err)
		return
	}

	// Write the response.
	resp := struct {
		Token string `json:"token"`
	}{
		token,
	}

	body, err := json.Marshal(resp)
	if err != nil {
		httpError(rw, r, err)
		return
	}

	rw.Header().Set("Content-Type", "application/json")
	io.Copy(rw, bytes.NewReader(body))
}
