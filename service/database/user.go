package database

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/google/uuid" // Assicurati di aver fatto 'go get github.com/google/uuid'
)

// DoLogin implementa la logica di login semplificato
func (db *appdbimpl) DoLogin(username string) (string, error) {
	var identifier string

	// 1. Cerchiamo se l'utente esiste già tramite lo username
	err := db.c.QueryRow("SELECT identifier FROM users WHERE username = ?", username).Scan(&identifier)

	if errors.Is(err, sql.ErrNoRows) {
		// 2. L'utente non esiste: lo creiamo (Registrazione)
		
		// Generiamo un nuovo identificatore univoco (questo sarà il Bearer token)
		newID := uuid.New().String()

		// Inseriamo il nuovo utente
		_, err = db.c.Exec("INSERT INTO users (identifier, username) VALUES (?, ?)", newID, username)
		if err != nil {
			return "", err
		}
		
		return newID, nil // Ritorniamo il nuovo ID appena creato
	} else if err != nil {
		// Errore generico del database
		return "", err
	}

	// 3. L'utente esisteva: ritorniamo il suo identificatore esistente
	return identifier, nil
}

// SetMyUserName permette all'utente di cambiare il proprio nome
func (db *appdbimpl) SetMyUserName(identifier string, newName string) error {
	// Controlliamo che il nuovo nome non sia vuoto (o altre validazioni se vuoi)
	if strings.TrimSpace(newName) == "" {
		return errors.New("username cannot be empty")
	}

	// Proviamo ad aggiornare il nome.
	// La query fallirà se il nome è già preso da qualcun altro (vincolo UNIQUE su username)
	res, err := db.c.Exec("UPDATE users SET username = ? WHERE identifier = ?", newName, identifier)
	if err != nil {
		return err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if affected == 0 {
		return errors.New("user not found")
	}

	return nil
}