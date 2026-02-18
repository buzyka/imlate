package entity

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

func TestUserRoleConstants(t *testing.T) {
	assert.Equal(t, UserRole("admin"), UserRoleAdmin)
	assert.Equal(t, UserRole("terminal"), UserRoleTerminal)
}

func TestSetPassword_Success(t *testing.T) {
	user := &User{}
	err := user.SetPassword("mysecret")

	assert.NoError(t, err)
	assert.NotEmpty(t, user.Password)
	assert.NotEqual(t, "mysecret", user.Password)

	// Verify it's a valid bcrypt hash
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte("mysecret"))
	assert.NoError(t, err)
}

func TestSetPassword_EmptyString(t *testing.T) {
	user := &User{}
	err := user.SetPassword("")

	assert.NoError(t, err)
	assert.NotEmpty(t, user.Password)

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(""))
	assert.NoError(t, err)
}

func TestSetPassword_TooLong(t *testing.T) {
	user := &User{}
	// bcrypt returns an error for passwords > 72 bytes
	longPassword := make([]byte, 73)
	for i := range longPassword {
		longPassword[i] = 'a'
	}
	err := user.SetPassword(string(longPassword))

	assert.Error(t, err)
	assert.Empty(t, user.Password)
}

func TestSetPassword_OverwritesPrevious(t *testing.T) {
	user := &User{}
	_ = user.SetPassword("first")
	firstHash := user.Password

	_ = user.SetPassword("second")
	assert.NotEqual(t, firstHash, user.Password)

	// Old password should no longer validate
	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte("first"))
	assert.Error(t, err)

	// New password should validate
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte("second"))
	assert.NoError(t, err)
}

func TestPasswordValidate_Correct(t *testing.T) {
	user := &User{}
	_ = user.SetPassword("correctpassword")

	assert.True(t, user.PasswordValidate("correctpassword"))
}

func TestPasswordValidate_Wrong(t *testing.T) {
	user := &User{}
	_ = user.SetPassword("correctpassword")

	assert.False(t, user.PasswordValidate("wrongpassword"))
}

func TestPasswordValidate_EmptyHash(t *testing.T) {
	user := &User{Password: ""}

	assert.False(t, user.PasswordValidate("anything"))
}

func TestPasswordValidate_EmptyPassword(t *testing.T) {
	user := &User{}
	_ = user.SetPassword("somepassword")

	assert.False(t, user.PasswordValidate(""))
}

func TestPasswordValidate_BothEmpty(t *testing.T) {
	user := &User{}
	_ = user.SetPassword("")

	assert.True(t, user.PasswordValidate(""))
}

func TestIsDeleted_Nil(t *testing.T) {
	user := &User{DeletedAt: nil}

	assert.False(t, user.IsDeleted())
}

func TestIsDeleted_Set(t *testing.T) {
	now := time.Now()
	user := &User{DeletedAt: &now}

	assert.True(t, user.IsDeleted())
}
