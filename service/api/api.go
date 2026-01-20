package api

import (
	"embed"
	"errors"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/julienschmidt/httprouter"
	"github.com/marcoarigoni03-coder/Homeworks/service/database"
	"github.com/sirupsen/logrus"
)

//go:embed dist/*
var frontendEmbed embed.FS

type Config struct {
	Logger   logrus.FieldLogger
	Database database.AppDatabase
}

type Router interface {
	Handler() http.Handler
	Close() error
}

func New(cfg Config) (Router, error) {
	if cfg.Logger == nil {
		return nil, errors.New("logger is required")
	}
	if cfg.Database == nil {
		return nil, errors.New("database is required")
	}

	router := httprouter.New()
	router.RedirectTrailingSlash = false
	router.RedirectFixedPath = false

	rt := &_router{
		router:     router,
		baseLogger: cfg.Logger,
		db:         cfg.Database,
	}

	rt.registerRoutes()
	return rt, nil
}

type _router struct {
	router     *httprouter.Router
	baseLogger logrus.FieldLogger
	db         database.AppDatabase
}

func (rt *_router) Handler() http.Handler {
	return rt.router
}

func (rt *_router) Close() error {
	return nil
}

func (rt *_router) registerRoutes() {
	// --- API ROUTES ---
	rt.router.GET("/status", rt.getStatus)
	rt.router.POST("/session", rt.doLogin)
	rt.router.PUT("/users/me/name", rt.setMyUserName)
	rt.router.GET("/conversations", rt.getMyConversations)
	rt.router.GET("/conversations/:conversationId", rt.getConversation)
	rt.router.POST("/conversations/:conversationId/messages", rt.sendMessage)

	// --- GESTORE UNIVERSALE FILE STATICI ---
	// Gestisce TUTTO quello che non è una rotta API (Assets, Bootstrap, Immagini, Favicon)
	rt.router.NotFound = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// 1. Cerchiamo se esiste un file fisico corrispondente alla richiesta
		// Es: richiesta "/bootstrap.css" -> cerchiamo "dist/bootstrap.css"
		path := r.URL.Path
		if strings.HasPrefix(path, "/") {
			path = path[1:] // Rimuovi lo slash iniziale
		}
		fullPath := "dist/" + path

		fileData, err := frontendEmbed.ReadFile(fullPath)

		if err == nil {
			// --- FILE TROVATO! Serviamolo col tipo giusto ---
			ext := strings.ToLower(filepath.Ext(fullPath))
			switch ext {
			case ".css":
				w.Header().Set("Content-Type", "text/css")
			case ".js":
				w.Header().Set("Content-Type", "application/javascript")
			case ".png":
				w.Header().Set("Content-Type", "image/png")
			case ".jpg", ".jpeg":
				w.Header().Set("Content-Type", "image/jpeg")
			case ".svg":
				w.Header().Set("Content-Type", "image/svg+xml")
			case ".json":
				w.Header().Set("Content-Type", "application/json")
			case ".ico":
				w.Header().Set("Content-Type", "image/x-icon")
			default:
				w.Header().Set("Content-Type", "text/plain")
			}
			w.Write(fileData)
			return
		}

		// 2. FILE NON TROVATO -> È una rotta frontend (es: /profile), serviamo index.html
		htmlData, err := frontendEmbed.ReadFile("dist/index.html")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("ERRORE CRITICO: Cartella 'dist' non trovata o vuota."))
			return
		}
		w.Header().Set("Content-Type", "text/html")
		w.Write(htmlData)
	})
}
