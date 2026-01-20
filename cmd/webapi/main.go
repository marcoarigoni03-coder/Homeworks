package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ardanlabs/conf"
	"github.com/sirupsen/logrus"

	// Import corretti per il tuo progetto
	"github.com/marcoarigoni03-coder/Homeworks/service/api"
	"github.com/marcoarigoni03-coder/Homeworks/service/database"
)

func main() {
	if err := run(); err != nil {
		fmt.Println("error: ", err)
		os.Exit(1)
	}
}

func run() error {
	var cfg struct {
		Web struct {
			APIHost         string        `conf:"default:0.0.0.0:3000"`
			DebugHost       string        `conf:"default:0.0.0.0:4000"`
			ReadTimeout     time.Duration `conf:"default:5s"`
			WriteTimeout    time.Duration `conf:"default:5s"`
			ShutdownTimeout time.Duration `conf:"default:5s"`
		}
		DB struct {
			Filename string `conf:"default:wasa.db"`
		}
	}

	// Parsing della configurazione
	if err := conf.Parse(os.Args[1:], "WASA", &cfg); err != nil {
		if err == conf.ErrHelpWanted {
			usage, err := conf.Usage("WASA", &cfg)
			if err != nil {
				return fmt.Errorf("generating config usage: %w", err)
			}
			fmt.Println(usage)
			return nil
		}
		return fmt.Errorf("parsing config: %w", err)
	}

	// Configurazione Logger
	logger := logrus.New()
	logger.SetOutput(os.Stdout)
	logger.SetLevel(logrus.DebugLevel)

	logger.Info("Starting service...")

	// Apertura Database
	appdb, err := database.New(cfg.DB.Filename)
	if err != nil {
		return fmt.Errorf("error creating database: %w", err)
	}

	// Creazione Router API
	apirouter, err := api.New(api.Config{
		Logger:   logger,
		Database: appdb,
	})
	if err != nil {
		return fmt.Errorf("error creating the API server instance: %w", err)
	}
	router := apirouter.Handler()

	// Configurazione Server HTTP
	apiserver := http.Server{
		Addr: cfg.Web.APIHost,
		// QUI STA LA CORREZIONE: Avvolgiamo il router con il gestore CORS
		Handler:           applyCORSHandler(router),
		ReadTimeout:       cfg.Web.ReadTimeout,
		ReadHeaderTimeout: cfg.Web.ReadTimeout,
		WriteTimeout:      cfg.Web.WriteTimeout,
	}

	serverErrors := make(chan error, 1)

	go func() {
		logger.Infof("API listening on %s", apiserver.Addr)
		serverErrors <- apiserver.ListenAndServe()
	}()

	// Gestione Segnali
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)

	case sig := <-shutdown:
		logger.Infof("signal %v received, starting shutdown", sig)

		ctx, cancel := context.WithTimeout(context.Background(), cfg.Web.ShutdownTimeout)
		defer cancel()

		if err := apiserver.Shutdown(ctx); err != nil {
			apiserver.Close()
			return fmt.Errorf("could not stop server gracefully: %w", err)
		}
	}

	return nil
}
