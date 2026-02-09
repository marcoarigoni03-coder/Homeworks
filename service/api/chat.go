package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"git.sapienzaapps.it/fantasticcoffee/fantastic-coffee-decaffeinated/service/api/reqcontext"
	"github.com/gofrs/uuid"
	"github.com/julienschmidt/httprouter"
)

type apiError struct {
	Error string `json:"error"`
}

type userDTO struct {
	ID          int64  `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
	Photo       string `json:"photo"`
}

type conversationDTO struct {
	ID          int64     `json:"id"`
	IsGroup     bool      `json:"isGroup"`
	Name        string    `json:"name"`
	Photo       string    `json:"photo"`
	Members     []userDTO `json:"members"`
	LastMessage *msgDTO   `json:"lastMessage,omitempty"`
}

type conversationDetailDTO struct {
	Conversation conversationDTO `json:"conversation"`
	Messages     []msgDTO        `json:"messages"`
}

type reactionDTO struct {
	UserID   int64  `json:"userId"`
	Username string `json:"username"`
	Emoji    string `json:"emoji"`
}

type msgDTO struct {
	ID             int64         `json:"id"`
	ConversationID int64         `json:"conversationId"`
	Sender         userDTO       `json:"sender"`
	Text           string        `json:"text"`
	Image          string        `json:"image"`
	ReplyToID      *int64        `json:"replyToId,omitempty"`
	Forwarded      bool          `json:"forwarded"`
	CreatedAt      string        `json:"createdAt"`
	ReadBy         []int64       `json:"readBy"`
	Reactions      []reactionDTO `json:"reactions"`
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func readJSON(r *http.Request, dst interface{}) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

func (rt *_router) authUser(r *http.Request) (userDTO, error) {
	h := r.Header.Get("Authorization")
	if !strings.HasPrefix(h, "Bearer ") {
		return userDTO{}, errors.New("missing bearer token")
	}
	token := strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))
	row := rt.db.Conn().QueryRow(`SELECT u.id, u.username, u.display_name, u.photo
		FROM sessions s JOIN users u ON u.id=s.user_id WHERE s.token=?`, token)
	var u userDTO
	if err := row.Scan(&u.ID, &u.Username, &u.DisplayName, &u.Photo); err != nil {
		return userDTO{}, errors.New("invalid token")
	}
	return u, nil
}

func (rt *_router) login(w http.ResponseWriter, r *http.Request, _ httprouter.Params, _ reqcontext.RequestContext) {
	var body struct {
		Username string `json:"username"`
	}
	if err := readJSON(r, &body); err != nil || strings.TrimSpace(body.Username) == "" {
		writeJSON(w, http.StatusBadRequest, apiError{Error: "username richiesto"})
		return
	}
	username := strings.TrimSpace(body.Username)
	if len(username) < 3 {
		writeJSON(w, http.StatusBadRequest, apiError{Error: "username troppo corto"})
		return
	}
	displayName := username
	_, _ = rt.db.Conn().Exec(`INSERT INTO users(username,display_name) VALUES(?,?) ON CONFLICT(username) DO NOTHING`, username, displayName)
	var uid int64
	var dn, photo string
	if err := rt.db.Conn().QueryRow(`SELECT id, display_name, photo FROM users WHERE username=?`, username).Scan(&uid, &dn, &photo); err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError{Error: "errore login"})
		return
	}
	tok, _ := uuid.NewV4()
	token := tok.String()
	if _, err := rt.db.Conn().Exec(`INSERT INTO sessions(token,user_id) VALUES(?,?)`, token, uid); err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError{Error: "errore sessione"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"token": token,
		"user":  userDTO{ID: uid, Username: username, DisplayName: dn, Photo: photo},
	})
}

func (rt *_router) logout(w http.ResponseWriter, r *http.Request, _ httprouter.Params, _ reqcontext.RequestContext) {
	h := r.Header.Get("Authorization")
	if !strings.HasPrefix(h, "Bearer ") {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	token := strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))
	_, _ = rt.db.Conn().Exec(`DELETE FROM sessions WHERE token=?`, token)
	w.WriteHeader(http.StatusNoContent)
}

func (rt *_router) getMe(w http.ResponseWriter, r *http.Request, _ httprouter.Params, _ reqcontext.RequestContext) {
	u, err := rt.authUser(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, apiError{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, u)
}

func (rt *_router) updateMe(w http.ResponseWriter, r *http.Request, _ httprouter.Params, _ reqcontext.RequestContext) {
	u, err := rt.authUser(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, apiError{Error: err.Error()})
		return
	}
	var body struct {
		DisplayName string `json:"displayName"`
		Photo       string `json:"photo"`
	}
	if err := readJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: "body non valida"})
		return
	}
	if strings.TrimSpace(body.DisplayName) != "" {
		var c int
		_ = rt.db.Conn().QueryRow(`SELECT COUNT(1) FROM users WHERE display_name=? AND id<>?`, strings.TrimSpace(body.DisplayName), u.ID).Scan(&c)
		if c > 0 {
			writeJSON(w, http.StatusConflict, apiError{Error: "nome profilo già in uso"})
			return
		}
	}
	if _, err := rt.db.Conn().Exec(`UPDATE users SET display_name=COALESCE(NULLIF(?,''),display_name), photo=COALESCE(?,photo) WHERE id=?`, strings.TrimSpace(body.DisplayName), body.Photo, u.ID); err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError{Error: "errore update"})
		return
	}
	rt.getMe(w, r, httprouter.Params{}, reqcontext.RequestContext{})
}

func (rt *_router) updateMyPhoto(w http.ResponseWriter, r *http.Request, _ httprouter.Params, _ reqcontext.RequestContext) {
	u, err := rt.authUser(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, apiError{Error: err.Error()})
		return
	}
	var body struct {
		Photo string `json:"photo"`
	}
	if err := readJSON(r, &body); err != nil || strings.TrimSpace(body.Photo) == "" {
		writeJSON(w, http.StatusBadRequest, apiError{Error: "photo richiesta"})
		return
	}
	if _, err := rt.db.Conn().Exec(`UPDATE users SET photo=? WHERE id=?`, strings.TrimSpace(body.Photo), u.ID); err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError{Error: "errore update foto"})
		return
	}
	rt.getMe(w, r, httprouter.Params{}, reqcontext.RequestContext{})
}

func (rt *_router) listUsers(w http.ResponseWriter, r *http.Request, _ httprouter.Params, _ reqcontext.RequestContext) {
	if _, err := rt.authUser(r); err != nil {
		writeJSON(w, http.StatusUnauthorized, apiError{Error: err.Error()})
		return
	}
	rows, err := rt.db.Conn().Query(`SELECT id, username, display_name, photo FROM users ORDER BY username`)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError{Error: "errore query utenti"})
		return
	}
	defer rows.Close()
	users := make([]userDTO, 0)
	for rows.Next() {
		var u userDTO
		if err := rows.Scan(&u.ID, &u.Username, &u.DisplayName, &u.Photo); err == nil {
			users = append(users, u)
		}
	}
	if err := rows.Err(); err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError{Error: "errore lettura utenti"})
		return
	}
	writeJSON(w, http.StatusOK, users)
}

func (rt *_router) createDirectConversation(w http.ResponseWriter, r *http.Request, _ httprouter.Params, _ reqcontext.RequestContext) {
	me, err := rt.authUser(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, apiError{Error: err.Error()})
		return
	}
	var body struct {
		Username string `json:"username"`
	}
	if err := readJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: "body non valida"})
		return
	}
	var peerID int64
	if err := rt.db.Conn().QueryRow(`SELECT id FROM users WHERE username=?`, strings.TrimSpace(body.Username)).Scan(&peerID); err != nil {
		writeJSON(w, http.StatusNotFound, apiError{Error: "utente non trovato"})
		return
	}
	if peerID == me.ID {
		writeJSON(w, http.StatusBadRequest, apiError{Error: "conversazione diretta con te stesso non consentita"})
		return
	}
	convID, err := rt.findOrCreateDirect(me.ID, peerID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError{Error: "errore creazione chat"})
		return
	}
	c, msgs, err := rt.conversationDetail(convID, me.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError{Error: "errore lettura chat"})
		return
	}
	writeJSON(w, http.StatusCreated, conversationDetailDTO{Conversation: c, Messages: msgs})
}

func (rt *_router) findOrCreateDirect(a, b int64) (int64, error) {
	rows, err := rt.db.Conn().Query(`SELECT c.id FROM conversations c
		JOIN conversation_members m1 ON m1.conversation_id=c.id AND m1.user_id=?
		JOIN conversation_members m2 ON m2.conversation_id=c.id AND m2.user_id=?
		WHERE c.is_group=0`, a, b)
	if err == nil {
		defer rows.Close()
		if rows.Next() {
			var id int64
			if err := rows.Scan(&id); err == nil {
				return id, nil
			}
		}
		if err := rows.Err(); err != nil {
			return 0, err
		}
	}
	res, err := rt.db.Conn().Exec(`INSERT INTO conversations(is_group,name,photo) VALUES(0,'','')`)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	_, err = rt.db.Conn().Exec(`INSERT INTO conversation_members(conversation_id,user_id) VALUES(?,?),(?,?)`, id, a, id, b)
	return id, err
}

func (rt *_router) createGroupConversation(w http.ResponseWriter, r *http.Request, _ httprouter.Params, _ reqcontext.RequestContext) {
	me, err := rt.authUser(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, apiError{Error: err.Error()})
		return
	}
	var body struct {
		Name    string   `json:"name"`
		Members []string `json:"members"`
	}
	if err := readJSON(r, &body); err != nil || strings.TrimSpace(body.Name) == "" {
		writeJSON(w, http.StatusBadRequest, apiError{Error: "nome gruppo richiesto"})
		return
	}
	res, err := rt.db.Conn().Exec(`INSERT INTO conversations(is_group,name,photo) VALUES(1,?,'')`, strings.TrimSpace(body.Name))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError{Error: "errore creazione gruppo"})
		return
	}
	id, _ := res.LastInsertId()
	_, _ = rt.db.Conn().Exec(`INSERT INTO conversation_members(conversation_id,user_id) VALUES(?,?)`, id, me.ID)
	for _, uname := range body.Members {
		var uid int64
		if err := rt.db.Conn().QueryRow(`SELECT id FROM users WHERE username=?`, strings.TrimSpace(uname)).Scan(&uid); err == nil && uid != me.ID {
			_, _ = rt.db.Conn().Exec(`INSERT OR IGNORE INTO conversation_members(conversation_id,user_id) VALUES(?,?)`, id, uid)
		}
	}
	c, msgs, err := rt.conversationDetail(id, me.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError{Error: "errore lettura chat"})
		return
	}
	writeJSON(w, http.StatusCreated, conversationDetailDTO{Conversation: c, Messages: msgs})
}

func (rt *_router) listConversations(w http.ResponseWriter, r *http.Request, _ httprouter.Params, _ reqcontext.RequestContext) {
	me, err := rt.authUser(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, apiError{Error: err.Error()})
		return
	}
	rows, err := rt.db.Conn().Query(`SELECT c.id,c.is_group,c.name,c.photo
		FROM conversations c
		JOIN conversation_members cm ON cm.conversation_id=c.id
		WHERE cm.user_id=? ORDER BY c.id DESC`, me.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError{Error: "errore lista chat"})
		return
	}
	defer rows.Close()
	out := make([]conversationDTO, 0)
	for rows.Next() {
		var c conversationDTO
		var isg int
		if err := rows.Scan(&c.ID, &isg, &c.Name, &c.Photo); err != nil {
			continue
		}
		c.IsGroup = isg == 1
		c.Members = rt.membersOf(c.ID)
		if !c.IsGroup && len(c.Members) == 2 {
			for _, m := range c.Members {
				if m.ID != me.ID {
					c.Name = m.DisplayName
					c.Photo = m.Photo
				}
			}
		}
		if m, err := rt.lastMessage(c.ID); err == nil {
			c.LastMessage = &m
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError{Error: "errore lettura chat"})
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (rt *_router) getConversation(w http.ResponseWriter, r *http.Request, ps httprouter.Params, _ reqcontext.RequestContext) {
	me, err := rt.authUser(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, apiError{Error: err.Error()})
		return
	}
	convID, _ := strconv.ParseInt(ps.ByName("id"), 10, 64)
	if !rt.conversationExists(convID) {
		writeJSON(w, http.StatusNotFound, apiError{Error: "conversazione non trovata"})
		return
	}
	if !rt.isMember(convID, me.ID) {
		writeJSON(w, http.StatusForbidden, apiError{Error: "non sei membro"})
		return
	}
	c, msgs, err := rt.conversationDetail(convID, me.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError{Error: "errore lettura chat"})
		return
	}
	writeJSON(w, http.StatusOK, conversationDetailDTO{Conversation: c, Messages: msgs})
}

func (rt *_router) membersOf(convID int64) []userDTO {
	rows, err := rt.db.Conn().Query(`SELECT u.id,u.username,u.display_name,u.photo
		FROM users u JOIN conversation_members cm ON cm.user_id=u.id
		WHERE cm.conversation_id=? ORDER BY u.username`, convID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := make([]userDTO, 0)
	for rows.Next() {
		var u userDTO
		if rows.Scan(&u.ID, &u.Username, &u.DisplayName, &u.Photo) == nil {
			out = append(out, u)
		}
	}
	if err := rows.Err(); err != nil {
		return nil
	}
	return out
}

func (rt *_router) lastMessage(convID int64) (msgDTO, error) {
	var m msgDTO
	row := rt.db.Conn().QueryRow(`SELECT m.id,m.conversation_id,m.text,m.image,m.reply_to_id,m.forwarded_from_id,m.created_at,
		u.id,u.username,u.display_name,u.photo
		FROM messages m JOIN users u ON u.id=m.sender_id WHERE m.conversation_id=? ORDER BY m.id DESC LIMIT 1`, convID)
	var reply, fwd sql.NullInt64
	if err := row.Scan(&m.ID, &m.ConversationID, &m.Text, &m.Image, &reply, &fwd, &m.CreatedAt, &m.Sender.ID, &m.Sender.Username, &m.Sender.DisplayName, &m.Sender.Photo); err != nil {
		return m, err
	}
	if reply.Valid {
		m.ReplyToID = &reply.Int64
	}
	m.Forwarded = fwd.Valid
	m.ReadBy = rt.readBy(m.ID)
	m.Reactions = rt.reactionsOf(m.ID)
	return m, nil
}

func (rt *_router) conversationDetail(convID, meID int64) (conversationDTO, []msgDTO, error) {
	var c conversationDTO
	var isg int
	if err := rt.db.Conn().QueryRow(`SELECT id,is_group,name,photo FROM conversations WHERE id=?`, convID).Scan(&c.ID, &isg, &c.Name, &c.Photo); err != nil {
		return c, nil, err
	}
	c.IsGroup = isg == 1
	c.Members = rt.membersOf(c.ID)
	if !c.IsGroup && len(c.Members) == 2 {
		for _, m := range c.Members {
			if m.ID != meID {
				c.Name = m.DisplayName
				c.Photo = m.Photo
			}
		}
	}
	rows, err := rt.db.Conn().Query(`SELECT m.id,m.conversation_id,m.text,m.image,m.reply_to_id,m.forwarded_from_id,m.created_at,
		u.id,u.username,u.display_name,u.photo
		FROM messages m JOIN users u ON u.id=m.sender_id WHERE m.conversation_id=? ORDER BY m.id ASC`, convID)
	if err != nil {
		return c, nil, err
	}
	defer rows.Close()
	msgs := make([]msgDTO, 0)
	for rows.Next() {
		var m msgDTO
		var reply, fwd sql.NullInt64
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.Text, &m.Image, &reply, &fwd, &m.CreatedAt, &m.Sender.ID, &m.Sender.Username, &m.Sender.DisplayName, &m.Sender.Photo); err == nil {
			if reply.Valid {
				m.ReplyToID = &reply.Int64
			}
			m.Forwarded = fwd.Valid
			m.ReadBy = rt.readBy(m.ID)
			m.Reactions = rt.reactionsOf(m.ID)
			msgs = append(msgs, m)
		}
	}
	if err := rows.Err(); err != nil {
		return c, nil, err
	}
	return c, msgs, nil
}

func (rt *_router) isMember(convID, uid int64) bool {
	var c int
	_ = rt.db.Conn().QueryRow(`SELECT COUNT(1) FROM conversation_members WHERE conversation_id=? AND user_id=?`, convID, uid).Scan(&c)
	return c > 0
}

func (rt *_router) conversationExists(convID int64) bool {
	var c int
	if err := rt.db.Conn().QueryRow(`SELECT COUNT(1) FROM conversations WHERE id=?`, convID).Scan(&c); err != nil {
		return false
	}
	return c > 0
}

func (rt *_router) sendMessage(w http.ResponseWriter, r *http.Request, ps httprouter.Params, _ reqcontext.RequestContext) {
	me, err := rt.authUser(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, apiError{Error: err.Error()})
		return
	}
	convID, _ := strconv.ParseInt(ps.ByName("id"), 10, 64)
	if !rt.conversationExists(convID) {
		writeJSON(w, http.StatusNotFound, apiError{Error: "conversazione non trovata"})
		return
	}
	if !rt.isMember(convID, me.ID) {
		writeJSON(w, http.StatusForbidden, apiError{Error: "non sei membro"})
		return
	}
	var body struct {
		Text            string `json:"text"`
		Image           string `json:"image"`
		ReplyToID       *int64 `json:"replyToId"`
		ForwardedFromID *int64 `json:"forwardedFromId"`
	}
	if err := readJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: "body non valida"})
		return
	}
	if strings.TrimSpace(body.Text) == "" && strings.TrimSpace(body.Image) == "" {
		writeJSON(w, http.StatusBadRequest, apiError{Error: "messaggio vuoto"})
		return
	}
	res, err := rt.db.Conn().Exec(`INSERT INTO messages(conversation_id,sender_id,text,image,reply_to_id,forwarded_from_id)
		VALUES(?,?,?,?,?,?)`, convID, me.ID, strings.TrimSpace(body.Text), body.Image, body.ReplyToID, body.ForwardedFromID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError{Error: "errore invio"})
		return
	}
	id, _ := res.LastInsertId()
	_, _ = rt.db.Conn().Exec(`INSERT OR IGNORE INTO message_reads(message_id,user_id) VALUES(?,?)`, id, me.ID)
	writeJSON(w, http.StatusCreated, map[string]int64{"messageId": id})
}

func (rt *_router) markRead(w http.ResponseWriter, r *http.Request, ps httprouter.Params, _ reqcontext.RequestContext) {
	me, err := rt.authUser(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, apiError{Error: err.Error()})
		return
	}
	msgID, _ := strconv.ParseInt(ps.ByName("id"), 10, 64)
	var convID int64
	if err := rt.db.Conn().QueryRow(`SELECT conversation_id FROM messages WHERE id=?`, msgID).Scan(&convID); err != nil {
		writeJSON(w, http.StatusNotFound, apiError{Error: "messaggio non trovato"})
		return
	}
	if !rt.isMember(convID, me.ID) {
		writeJSON(w, http.StatusForbidden, apiError{Error: "non autorizzato"})
		return
	}
	_, _ = rt.db.Conn().Exec(`INSERT OR IGNORE INTO message_reads(message_id,user_id) VALUES(?,?)`, msgID, me.ID)
	w.WriteHeader(http.StatusNoContent)
}

func (rt *_router) readBy(msgID int64) []int64 {
	rows, err := rt.db.Conn().Query(`SELECT user_id FROM message_reads WHERE message_id=?`, msgID)
	if err != nil {
		return []int64{}
	}
	defer rows.Close()
	out := make([]int64, 0)
	for rows.Next() {
		var uid int64
		if rows.Scan(&uid) == nil {
			out = append(out, uid)
		}
	}
	if err := rows.Err(); err != nil {
		return []int64{}
	}
	return out
}

func (rt *_router) setReaction(w http.ResponseWriter, r *http.Request, ps httprouter.Params, _ reqcontext.RequestContext) {
	me, err := rt.authUser(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, apiError{Error: err.Error()})
		return
	}
	var body struct {
		Emoji string `json:"emoji"`
	}
	if err := readJSON(r, &body); err != nil || strings.TrimSpace(body.Emoji) == "" {
		writeJSON(w, http.StatusBadRequest, apiError{Error: "emoji richiesta"})
		return
	}
	msgID, _ := strconv.ParseInt(ps.ByName("id"), 10, 64)
	var convID int64
	if err := rt.db.Conn().QueryRow(`SELECT conversation_id FROM messages WHERE id=?`, msgID).Scan(&convID); err != nil || !rt.isMember(convID, me.ID) {
		writeJSON(w, http.StatusForbidden, apiError{Error: "messaggio non accessibile"})
		return
	}
	_, err = rt.db.Conn().Exec(`INSERT INTO reactions(message_id,user_id,emoji) VALUES(?,?,?)
		ON CONFLICT(message_id,user_id) DO UPDATE SET emoji=excluded.emoji`, msgID, me.ID, body.Emoji)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError{Error: "errore reazione"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (rt *_router) removeReaction(w http.ResponseWriter, r *http.Request, ps httprouter.Params, _ reqcontext.RequestContext) {
	me, err := rt.authUser(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, apiError{Error: err.Error()})
		return
	}
	msgID, _ := strconv.ParseInt(ps.ByName("id"), 10, 64)
	_, _ = rt.db.Conn().Exec(`DELETE FROM reactions WHERE message_id=? AND user_id=?`, msgID, me.ID)
	w.WriteHeader(http.StatusNoContent)
}

func (rt *_router) reactionsOf(msgID int64) []reactionDTO {
	rows, err := rt.db.Conn().Query(`SELECT r.user_id,u.username,r.emoji FROM reactions r JOIN users u ON u.id=r.user_id WHERE r.message_id=?`, msgID)
	if err != nil {
		return []reactionDTO{}
	}
	defer rows.Close()
	out := make([]reactionDTO, 0)
	for rows.Next() {
		var rr reactionDTO
		if rows.Scan(&rr.UserID, &rr.Username, &rr.Emoji) == nil {
			out = append(out, rr)
		}
	}
	if err := rows.Err(); err != nil {
		return []reactionDTO{}
	}
	return out
}

func (rt *_router) forwardMessage(w http.ResponseWriter, r *http.Request, ps httprouter.Params, _ reqcontext.RequestContext) {
	me, err := rt.authUser(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, apiError{Error: err.Error()})
		return
	}
	msgID, _ := strconv.ParseInt(ps.ByName("id"), 10, 64)
	var body struct {
		ToConversationID int64 `json:"toConversationId"`
	}
	if err := readJSON(r, &body); err != nil || body.ToConversationID <= 0 {
		writeJSON(w, http.StatusBadRequest, apiError{Error: "toConversationId richiesto"})
		return
	}
	if !rt.isMember(body.ToConversationID, me.ID) {
		writeJSON(w, http.StatusForbidden, apiError{Error: "non membro chat destinazione"})
		return
	}
	var text, image string
	if err := rt.db.Conn().QueryRow(`SELECT text,image FROM messages WHERE id=?`, msgID).Scan(&text, &image); err != nil {
		writeJSON(w, http.StatusNotFound, apiError{Error: "messaggio da inoltrare non trovato"})
		return
	}
	res, err := rt.db.Conn().Exec(`INSERT INTO messages(conversation_id,sender_id,text,image,forwarded_from_id) VALUES(?,?,?,?,?)`, body.ToConversationID, me.ID, text, image, msgID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError{Error: "errore inoltro"})
		return
	}
	id, _ := res.LastInsertId()
	_, _ = rt.db.Conn().Exec(`INSERT OR IGNORE INTO message_reads(message_id,user_id) VALUES(?,?)`, id, me.ID)
	writeJSON(w, http.StatusCreated, map[string]int64{"messageId": id})
}

func (rt *_router) deleteMessage(w http.ResponseWriter, r *http.Request, ps httprouter.Params, _ reqcontext.RequestContext) {
	me, err := rt.authUser(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, apiError{Error: err.Error()})
		return
	}
	msgID, _ := strconv.ParseInt(ps.ByName("id"), 10, 64)
	var senderID int64
	if err := rt.db.Conn().QueryRow(`SELECT sender_id FROM messages WHERE id=?`, msgID).Scan(&senderID); err != nil {
		writeJSON(w, http.StatusNotFound, apiError{Error: "messaggio non trovato"})
		return
	}
	if senderID != me.ID {
		writeJSON(w, http.StatusForbidden, apiError{Error: "non autorizzato a eliminare"})
		return
	}
	_, _ = rt.db.Conn().Exec(`DELETE FROM reactions WHERE message_id=?`, msgID)
	_, _ = rt.db.Conn().Exec(`DELETE FROM message_reads WHERE message_id=?`, msgID)
	if _, err := rt.db.Conn().Exec(`DELETE FROM messages WHERE id=?`, msgID); err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError{Error: "errore eliminazione"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (rt *_router) addToGroup(w http.ResponseWriter, r *http.Request, ps httprouter.Params, _ reqcontext.RequestContext) {
	me, err := rt.authUser(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, apiError{Error: err.Error()})
		return
	}
	convID, _ := strconv.ParseInt(ps.ByName("id"), 10, 64)
	if !rt.conversationExists(convID) {
		writeJSON(w, http.StatusNotFound, apiError{Error: "conversazione non trovata"})
		return
	}
	if !rt.isMember(convID, me.ID) {
		writeJSON(w, http.StatusForbidden, apiError{Error: "non sei membro"})
		return
	}
	var isg int
	if err := rt.db.Conn().QueryRow(`SELECT is_group FROM conversations WHERE id=?`, convID).Scan(&isg); err != nil || isg != 1 {
		writeJSON(w, http.StatusBadRequest, apiError{Error: "non è un gruppo"})
		return
	}
	var body struct {
		Username string `json:"username"`
	}
	if err := readJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: "body non valida"})
		return
	}
	var uid int64
	if err := rt.db.Conn().QueryRow(`SELECT id FROM users WHERE username=?`, strings.TrimSpace(body.Username)).Scan(&uid); err != nil {
		writeJSON(w, http.StatusNotFound, apiError{Error: "utente non trovato"})
		return
	}
	_, _ = rt.db.Conn().Exec(`INSERT OR IGNORE INTO conversation_members(conversation_id,user_id) VALUES(?,?)`, convID, uid)
	w.WriteHeader(http.StatusNoContent)
}

func (rt *_router) leaveGroup(w http.ResponseWriter, r *http.Request, ps httprouter.Params, _ reqcontext.RequestContext) {
	me, err := rt.authUser(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, apiError{Error: err.Error()})
		return
	}
	convID, _ := strconv.ParseInt(ps.ByName("id"), 10, 64)
	if !rt.conversationExists(convID) {
		writeJSON(w, http.StatusNotFound, apiError{Error: "conversazione non trovata"})
		return
	}
	_, _ = rt.db.Conn().Exec(`DELETE FROM conversation_members WHERE conversation_id=? AND user_id=?`, convID, me.ID)
	w.WriteHeader(http.StatusNoContent)
}

func (rt *_router) updateConversation(w http.ResponseWriter, r *http.Request, ps httprouter.Params, _ reqcontext.RequestContext) {
	me, err := rt.authUser(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, apiError{Error: err.Error()})
		return
	}
	convID, _ := strconv.ParseInt(ps.ByName("id"), 10, 64)
	if !rt.conversationExists(convID) {
		writeJSON(w, http.StatusNotFound, apiError{Error: "conversazione non trovata"})
		return
	}
	if !rt.isMember(convID, me.ID) {
		writeJSON(w, http.StatusForbidden, apiError{Error: "non sei membro"})
		return
	}
	var isg int
	if err := rt.db.Conn().QueryRow(`SELECT is_group FROM conversations WHERE id=?`, convID).Scan(&isg); err != nil || isg != 1 {
		writeJSON(w, http.StatusBadRequest, apiError{Error: "non è un gruppo"})
		return
	}
	var body struct {
		Name  string `json:"name"`
		Photo string `json:"photo"`
	}
	if err := readJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: "body non valida"})
		return
	}
	_, err = rt.db.Conn().Exec(`UPDATE conversations SET name=COALESCE(NULLIF(?,''),name), photo=COALESCE(?,photo) WHERE id=?`, strings.TrimSpace(body.Name), body.Photo, convID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError{Error: fmt.Sprintf("errore update gruppo: %v", err)})
		return
	}
	c, msgs, err := rt.conversationDetail(convID, me.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError{Error: "errore lettura chat"})
		return
	}
	writeJSON(w, http.StatusOK, conversationDetailDTO{Conversation: c, Messages: msgs})
}

func (rt *_router) updateGroupPhoto(w http.ResponseWriter, r *http.Request, ps httprouter.Params, _ reqcontext.RequestContext) {
	me, err := rt.authUser(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, apiError{Error: err.Error()})
		return
	}
	convID, _ := strconv.ParseInt(ps.ByName("id"), 10, 64)
	if !rt.conversationExists(convID) {
		writeJSON(w, http.StatusNotFound, apiError{Error: "conversazione non trovata"})
		return
	}
	if !rt.isMember(convID, me.ID) {
		writeJSON(w, http.StatusForbidden, apiError{Error: "non sei membro"})
		return
	}
	var isg int
	if err := rt.db.Conn().QueryRow(`SELECT is_group FROM conversations WHERE id=?`, convID).Scan(&isg); err != nil || isg != 1 {
		writeJSON(w, http.StatusBadRequest, apiError{Error: "non è un gruppo"})
		return
	}
	var body struct {
		Photo string `json:"photo"`
	}
	if err := readJSON(r, &body); err != nil || strings.TrimSpace(body.Photo) == "" {
		writeJSON(w, http.StatusBadRequest, apiError{Error: "photo richiesta"})
		return
	}
	if _, err := rt.db.Conn().Exec(`UPDATE conversations SET photo=? WHERE id=?`, strings.TrimSpace(body.Photo), convID); err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError{Error: "errore update foto gruppo"})
		return
	}
	c, msgs, err := rt.conversationDetail(convID, me.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError{Error: "errore lettura chat"})
		return
	}
	writeJSON(w, http.StatusOK, conversationDetailDTO{Conversation: c, Messages: msgs})
}
