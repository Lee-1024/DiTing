package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

type Config struct {
	Secret       string
	ExpiresHours int
}

type User struct {
	ID           string   `json:"id"`
	Username     string   `json:"username"`
	DisplayName  string   `json:"displayName"`
	Email        string   `json:"email"`
	PasswordHash string   `json:"-"`
	Status       string   `json:"status"`
	Roles        []string `json:"roles"`
}

type Claims struct {
	UserID    string   `json:"userId"`
	Username  string   `json:"username"`
	Roles     []string `json:"roles"`
	ExpiresAt int64    `json:"exp"`
}

type LoginResult struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type UserRepository interface {
	FindByUsername(ctx context.Context, username string) (User, error)
	UpdatePassword(ctx context.Context, userID, passwordHash string) error
}

type Service struct {
	repository UserRepository
	config     Config
}

var ErrInvalidCredentials = errors.New("invalid username or password")

func NewService(repository UserRepository, config Config) *Service {
	if config.ExpiresHours <= 0 {
		config.ExpiresHours = 24
	}
	return &Service{repository: repository, config: config}
}

func (s *Service) Login(ctx context.Context, username, password string) (LoginResult, error) {
	user, err := s.repository.FindByUsername(ctx, username)
	if err != nil {
		return LoginResult{}, ErrInvalidCredentials
	}
	if user.Status != "active" {
		return LoginResult{}, ErrInvalidCredentials
	}
	if !VerifyPassword(password, user.PasswordHash) {
		return LoginResult{}, ErrInvalidCredentials
	}
	token, err := s.SignToken(Claims{
		UserID:    user.ID,
		Username:  user.Username,
		Roles:     user.Roles,
		ExpiresAt: time.Now().Add(time.Duration(s.config.ExpiresHours) * time.Hour).Unix(),
	})
	if err != nil {
		return LoginResult{}, err
	}
	return LoginResult{Token: token, User: user}, nil
}

func (s *Service) ChangePassword(ctx context.Context, username, oldPassword, newPassword string) error {
	user, err := s.repository.FindByUsername(ctx, username)
	if err != nil {
		return ErrInvalidCredentials
	}
	if !VerifyPassword(oldPassword, user.PasswordHash) {
		return ErrInvalidCredentials
	}
	return s.repository.UpdatePassword(ctx, user.ID, HashPassword(newPassword, randomSalt()))
}

func (s *Service) SignToken(claims Claims) (string, error) {
	header := map[string]string{"alg": "HS256", "typ": "JWT"}
	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	unsigned := base64.RawURLEncoding.EncodeToString(headerJSON) + "." + base64.RawURLEncoding.EncodeToString(claimsJSON)
	signature := sign(unsigned, s.config.Secret)
	return unsigned + "." + signature, nil
}

func (s *Service) VerifyToken(token string) (Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Claims{}, errors.New("invalid token")
	}
	unsigned := parts[0] + "." + parts[1]
	expected := sign(unsigned, s.config.Secret)
	if !hmac.Equal([]byte(expected), []byte(parts[2])) {
		return Claims{}, errors.New("invalid token signature")
	}
	claimsJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Claims{}, err
	}
	var claims Claims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return Claims{}, err
	}
	if claims.ExpiresAt <= time.Now().Unix() {
		return Claims{}, errors.New("token expired")
	}
	return claims, nil
}

func sign(value, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(value))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func HashPassword(password, salt string) string {
	sum := sha256.Sum256([]byte(salt + ":" + password))
	return fmt.Sprintf("sha256$%s$%s", salt, hex.EncodeToString(sum[:]))
}

func VerifyPassword(password, hash string) bool {
	parts := strings.Split(hash, "$")
	if len(parts) != 3 || parts[0] != "sha256" {
		return false
	}
	return hmac.Equal([]byte(HashPassword(password, parts[1])), []byte(hash))
}

func randomSalt() string {
	var data [16]byte
	if _, err := io.ReadFull(rand.Reader, data[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(data[:])
}
