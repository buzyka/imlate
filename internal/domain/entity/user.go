package entity

import (
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// UserRole represents the role of a user in the system.
type UserRole string

const (
	UserRoleAdmin    UserRole = "admin"
	UserRoleTerminal UserRole = "terminal"
)

type User struct {
	ID        uuid.UUID  `json:"id"`
	UserName  string     `json:"username"`
	Password  string     `json:"-"`
	Name      string     `json:"name"`
	Surname   string     `json:"surname"`
	Role      UserRole   `json:"role"`
	IsActive  bool       `json:"is_active"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at"`
}

// SetPassword hashes the given plaintext password with bcrypt and stores the hash.
func (u *User) SetPassword(pwd string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hash)
	return nil
}

// PasswordValidate compares the given plaintext password against the stored bcrypt hash.
func (u *User) PasswordValidate(pwd string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(pwd))
	return err == nil
}

// IsDeleted returns true if the user has been soft-deleted.
func (u *User) IsDeleted() bool {
	return u.DeletedAt != nil
}
