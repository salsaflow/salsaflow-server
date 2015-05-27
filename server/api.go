package server

import (
	// Stdlib
	"bytes"
	"crypto/rand"
	"encoding/hex"
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

// GetGenerateToken handles GET /users/{userId}/generateToken
func (api *API) GetGenerateToken(rw http.ResponseWriter, r *http.Request) {
	// Get user ID.
	userId := mux.Vars(r)["userId"]

	// Fetch user record for the given ID.
	user, err := api.store.FindUserById(userId)
	if err != nil {
		httpError(rw, r, err)
		return
	}
	if user == nil {
		http.Error(rw, http.StatusText(http.StatusNotFound), http.StatusNotFound)
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

func generateAccessToken() (token string, err error) {
	// Generate new token.
	tok := make([]byte, TokenByteLen)
	if _, err := rand.Read(tok); err != nil {
		return "", err
	}

	// Encode the token using hex and return.
	return hex.EncodeToString(tok), nil
}
