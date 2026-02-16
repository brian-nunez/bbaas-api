package users

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/brian-nunez/bbaas-api/internal/data"
	"github.com/brian-nunez/bbaas-api/internal/security"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailRequired      = errors.New("email is required")
	ErrInvalidEmail       = errors.New("email must be valid")
	ErrPasswordRequired   = errors.New("password is required")
	ErrPasswordTooShort   = errors.New("password must be at least 8 characters")
	ErrEmailAlreadyExists = errors.New("email is already registered")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrSessionExpired     = errors.New("session has expired")
)

type Service struct {
	store      *data.Store
	now        func() time.Time
	sessionTTL time.Duration
}

func NewService(store *data.Store) *Service {
	return &Service{
		store:      store,
		now:        time.Now,
		sessionTTL: 30 * 24 * time.Hour,
	}
}

func (s *Service) Register(ctx context.Context, email string, password string) (User, error) {
	normalizedEmail, err := normalizeEmail(email)
	if err != nil {
		return User{}, err
	}
	if err := validatePassword(password); err != nil {
		return User{}, err
	}

	existingUser, found, err := s.store.GetUserByEmail(ctx, normalizedEmail)
	if err != nil {
		return User{}, fmt.Errorf("lookup existing user: %w", err)
	}
	if found && existingUser.ID != "" {
		return User{}, ErrEmailAlreadyExists
	}

	passwordHashBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, fmt.Errorf("hash password: %w", err)
	}

	userID, err := security.GeneratePrefixedToken("usr", 18)
	if err != nil {
		return User{}, fmt.Errorf("generate user id: %w", err)
	}

	role := "user"
	usersCount, err := s.store.CountUsers(ctx)
	if err != nil {
		return User{}, fmt.Errorf("count existing users: %w", err)
	}
	if usersCount == 0 {
		role = "admin"
	}

	now := s.now().UTC()
	record := data.UserRecord{
		ID:           userID,
		Email:        normalizedEmail,
		PasswordHash: string(passwordHashBytes),
		Role:         role,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.store.CreateUser(ctx, record); err != nil {
		return User{}, fmt.Errorf("create user: %w", err)
	}

	return mapUserRecord(record), nil
}

func (s *Service) Login(ctx context.Context, email string, password string) (User, string, error) {
	normalizedEmail, err := normalizeEmail(email)
	if err != nil {
		return User{}, "", ErrInvalidCredentials
	}

	userRecord, found, err := s.store.GetUserByEmail(ctx, normalizedEmail)
	if err != nil {
		return User{}, "", fmt.Errorf("lookup user by email: %w", err)
	}
	if !found {
		return User{}, "", ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(userRecord.PasswordHash), []byte(password)); err != nil {
		return User{}, "", ErrInvalidCredentials
	}

	sessionToken, err := security.GeneratePrefixedToken("sess", 24)
	if err != nil {
		return User{}, "", fmt.Errorf("generate session token: %w", err)
	}

	sessionID, err := security.GeneratePrefixedToken("sid", 12)
	if err != nil {
		return User{}, "", fmt.Errorf("generate session id: %w", err)
	}

	now := s.now().UTC()
	sessionRecord := data.SessionRecord{
		ID:        sessionID,
		UserID:    userRecord.ID,
		TokenHash: security.DigestSHA256(sessionToken),
		CreatedAt: now,
		ExpiresAt: now.Add(s.sessionTTL),
	}
	if err := s.store.CreateSession(ctx, sessionRecord); err != nil {
		return User{}, "", fmt.Errorf("create session: %w", err)
	}

	return mapUserRecord(userRecord), sessionToken, nil
}

func (s *Service) AuthenticateSession(ctx context.Context, sessionToken string) (User, bool, error) {
	sessionToken = strings.TrimSpace(sessionToken)
	if sessionToken == "" {
		return User{}, false, nil
	}

	sessionRecord, userRecord, found, err := s.store.GetSessionWithUserByTokenHash(ctx, security.DigestSHA256(sessionToken))
	if err != nil {
		return User{}, false, fmt.Errorf("lookup session: %w", err)
	}
	if !found {
		return User{}, false, nil
	}

	now := s.now().UTC()
	if !sessionRecord.ExpiresAt.After(now) {
		_ = s.store.DeleteSessionByTokenHash(ctx, sessionRecord.TokenHash)
		return User{}, false, ErrSessionExpired
	}

	return mapUserRecord(userRecord), true, nil
}

func (s *Service) Logout(ctx context.Context, sessionToken string) error {
	sessionToken = strings.TrimSpace(sessionToken)
	if sessionToken == "" {
		return nil
	}

	if err := s.store.DeleteSessionByTokenHash(ctx, security.DigestSHA256(sessionToken)); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}

	return nil
}

func (s *Service) ListUsersForViewer(ctx context.Context, viewer User) ([]User, error) {
	if viewer.IsAdmin() {
		allUsers, err := s.store.ListUsers(ctx, 50)
		if err != nil {
			return nil, fmt.Errorf("list users as admin: %w", err)
		}

		users := make([]User, 0, len(allUsers))
		for _, userRecord := range allUsers {
			users = append(users, mapUserRecord(userRecord))
		}

		return users, nil
	}

	currentUser, found, err := s.store.GetUserByID(ctx, viewer.ID)
	if err != nil {
		return nil, fmt.Errorf("lookup current user: %w", err)
	}
	if !found {
		return []User{}, nil
	}

	return []User{mapUserRecord(currentUser)}, nil
}

func normalizeEmail(email string) (string, error) {
	trimmed := strings.TrimSpace(strings.ToLower(email))
	if trimmed == "" {
		return "", ErrEmailRequired
	}

	if _, err := mail.ParseAddress(trimmed); err != nil {
		return "", ErrInvalidEmail
	}

	return trimmed, nil
}

func validatePassword(password string) error {
	if password == "" {
		return ErrPasswordRequired
	}
	if len(password) < 8 {
		return ErrPasswordTooShort
	}

	return nil
}

func mapUserRecord(record data.UserRecord) User {
	return User{
		ID:        record.ID,
		Email:     record.Email,
		Role:      record.Role,
		CreatedAt: record.CreatedAt,
		UpdatedAt: record.UpdatedAt,
	}
}
