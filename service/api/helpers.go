package api

import (
	"encoding/json"
	"errors"
	"net/http"
)

// decodeJSONBody decodifica il corpo della richiesta nella struct dst
func (rt *_router) decodeJSONBody(w http.ResponseWriter, r *http.Request, dst interface{}) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	err := dec.Decode(dst)
	if err != nil {
		// Abbiamo rimosso il log qui per evitare l'errore di import inutilizzato
		w.WriteHeader(http.StatusBadRequest)
		return errors.New("invalid json body")
	}
	return nil
}

// checkAuth verifica l'header Authorization
func (rt *_router) checkAuth(w http.ResponseWriter, r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return "", errors.New("missing authorization header")
	}

	const prefix = "Bearer "
	var token string
	if len(authHeader) > len(prefix) && authHeader[:len(prefix)] == prefix {
		token = authHeader[len(prefix):]
	} else {
		token = authHeader
	}
	
	return token, nil
}