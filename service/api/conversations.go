package api

import (
	"encoding/json"
	"net/http"
	"github.com/julienschmidt/httprouter"
)

// getMyConversations: GET /conversations
func (rt *_router) getMyConversations(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	// 1. Autenticazione
	userID, err := rt.checkAuth(w, r)
	if err != nil { return }

	// 2. Database
	conversations, err := rt.db.GetMyConversations(userID)
	if err != nil {
		rt.baseLogger.WithError(err).Error("error loading conversations")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// 3. Risposta
	w.Header().Set("Content-Type", "application/json")
	if conversations == nil {
		w.Write([]byte("[]")) // Array vuoto se nil
	} else {
		_ = json.NewEncoder(w).Encode(conversations)
	}
}

// getConversation: GET /conversations/:conversationId
func (rt *_router) getConversation(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	userID, err := rt.checkAuth(w, r)
	if err != nil { return }

	conversationID := ps.ByName("conversationId")

	// Recuperiamo dettagli e messaggi
	conversation, messages, err := rt.db.GetConversation(conversationID, userID)
	if err != nil {
		rt.baseLogger.WithError(err).Error("error loading conversation details")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Costruiamo la risposta JSON combinata
	type Response struct {
		ID       string      `json:"conversationId"`
		Name     string      `json:"name"`
		Messages interface{} `json:"messages"`
	}
	
	resp := Response{
		ID:       conversation.ID,
		Name:     conversation.Name,
		Messages: messages,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// sendMessage: POST /conversations/:conversationId/messages
func (rt *_router) sendMessage(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	userID, err := rt.checkAuth(w, r)
	if err != nil { return }

	conversationID := ps.ByName("conversationId")

	// Leggiamo il body
	var body struct {
		Content string `json:"content"`
	}
	if err := rt.decodeJSONBody(w, r, &body); err != nil { return }

	// Salviamo nel DB
	msg, err := rt.db.SendMessage(conversationID, userID, body.Content)
	if err != nil {
		rt.baseLogger.WithError(err).Error("error sending message")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Rispondiamo con il messaggio creato (201 Created)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(msg)
}