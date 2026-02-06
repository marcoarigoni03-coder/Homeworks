package api

import (
	"github.com/julienschmidt/httprouter"
	"net/http"
)

func (rt *_router) liveness(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
	if err := rt.db.Ping(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
