package auth

import (
	"context"
	"testing"
	"time"
)

type fakeUserRepository struct {
	user User
	updatedHash string
}

func (f fakeUserRepository) FindByUsername(_ context.Context, username string) (User, error) {
	if username != f.user.Username {
		return User{}, ErrInvalidCredentials
	}
	return f.user, nil
}

func (f *fakeUserRepository) UpdatePassword(_ context.Context, userID, passwordHash string) error {
	f.updatedHash = passwordHash
	return nil
}

func TestServiceLoginReturnsVerifiableToken(t *testing.T) {
	hash := HashPassword("admin123", "fixed-salt")
	service := NewService(&fakeUserRepository{user: User{
		ID: "user-1", Username: "admin", DisplayName: "Administrator", PasswordHash: hash, Status: "active", Roles: []string{"admin"},
	}}, Config{Secret: "test-secret", ExpiresHours: 1})

	result, err := service.Login(context.Background(), "admin", "admin123")
	if err != nil {
		t.Fatalf("Login returned error: %v", err)
	}
	if result.Token == "" {
		t.Fatal("expected token")
	}
	if result.User.Username != "admin" {
		t.Fatalf("expected admin user, got %q", result.User.Username)
	}

	claims, err := service.VerifyToken(result.Token)
	if err != nil {
		t.Fatalf("VerifyToken returned error: %v", err)
	}
	if claims.UserID != "user-1" {
		t.Fatalf("expected user-1, got %q", claims.UserID)
	}
}

func TestVerifyTokenRejectsExpiredToken(t *testing.T) {
	service := NewService(&fakeUserRepository{}, Config{Secret: "test-secret", ExpiresHours: 1})
	token, err := service.SignToken(Claims{UserID: "user-1", Username: "admin", ExpiresAt: time.Now().Add(-time.Minute).Unix()})
	if err != nil {
		t.Fatalf("SignToken returned error: %v", err)
	}

	_, err = service.VerifyToken(token)
	if err == nil {
		t.Fatal("expected expired token error")
	}
}

func TestChangePasswordVerifiesOldPassword(t *testing.T) {
	repository := &fakeUserRepository{user: User{
		ID: "user-1", Username: "admin", PasswordHash: HashPassword("old-pass", "fixed-salt"), Status: "active",
	}}
	service := NewService(repository, Config{Secret: "test-secret", ExpiresHours: 1})

	err := service.ChangePassword(context.Background(), "admin", "old-pass", "new-pass")
	if err != nil {
		t.Fatalf("ChangePassword returned error: %v", err)
	}
	if repository.updatedHash == "" {
		t.Fatal("expected password hash to be updated")
	}
	if !VerifyPassword("new-pass", repository.updatedHash) {
		t.Fatal("expected updated hash to verify new password")
	}
}
