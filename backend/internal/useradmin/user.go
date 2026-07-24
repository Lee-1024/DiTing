package useradmin

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

type User struct {
	ID          string    `json:"id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"displayName"`
	Email       string    `json:"email"`
	Status      string    `json:"status"`
	Roles       []string  `json:"roles"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type CreateUserRequest struct {
	Username    string   `json:"username"`
	Password    string   `json:"password"`
	DisplayName string   `json:"displayName"`
	Email       string   `json:"email"`
	Status      string   `json:"status"`
	Roles       []string `json:"roles"`
}

type UpdateUserRequest struct {
	DisplayName string   `json:"displayName"`
	Email       string   `json:"email"`
	Status      string   `json:"status"`
	Roles       []string `json:"roles"`
}

type ResetPasswordRequest struct {
	Password string `json:"password"`
}

type Role struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type Repository interface {
	ListUsers(ctx context.Context) ([]User, error)
	CreateUser(ctx context.Context, request CreateUserRequest) (User, error)
	UpdateUser(ctx context.Context, id string, request UpdateUserRequest) (User, error)
	ResetPassword(ctx context.Context, id string, password string) error
	DeleteUser(ctx context.Context, id string) error
	ListRoles(ctx context.Context) ([]Role, error)
}

var (
	ErrNotFound       = errors.New("user not found")
	ErrConflict       = errors.New("user already exists")
	ErrLastAdmin      = errors.New("cannot remove the last admin")
	ErrRoleNotFound   = errors.New("role not found")
	ErrWeakPassword   = errors.New("password must be at least 6 characters")
	ErrInvalidRequest = errors.New("invalid request")
)

type MemoryRepository struct {
	mu         sync.Mutex
	users      []User
	roles      []Role
	passwords  map[string]string
	nextUserID int
}

// NewMemoryRepository 创建并初始化 New Memory Repository 实例。
func NewMemoryRepository() *MemoryRepository {
	now := time.Now().UTC()
	return &MemoryRepository{
		nextUserID: 1,
		passwords:  map[string]string{},
		roles: []Role{{
			ID:          "role-admin",
			Name:        "admin",
			Description: "System administrator",
			CreatedAt:   now,
			UpdatedAt:   now,
		}},
	}
}

// ListUsers 查询并返回 List Users 列表。
func (r *MemoryRepository) ListUsers(_ context.Context) ([]User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	result := make([]User, len(r.users))
	copy(result, r.users)
	return result, nil
}

// CreateUser 创建新的 Create User。
func (r *MemoryRepository) CreateUser(_ context.Context, request CreateUserRequest) (User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.usernameExists(request.Username) {
		return User{}, ErrConflict
	}
	if !r.rolesExist(request.Roles) {
		return User{}, ErrRoleNotFound
	}
	now := time.Now().UTC()
	user := User{
		ID:          fmt.Sprintf("user-%d", r.nextUserID),
		Username:    request.Username,
		DisplayName: request.DisplayName,
		Email:       request.Email,
		Status:      normalizeStatus(request.Status),
		Roles:       normalizeRoles(request.Roles),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	r.nextUserID++
	r.users = append(r.users, user)
	r.passwords[user.ID] = request.Password
	return user, nil
}

// UpdateUser 更新指定的 Update User。
func (r *MemoryRepository) UpdateUser(_ context.Context, id string, request UpdateUserRequest) (User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.rolesExist(request.Roles) {
		return User{}, ErrRoleNotFound
	}
	for index, user := range r.users {
		if user.ID != id {
			continue
		}
		if r.removesLastAdmin(user, request.Roles, request.Status) {
			return User{}, ErrLastAdmin
		}
		user.DisplayName = request.DisplayName
		user.Email = request.Email
		user.Status = normalizeStatus(request.Status)
		user.Roles = normalizeRoles(request.Roles)
		user.UpdatedAt = time.Now().UTC()
		r.users[index] = user
		return user, nil
	}
	return User{}, ErrNotFound
}

// ResetPassword 重置 Reset Password。
func (r *MemoryRepository) ResetPassword(_ context.Context, id string, password string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, user := range r.users {
		if user.ID == id {
			r.passwords[id] = password
			return nil
		}
	}
	return ErrNotFound
}

// DeleteUser 删除指定的 Delete User。
func (r *MemoryRepository) DeleteUser(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for index, user := range r.users {
		if user.ID != id {
			continue
		}
		if r.isLastActiveAdmin(user) {
			return ErrLastAdmin
		}
		r.users = append(r.users[:index], r.users[index+1:]...)
		delete(r.passwords, id)
		return nil
	}
	return ErrNotFound
}

// ListRoles 查询并返回 List Roles 列表。
func (r *MemoryRepository) ListRoles(_ context.Context) ([]Role, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	result := make([]Role, len(r.roles))
	copy(result, r.roles)
	return result, nil
}

// usernameExists 处理 username Exists 相关逻辑。
func (r *MemoryRepository) usernameExists(username string) bool {
	for _, user := range r.users {
		if user.Username == username {
			return true
		}
	}
	return false
}

// rolesExist 处理 roles Exist 相关逻辑。
func (r *MemoryRepository) rolesExist(roles []string) bool {
	for _, roleName := range normalizeRoles(roles) {
		found := false
		for _, role := range r.roles {
			if role.Name == roleName {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// removesLastAdmin 删除指定的 removes Last Admin。
func (r *MemoryRepository) removesLastAdmin(user User, nextRoles []string, nextStatus string) bool {
	if !hasRole(user.Roles, "admin") || user.Status != "active" {
		return false
	}
	if hasRole(nextRoles, "admin") && normalizeStatus(nextStatus) == "active" {
		return false
	}
	return r.activeAdminCount() == 1
}

// isLastActiveAdmin 判断 is Last Active Admin 是否符合条件。
func (r *MemoryRepository) isLastActiveAdmin(user User) bool {
	return user.Status == "active" && hasRole(user.Roles, "admin") && r.activeAdminCount() == 1
}

// activeAdminCount 处理 active Admin Count 相关逻辑。
func (r *MemoryRepository) activeAdminCount() int {
	count := 0
	for _, user := range r.users {
		if user.Status == "active" && hasRole(user.Roles, "admin") {
			count++
		}
	}
	return count
}

// normalizeStatus 规范化 normalize Status 的默认值和边界值。
func normalizeStatus(value string) string {
	if value == "disabled" {
		return "disabled"
	}
	return "active"
}

// normalizeRoles 规范化 normalize Roles 的默认值和边界值。
func normalizeRoles(roles []string) []string {
	if len(roles) == 0 {
		return []string{"admin"}
	}
	result := make([]string, 0, len(roles))
	seen := map[string]bool{}
	for _, role := range roles {
		if role == "" || seen[role] {
			continue
		}
		seen[role] = true
		result = append(result, role)
	}
	return result
}

// hasRole 判断 has Role 是否符合条件。
func hasRole(roles []string, target string) bool {
	for _, role := range roles {
		if role == target {
			return true
		}
	}
	return false
}
