package database

import "time"

// Conversation rappresenta una chat (singola o gruppo)
type Conversation struct {
	ID          string    `json:"conversationId"`
	Name        string    `json:"name"`   // Nome del gruppo o dell'altro utente
	Photo       string    `json:"photo"`  // Foto (in base64 o URL)
	IsGroup     bool      `json:"isGroup"`
	UnreadCount int       `json:"unreadCount"`
	LastMessage *Message  `json:"lastMessage"` // Può essere null se la chat è vuota
}

// Message rappresenta un singolo messaggio
type Message struct {
	ID        string    `json:"messageId"`
	ConversationID string `json:"-"` // Non serve mandarlo nel JSON del messaggio
	SenderID  string    `json:"senderId"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	Status    int       `json:"status"` // 1=received, 2=read
}