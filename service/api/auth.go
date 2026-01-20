package api

import (
	"encoding/json"
	"net/http"
	"github.com/julienschmidt/httprouter"
)

// doLogin gestisce POST /session
func (rt *_router) doLogin(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	// Definiamo la struttura del JSON in arrivo
	var user struct {
		Name string `json:"name"`
	}

	// Decodifichiamo il body
	if err := rt.decodeJSONBody(w, r, &user); err != nil {
		return // L'errore è già stato gestito (400) nell'helper
	}

	// Chiamiamo il database
	identifier, err := rt.db.DoLogin(user.Name)
	if err != nil {
		rt.baseLogger.WithError(err).Error("database error during login")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Prepariamo la risposta
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	
	// Inviamo il JSON con l'identifier
	_ = json.NewEncoder(w).Encode(map[string]string{
		"identifier": identifier,
	})
}