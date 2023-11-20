package yubikey

import (
	"errors"
	"sync"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrUnknownIDType     = errors.New("unknown user ID type must be uuid")
)

func NewUsers() *Users {
	return &Users{
		users:  make(map[uuid.UUID]*User),
		emails: make(map[string]uuid.UUID),
	}
}

type Users struct {
	sync.RWMutex
	users  map[uuid.UUID]*User
	emails map[string]uuid.UUID
}

func (db *Users) Lookup(id interface{}) (_ *User, err error) {
	var userID uuid.UUID
	switch idtyp := id.(type) {
	case string:
		if userID, err = uuid.Parse(idtyp); err != nil {
			return nil, err
		}
	case []byte:
		if userID, err = uuid.FromBytes(idtyp); err != nil {
			return nil, err
		}
	case uuid.UUID:
		userID = idtyp
	default:
		return nil, ErrUnknownIDType
	}

	if user, ok := db.users[userID]; ok {
		return user, nil
	}
	return nil, ErrUserNotFound
}

func (db *Users) GetUser(email string) (*User, error) {
	db.RLock()
	defer db.RUnlock()
	id := db.emails[email]
	if user, ok := db.users[id]; ok {
		return user, nil
	}
	return nil, ErrUserNotFound
}

func (db *Users) NewUser(name, email string) (*User, error) {
	user := &User{
		ID:          uuid.New(),
		Name:        name,
		Email:       email,
		credentials: make([]webauthn.Credential, 0, 1),
	}

	db.Lock()
	defer db.Unlock()
	if _, ok := db.emails[email]; ok {
		return nil, ErrUserAlreadyExists
	}

	db.emails[email] = user.ID
	db.users[user.ID] = user
	return user, nil
}

type User struct {
	sync.RWMutex
	ID          uuid.UUID
	Name        string
	Email       string
	credentials []webauthn.Credential
}

// WebAuthnID provides the user handle of the user account. A user handle is an opaque byte sequence with a maximum
// size of 64 bytes, and is not meant to be displayed to the user.
//
// To ensure secure operation, authentication and authorization decisions MUST be made on the basis of this id
// member, not the displayName nor name members. See Section 6.1 of [RFC8266].
//
// It's recommended this value is completely random and uses the entire 64 bytes.
//
// Specification: §5.4.3. User Account Parameters for Credential Generation (https://w3c.github.io/webauthn/#dom-publickeycredentialuserentity-id)
func (u *User) WebAuthnID() []byte {
	u.RLock()
	defer u.RUnlock()
	return u.ID[:]
}

// WebAuthnName provides the name attribute of the user account during registration and is a human-palatable name for the user
// account, intended only for display. For example, "Alex Müller" or "田中倫". The Relying Party SHOULD let the user
// choose this, and SHOULD NOT restrict the choice more than necessary.
//
// Specification: §5.4.3. User Account Parameters for Credential Generation (https://w3c.github.io/webauthn/#dictdef-publickeycredentialuserentity)
func (u *User) WebAuthnName() string {
	u.RLock()
	defer u.RUnlock()
	return u.Email
}

// WebAuthnDisplayName provides the name attribute of the user account during registration and is a human-palatable
// name for the user account, intended only for display. For example, "Alex Müller" or "田中倫". The Relying Party
// SHOULD let the user choose this, and SHOULD NOT restrict the choice more than necessary.
//
// Specification: §5.4.3. User Account Parameters for Credential Generation (https://www.w3.org/TR/webauthn/#dom-publickeycredentialuserentity-displayname)
func (u *User) WebAuthnDisplayName() string {
	u.RLock()
	defer u.RUnlock()
	return u.Name
}

// WebAuthnCredentials provides the list of Credential objects owned by the user.
func (u *User) WebAuthnCredentials() []webauthn.Credential {
	u.RLock()
	defer u.RUnlock()
	return u.credentials
}

func (u *User) CredentialExcludeList() []protocol.CredentialDescriptor {
	u.RLock()
	defer u.RUnlock()
	exclude := make([]protocol.CredentialDescriptor, 0, len(u.credentials))
	for _, cred := range u.credentials {
		descriptor := protocol.CredentialDescriptor{
			Type:         protocol.PublicKeyCredentialType,
			CredentialID: cred.ID,
		}
		exclude = append(exclude, descriptor)
	}
	return exclude
}

func (u *User) AddCredential(cred webauthn.Credential) {
	u.Lock()
	defer u.Unlock()
	u.credentials = append(u.credentials, cred)
}

// WebAuthnIcon is a deprecated option.
// Deprecated: this has been removed from the specification recommendation. Suggest a blank string.
func (u *User) WebAuthnIcon() string { return "" }
