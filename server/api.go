package server

import (
	"crypto/rand"
	"encoding/hex"
)

const TokenByteLen = 16

const TokenHeader = "X-SalsaFlow-Token"

type API struct {
	ds DataStore
}

// HandleGenerateToken handles /users/{userId}/generateToken
func (api *API) HandleGenerateToken(w http.ResponseWriter, r *http.Request) {
	// Get user ID.
	userId := mux.Vars(r)["userId"]

	// Generate new token.
	token := make([]byte, TokenByteLen)
	if _, err := rand.Read(token); err != nil {
		httpError(w, r, err)
		return
	}

	// Encode the token using hex.
	tokenString := hex.EncodeToString(token)

	// Save the new token.
	if err := api.ds.SetToken(userId, tokenString); err != nil {
		httpError(w, r, err)
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
		httpError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	io.Copy(w, bytes.NewReader(body))
}
