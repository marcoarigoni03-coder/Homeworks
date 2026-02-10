# Guida per l'orale – panoramica completa del progetto

Questa guida riassume **tutte le API HTTP implementate**, i relativi metodi/codici di stato, e come sono usate dal frontend.

## 1) Architettura generale

- Backend in Go (`cmd/webapi` + `service/api` + `service/database`).
- Frontend in Vue (`webui/src`).
- Database SQLite con schema creato all'avvio (`service/database/database.go`).

## 2) Avvio e wiring backend

- `cmd/webapi/main.go`:
  - carica config;
  - inizializza logger;
  - apre SQLite e crea schema;
  - crea router API (`api.New(...)`);
  - opzionalmente registra la WebUI embedded (`registerWebUI`);
  - applica CORS (`applyCORSHandler`);
  - avvia HTTP server.
- `service/api/api-handler.go` registra tutte le route con metodo HTTP.
- `service/api/api-context-wrapper.go` aggiunge un request context con UUID e logger per ogni richiesta.

## 3) Modello dati (SQLite)

Tabelle principali:
- `users`
- `sessions`
- `conversations`
- `conversation_members`
- `messages`
- `message_reads`
- `reactions`

## 4) API endpoint: metodi, funzione backend, status code, uso frontend

## 4.1 Health

### `GET /liveness`
- **Backend**: `liveness` in `service/api/liveness.go`.
- **Logica**: fa `db.Ping()`.
- **Status code**:
  - `200 OK` se DB raggiungibile.
  - `500 Internal Server Error` se ping fallisce.
- **Frontend**: non chiamata direttamente dalla pagina principale; usata dal tool `cmd/healthcheck`.

## 4.2 Auth

### `POST /api/login`
- **Backend**: `login` in `service/api/chat.go`.
- **Body**: `{ "username": string }`.
- **Logica**:
  - valida body e lunghezza username;
  - crea utente se non esiste (`INSERT ... ON CONFLICT DO NOTHING`);
  - crea sessione e token UUID;
  - risponde con token + utente.
- **Status code**:
  - `200 OK` login riuscito.
  - `400 Bad Request` username mancante o troppo corto.
  - `500 Internal Server Error` errori DB/sessione.
- **Frontend**: metodo `login()` in `HomeView.vue`.

### `POST /api/logout`
- **Backend**: `logout` in `service/api/chat.go`.
- **Logica**:
  - se `Authorization` non è bearer valido, risponde comunque successo;
  - se presente token, elimina sessione.
- **Status code**:
  - `204 No Content` sempre.
- **Frontend**: metodo `logout()` in `HomeView.vue`.

## 4.3 Profilo

### `GET /api/me`
- **Backend**: `getMe`.
- **Auth**: richiesta bearer token (`authUser`).
- **Status code**:
  - `200 OK` con user profile.
  - `401 Unauthorized` token mancante/non valido.
- **Frontend**: usata da `refreshAll()`.

### `PUT /api/me`
- **Backend**: `updateMe`.
- **Body**: `{ displayName, photo }`.
- **Logica**:
  - valida JSON;
  - se displayName non vuoto, controlla unicità;
  - aggiorna profilo e restituisce nuovo profilo.
- **Status code**:
  - `200 OK` update riuscito.
  - `400 Bad Request` body non valido.
  - `401 Unauthorized` non autenticato.
  - `409 Conflict` displayName già in uso.
  - `500 Internal Server Error` errore update.
- **Frontend**: `updateProfile()`.

### `PUT /api/me/photo`
- **Backend**: `updateMyPhoto`.
- **Body**: `{ photo }`.
- **Status code**:
  - `200 OK` foto aggiornata (ritorna profilo).
  - `400 Bad Request` photo assente/vuota.
  - `401 Unauthorized` token non valido.
  - `500 Internal Server Error` errore DB.
- **Frontend**: non invocata direttamente nella vista attuale (si usa `PUT /api/me`).

## 4.4 Utenti

### `GET /api/users`
- **Backend**: `listUsers`.
- **Status code**:
  - `200 OK` lista utenti.
  - `401 Unauthorized` non autenticato.
  - `500 Internal Server Error` errore query/scan.
- **Frontend**: `refreshAll()`.

## 4.5 Conversazioni

### `GET /api/conversations`
- **Backend**: `listConversations`.
- **Logica**:
  - prende conversazioni dell'utente;
  - arricchisce con membri e ultimo messaggio;
  - per direct chat sostituisce nome/foto con quelli del peer.
- **Status code**:
  - `200 OK` lista conversazioni.
  - `401 Unauthorized` non autenticato.
  - `500 Internal Server Error` errore DB.
- **Frontend**: `refreshAll()`.

### `POST /api/conversations/direct`
- **Backend**: `createDirectConversation` + helper `findOrCreateDirect`.
- **Body**: `{ username }`.
- **Status code**:
  - `201 Created` conversazione trovata/creata.
  - `400 Bad Request` body non valido o self-chat.
  - `401 Unauthorized` non autenticato.
  - `404 Not Found` utente destinatario non trovato.
  - `500 Internal Server Error` errore creazione/lettura.
- **Frontend**: `createDirect(username)`.

### `POST /api/conversations/group`
- **Backend**: `createGroupConversation`.
- **Body**: `{ name, members[] }`.
- **Status code**:
  - `201 Created` gruppo creato.
  - `400 Bad Request` nome mancante/body invalido.
  - `401 Unauthorized` non autenticato.
  - `500 Internal Server Error` errore DB.
- **Frontend**: `createGroup()`.

### `GET /api/conversation/:id`
- **Backend**: `getConversation`.
- **Status code**:
  - `200 OK` dettaglio conversazione + messaggi.
  - `401 Unauthorized` non autenticato.
  - `403 Forbidden` utente non membro.
  - `404 Not Found` conversazione inesistente.
  - `500 Internal Server Error` errore lettura.
- **Frontend**: `openConversation(id)`.

### `PUT /api/conversation/:id`
- **Backend**: `updateConversation`.
- **Body**: `{ name, photo }`.
- **Note**: valida che conversazione esista, utente sia membro, e che sia gruppo (`is_group=1`).
- **Status code**:
  - `200 OK` gruppo aggiornato.
  - `400 Bad Request` body invalido o non gruppo.
  - `401 Unauthorized` non autenticato.
  - `403 Forbidden` non membro.
  - `404 Not Found` conversazione non trovata.
  - `500 Internal Server Error` errore update/lettura.
- **Frontend**: `updateGroup()`.

### `PUT /api/conversation/:id/photo`
- **Backend**: route registrata in `api-handler.go`, ma nel codice fornito non c'è implementazione del metodo `updateGroupPhoto`.
- **Impatto**: endpoint dichiarato nelle route ma non usato dal frontend corrente.

### `POST /api/conversation/:id/add`
- **Backend**: `addToGroup`.
- **Body**: `{ username }`.
- **Status code**:
  - `204 No Content` utente aggiunto (o già presente).
  - `400 Bad Request` body invalido o conversazione non gruppo.
  - `401 Unauthorized` non autenticato.
  - `403 Forbidden` non membro del gruppo.
  - `404 Not Found` conversazione/utente non trovati.
- **Frontend**: `addMember(username)`.

### `POST /api/conversation/:id/leave`
- **Backend**: `leaveGroup`.
- **Status code**:
  - `204 No Content` uscita effettuata.
  - `401 Unauthorized` non autenticato.
  - `404 Not Found` conversazione non trovata.
- **Frontend**: `leaveGroup()`.

## 4.6 Messaggi

### `POST /api/conversation/:id/messages`
- **Backend**: `sendMessage`.
- **Body**: `{ text, image, replyToId?, forwardedFromId? }`.
- **Status code**:
  - `201 Created` messaggio creato.
  - `400 Bad Request` body invalido o messaggio vuoto.
  - `401 Unauthorized` non autenticato.
  - `403 Forbidden` non membro conversazione.
  - `404 Not Found` conversazione inesistente.
  - `500 Internal Server Error` errore insert.
- **Frontend**: `send()`.

### `POST /api/messages/:id/read`
- **Backend**: `markRead`.
- **Status code**:
  - `204 No Content` segnato come letto.
  - `401 Unauthorized` non autenticato.
  - `403 Forbidden` messaggio non accessibile (non membro).
  - `404 Not Found` messaggio non trovato.
- **Frontend**: chiamata da `openConversation` per ogni messaggio ricevuto.

### `POST /api/messages/:id/forward`
- **Backend**: `forwardMessage`.
- **Body**: `{ toConversationId }`.
- **Status code**:
  - `201 Created` inoltro riuscito.
  - `400 Bad Request` body invalido / id mancante.
  - `401 Unauthorized` non autenticato.
  - `403 Forbidden` non membro chat destinazione.
  - `404 Not Found` messaggio origine non trovato.
  - `500 Internal Server Error` errore insert.
- **Frontend**: `forward(msg, toConversationId)`.

### `DELETE /api/messages/:id`
- **Backend**: `deleteMessage`.
- **Logica**: solo il sender può eliminare.
- **Status code**:
  - `204 No Content` eliminato.
  - `401 Unauthorized` non autenticato.
  - `403 Forbidden` utente non autore.
  - `404 Not Found` messaggio non trovato.
  - `500 Internal Server Error` errore delete.
- **Frontend**: endpoint disponibile backend, non esposto in una funzione dedicata nella vista corrente.

## 4.7 Reazioni

### `POST /api/messages/:id/reaction`
- **Backend**: `setReaction`.
- **Body**: `{ emoji }`.
- **Status code**:
  - `204 No Content` reazione impostata/aggiornata.
  - `400 Bad Request` emoji mancante.
  - `401 Unauthorized` non autenticato.
  - `403 Forbidden` messaggio non accessibile.
  - `500 Internal Server Error` errore DB.
- **Frontend**: `react(msg)`.

### `DELETE /api/messages/:id/reaction`
- **Backend**: `removeReaction`.
- **Status code**:
  - `204 No Content` reazione rimossa.
  - `401 Unauthorized` non autenticato.
- **Frontend**: `unreact(msg)`.

## 5) Funzioni helper backend importanti (da sapere all'orale)

- `authUser`: valida Bearer token e risolve utente dalla tabella `sessions`.
- `writeJSON` / `readJSON`: serializzazione e validazione JSON (`DisallowUnknownFields`).
- `conversationDetail`: costruisce DTO completo conversazione + messaggi + readBy + reactions.
- `membersOf`, `lastMessage`, `readBy`, `reactionsOf`: arricchimento dati.
- `conversationExists`, `isMember`: guardie autorizzative.

## 6) Frontend: organizzazione e flusso

- `webui/src/services/axios.js`: crea client axios con baseURL risolto dinamicamente da `__API_URL__`.
- `webui/src/main.js`: registra router, axios globale e componenti comuni.
- `webui/src/router/index.js`: tutte le route puntano a `HomeView`.
- `webui/src/views/HomeView.vue`: vista principale con tutte le azioni utente:
  - autenticazione,
  - fetch dati periodico (polling ogni 500ms),
  - apertura chat,
  - invio/reazione/inoltro messaggi,
  - gestione gruppo,
  - update profilo.

## 7) CORS e sicurezza richiesta

- CORS consente header `Content-Type` e `Authorization` e metodi `GET/POST/OPTIONS/DELETE/PUT`.
- Le API protette usano token Bearer in header `Authorization`.

## 8) Nota su specifica OpenAPI

- È presente `doc/api.yaml` con la documentazione API.
- Utile per ripasso rapido delle request/response, ma **la fonte di verità runtime** resta l'implementazione Go in `service/api/chat.go` e nel router `service/api/api-handler.go`.
