package server

import (
	// Stdlib
	"crypto/rand"
	"encoding/hex"
	"net/http"

	// Internal
	"github.com/salsaflow/salsaflow-server/server/common"

	// Vendor
	"github.com/gorilla/mux"
)

const TokenByteLen = 16

const TokenHeader = "X-SalsaFlow-Token"

type API struct {
	ds DataStore
}

// HandleGenerateToken handles /users/{userId}/generateToken
func (api *API) HandleGenerateToken(rw http.ResponseWriter, r *http.Request) {
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
	if err := api.ds.SaveUser(user); err != nil {
		httpError(rw, r, err)
		return
	}

	// Write the response.
	resp := struct {
		Token string `json:"token"`
	}{
		tokenString,
	}

	body, err := json.Marshal(resp)
	if err != nil {
		httpError(rw, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	io.Copy(rw, bytes.NewReader(body))
}

func generateAccessToken() (token string, err error) {
	// Generate new token.
	tok := make([]byte, TokenByteLen)
	if _, err := rand.Read(token); err != nil {
		return "", err
	}

	// Encode the token using hex and return.
	return hex.EncodeToString(tok), nil
}
