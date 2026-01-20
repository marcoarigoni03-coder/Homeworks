package api

import (
	"net/http"
	"github.com/julienschmidt/httprouter"
)

// setMyUserName gestisce PUT /users/me/name
func (rt *_router) setMyUserName(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	// 1. Autenticazione: chi sta facendo la richiesta?
	userID, err := rt.checkAuth(w, r)
	if err != nil {
		return // Risposta 401 già inviata
	}

	// 2. Parsing del body
	var body struct {
		Name string `json:"name"`
	}
	if err := rt.decodeJSONBody(w, r, &body); err != nil {
		return
	}

	// 3. Aggiornamento nel Database
	err = rt.db.SetMyUserName(userID, body.Name)
	if err != nil {
		rt.baseLogger.WithError(err).Error("error updating username")
		// Se l'errore è "user not found" o "conflict", dovremmo gestire gli status code specifici.
		// Per semplicità qui ritorniamo 500, ma potresti migliorare.
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// 4. Risposta successo
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("{}"))
}