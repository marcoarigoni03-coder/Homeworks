package database

import (
	"database/sql"
	"errors"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

// AppDatabase è l'interfaccia principale per il tuo database.
type AppDatabase interface {
	Ping() error

	// Utenti
	DoLogin(username string) (string, error)
	SetMyUserName(identifier string, newName string) error

	// Conversazioni e Messaggi
	CreateConversation(ownerID string, otherUserID string) (string, error) // Chat 1-to-1
	CreateGroup(ownerID string, groupName string) (string, error)          // Gruppo
	GetMyConversations(userID string) ([]Conversation, error)
	GetConversation(conversationID string, userID string) (Conversation, []Message, error)
	
	// Messaggistica
	SendMessage(conversationID string, senderID string, content string) (Message, error)
}

type appdbimpl struct {
	c *sql.DB
}

func New(dbPath string) (AppDatabase, error) {
	if dbPath == "" {
		return nil, errors.New("path del database richiesto")
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	// Abilita Foreign Keys (cruciale!)
	if _, err := db.Exec(`PRAGMA foreign_keys = ON;`); err != nil {
		return nil, fmt.Errorf("error setting pragmas: %w", err)
	}

	if err := createTables(db); err != nil {
		return nil, fmt.Errorf("error creating database structure: %w", err)
	}

	return &appdbimpl{
		c: db,
	}, nil
}

func (db *appdbimpl) Ping() error {
	return db.c.Ping()
}

func createTables(db *sql.DB) error {
	// 1. Tabella USERS
	usersTable := `CREATE TABLE IF NOT EXISTS users (
		identifier TEXT NOT NULL PRIMARY KEY,
		username TEXT NOT NULL UNIQUE
	);`
	if _, err := db.Exec(usersTable); err != nil { return err }

	// 2. Tabella CONVERSATIONS
	conversationsTable := `CREATE TABLE IF NOT EXISTS conversations (
		id TEXT NOT NULL PRIMARY KEY,
		name TEXT, 
		is_group BOOLEAN NOT NULL DEFAULT 0,
		photo TEXT
	);`
	if _, err := db.Exec(conversationsTable); err != nil { return err }

	// 3. Tabella PARTICIPANTS (chi sta in quale chat)
	participantsTable := `CREATE TABLE IF NOT EXISTS participants (
		conversation_id TEXT NOT NULL,
		user_id TEXT NOT NULL,
		PRIMARY KEY (conversation_id, user_id),
		FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE,
		FOREIGN KEY (user_id) REFERENCES users(identifier) ON DELETE CASCADE
	);`
	if _, err := db.Exec(participantsTable); err != nil { return err }

	// 4. Tabella MESSAGES
	messagesTable := `CREATE TABLE IF NOT EXISTS messages (
		id TEXT NOT NULL PRIMARY KEY,
		conversation_id TEXT NOT NULL,
		sender_id TEXT NOT NULL,
		content TEXT NOT NULL,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		status INTEGER DEFAULT 1,
		FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE,
		FOREIGN KEY (sender_id) REFERENCES users(identifier) ON DELETE CASCADE
	);`
	if _, err := db.Exec(messagesTable); err != nil { return err }

	return nil
}