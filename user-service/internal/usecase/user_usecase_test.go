package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/19parwiz/user-service/internal/domain"
	"github.com/stretchr/testify/assert"
)

// --- Mock AutoIncRepo ---
type mockAutoIncRepo struct{}

func (m *mockAutoIncRepo) Next(ctx context.Context, key string) (uint64, error) {
	return 1, nil
}

// --- Mock UserRepo ---
type mockUserRepo struct {
	users map[string]domain.User
}

func (m *mockUserRepo) Create(ctx context.Context, user domain.User) error {
	m.users[user.Email] = user
	return nil
}

func (m *mockUserRepo) GetWithFilter(ctx context.Context, filter domain.UserFilter) (domain.User, error) {
	if filter.Email != nil {
		if user, ok := m.users[*filter.Email]; ok {
			return user, nil
		}
	}
	return domain.User{}, errors.New("user not found")
}

func (m *mockUserRepo) Delete(ctx context.Context, filter domain.UserFilter) error {
	if filter.Email != nil {
		delete(m.users, *filter.Email)
		return nil
	}
	return errors.New("No user matched for deletion")
}

func (m *mockUserRepo) Update(ctx context.Context, filter domain.UserFilter, update domain.UserUpdate) error {
	if filter.Email != nil {
		user, ok := m.users[*filter.Email]
		if !ok {
			return errors.New("user not found")
		}
		if update.Name != nil {
			user.Name = *update.Name
		}
		if update.HashedPassword != nil {
			user.HashedPassword = *update.HashedPassword
		}
		m.users[*filter.Email] = user
		return nil
	}
	return errors.New("No matching user")
}

// --- Mock Hasher ---
type mockPasswordHasher struct{}

func (m *mockPasswordHasher) Hash(password string) (string, error) {
	return "hashed_" + password, nil
}

func (m *mockPasswordHasher) Verify(hash, password string) bool {
	return hash == "hashed_"+password
}

// --- Mock Mailer ---
type mockMailer struct{}

func (m *mockMailer) SendEmail(to []string, subject, body string) error {
	return nil
}

// --- Test Case ---
func TestUserUsecase_Register(t *testing.T) {
	ctx := context.Background()

	// Setup mocks
	autoInc := &mockAutoIncRepo{}
	userRepo := &mockUserRepo{users: make(map[string]domain.User)}
	hasher := &mockPasswordHasher{}
	mailer := &mockMailer{}

	// Initialize the usecase
	uc := NewUserUsecase(autoInc, userRepo, hasher, mailer, "https://app.example.com")

	// Input data
	user := domain.User{
		Name:           "Ali Parwiz",
		Email:          "ali@example.com",
		HashedPassword: "hashed_mypassword", //  Use correct field name
	}

	// Execute
	result, err := uc.Register(ctx, user)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, uint64(1), result.ID)
	assert.Equal(t, "Ali Parwiz", result.Name)
	assert.Equal(t, "ali@example.com", result.Email)
}
