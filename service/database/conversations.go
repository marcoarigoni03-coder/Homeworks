package database

import (
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
)

// CreateGroup crea un nuovo gruppo e aggiunge il creatore come partecipante
func (db *appdbimpl) CreateGroup(ownerID string, groupName string) (string, error) {
	groupID := uuid.New().String()

	// Creiamo la conversazione di tipo GRUPPO (is_group = 1)
	_, err := db.c.Exec("INSERT INTO conversations (id, name, is_group) VALUES (?, ?, 1)", groupID, groupName)
	if err != nil {
		return "", err
	}

	// Aggiungiamo il creatore ai partecipanti
	_, err = db.c.Exec("INSERT INTO participants (conversation_id, user_id) VALUES (?, ?)", groupID, ownerID)
	if err != nil {
		return "", err
	}

	return groupID, nil
}

// SendMessage aggiunge un messaggio al database
func (db *appdbimpl) SendMessage(conversationID string, senderID string, content string) (Message, error) {
	// Verifichiamo prima che l'utente faccia parte della conversazione
	var exists int
	err := db.c.QueryRow("SELECT 1 FROM participants WHERE conversation_id = ? AND user_id = ?", conversationID, senderID).Scan(&exists)
	if err != nil {
		return Message{}, errors.New("user not in conversation")
	}

	msgID := uuid.New().String()
	timestamp := time.Now()

	_, err = db.c.Exec(`INSERT INTO messages (id, conversation_id, sender_id, content, timestamp, status) 
		VALUES (?, ?, ?, ?, ?, 1)`, msgID, conversationID, senderID, content, timestamp)
	
	if err != nil {
		return Message{}, err
	}

	return Message{
		ID:        msgID,
		SenderID:  senderID,
		Content:   content,
		Timestamp: timestamp,
		Status:    1,
	}, nil
}

// GetMyConversations recupera la lista delle chat dell'utente
// NOTA: Questa è una versione semplificata per iniziare. 
// Recupera le chat ma non ancora l'ultimo messaggio per non complicare troppo la SQL ora.
func (db *appdbimpl) GetMyConversations(userID string) ([]Conversation, error) {
	rows, err := db.c.Query(`
		SELECT c.id, c.name, c.is_group, c.photo 
		FROM conversations c
		JOIN participants p ON c.id = p.conversation_id
		WHERE p.user_id = ?`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var conversations []Conversation
	for rows.Next() {
		var c Conversation
		// Le colonne SQL possono essere NULL, Go vuole tipi concreti.
		// Per semplicità usiamo variabili temporanee per gestire i NULL se necessario,
		// ma qui assumiamo che name/photo siano stringhe vuote se null.
		var name, photo sql.NullString
		
		err = rows.Scan(&c.ID, &name, &c.IsGroup, &photo)
		if err != nil {
			return nil, err
		}
		
		c.Name = name.String
		c.Photo = photo.String
		conversations = append(conversations, c)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}
	return conversations, nil
}

// CreateConversation (Direct 1-to-1) - Placeholder per completare l'interfaccia
func (db *appdbimpl) CreateConversation(ownerID string, otherUserID string) (string, error) {
	// Qui dovresti controllare se esiste già una chat tra i due.
	// Per ora creiamo una nuova chat.
	chatID := uuid.New().String()
	
	// Creiamo la chat
	_, err := db.c.Exec("INSERT INTO conversations (id, is_group) VALUES (?, 0)", chatID)
	if err != nil { return "", err }

	// Aggiungiamo i due partecipanti
	_, err = db.c.Exec("INSERT INTO participants (conversation_id, user_id) VALUES (?, ?), (?, ?)", 
		chatID, ownerID, chatID, otherUserID)
	if err != nil { return "", err }

	return chatID, nil
}

// GetConversation - Placeholder
func (db *appdbimpl) GetConversation(conversationID string, userID string) (Conversation, []Message, error) {
	// Implementazione basilare per non rompere la build
	// Recuperiamo i messaggi
	rows, err := db.c.Query("SELECT id, sender_id, content, timestamp FROM messages WHERE conversation_id = ? ORDER BY timestamp DESC", conversationID)
	if err != nil { return Conversation{}, nil, err }
	defer rows.Close()

	var msgs []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.SenderID, &m.Content, &m.Timestamp); err != nil {
			return Conversation{}, nil, err
		}
		msgs = append(msgs, m)
	}

	return Conversation{ID: conversationID}, msgs, nil
}