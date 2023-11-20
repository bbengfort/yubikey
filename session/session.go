package session

import (
	"crypto/rand"
	"encoding/json"
	"net/http"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/gorilla/sessions"
)

const (
	DefaultEncryptionKeyLength = 32
	WebauthnSession            = "webauthn-session"
)

// Store is a wrapper around sessions.CookieStore which provides some helper methods
// related to webauthn operations and encrypted cookies.
type Store struct {
	*sessions.CookieStore
}

func New(keyPairs ...[]byte) (*Store, error) {
	if len(keyPairs) == 0 {
		key, err := GenerateSecureKey(DefaultEncryptionKeyLength)
		if err != nil {
			return nil, err
		}
		keyPairs = append(keyPairs, key)
	}

	store := &Store{
		sessions.NewCookieStore(keyPairs...),
	}
	return store, nil
}

func (store *Store) SaveWebauthnSession(key string, data *webauthn.SessionData, r *http.Request, w http.ResponseWriter) error {
	marshaledData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return store.Set(key, marshaledData, r, w)
}

func (store *Store) GetWebauthnSession(key string, r *http.Request) (webauthn.SessionData, error) {
	sessionData := webauthn.SessionData{}
	session, err := store.Get(r, WebauthnSession)
	if err != nil {
		return sessionData, err
	}
	assertion, ok := session.Values[key].([]byte)
	if !ok {
		return sessionData, ErrMarshal
	}
	err = json.Unmarshal(assertion, &sessionData)
	if err != nil {
		return sessionData, err
	}
	// Delete the value from the session now that it's been read
	delete(session.Values, key)
	return sessionData, nil
}

func (store *Store) Set(key string, value interface{}, r *http.Request, w http.ResponseWriter) error {
	session, err := store.Get(r, WebauthnSession)
	if err != nil {
		return err
	}

	session.Values[key] = value
	session.Save(r, w)
	return nil
}

func GenerateSecureKey(n int) ([]byte, error) {
	buf := make([]byte, n)
	read, err := rand.Read(buf)
	if err != nil {
		return buf, err
	}

	if read != n {
		return buf, ErrInsufficientBytesRead
	}
	return buf, nil
}
