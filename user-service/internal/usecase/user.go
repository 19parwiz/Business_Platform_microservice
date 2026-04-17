package usecase

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/url"
	"strings"

	"github.com/19parwiz/user-service/internal/domain"
)

// MailSender defines only what the usecase needs (SendEmail)
type MailSender interface {
	SendEmail(to []string, subject, body string) error
}

type UserUsecase struct {
	aiRepo          AutoIncRepo
	userRepo        UserRepo
	pHasher         PasswordHasher
	mailer          MailSender
	publicBaseURL   string // trimmed origin for confirmation links (e.g. https://app.example.com)
}

func generateToken() string {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}

func NewUserUsecase(ai AutoIncRepo, userRepo UserRepo, pHasher PasswordHasher, mailer MailSender, publicBaseURL string) UserUsecase {
	return UserUsecase{
		aiRepo:        ai,
		userRepo:      userRepo,
		pHasher:       pHasher,
		mailer:        mailer,
		publicBaseURL: strings.TrimSpace(publicBaseURL),
	}
}

func (uc UserUsecase) Register(ctx context.Context, req domain.User) (domain.User, error) {
	emailFilter := domain.UserFilter{
		Email: &req.Email,
	}
	if exists, _ := uc.userRepo.GetWithFilter(ctx, emailFilter); exists != (domain.User{}) {
		return domain.User{}, domain.ErrUserExists
	}

	id, err := uc.aiRepo.Next(ctx, domain.UserDB)
	if err != nil {
		return domain.User{}, err
	}
	req.ID = id

	req.HashedPassword, err = uc.pHasher.Hash(req.HashedPassword)
	if err != nil {
		return domain.User{}, err
	}

	// === ADD THIS: generate and set email confirmation token here ===
	req.EmailConfirmToken = generateToken() // Implement generateToken to create a random token string

	err = uc.userRepo.Create(ctx, req)
	if err != nil {
		return domain.User{}, err
	}

	base := strings.TrimSuffix(uc.publicBaseURL, "/")
	if base == "" {
		base = "http://localhost:3000"
	}
	q := url.Values{}
	q.Set("email", req.Email)
	q.Set("token", req.EmailConfirmToken)
	confirmationLink := base + "/confirm?" + q.Encode()

	emailBody := "Confirm your email for " + req.Email + " by opening:\n\n" + confirmationLink

	err = uc.mailer.SendEmail([]string{req.Email}, "Confirm your email", emailBody)

	if err != nil {
		return domain.User{}, err
	}

	return domain.User{
		ID:    id,
		Name:  req.Name,
		Email: req.Email, //  Added this line for testing
	}, nil
}

func (uc UserUsecase) Authenticate(ctx context.Context, req domain.User) (domain.User, error) {
	emailFilter := domain.UserFilter{
		Email: &req.Email,
	}
	existingUser, err := uc.userRepo.GetWithFilter(ctx, emailFilter)
	if err != nil {
		return domain.User{}, err
	}

	if existingUser == (domain.User{}) {
		return domain.User{}, domain.ErrUserNotFound
	}

	isValid := uc.pHasher.Verify(existingUser.HashedPassword, req.HashedPassword)
	if !isValid {
		return domain.User{}, domain.ErrInvalidPassword
	}

	return domain.User{
		ID:   existingUser.ID,
		Name: existingUser.Name,
	}, nil
}

func (uc UserUsecase) Get(ctx context.Context, filter domain.UserFilter) (domain.User, error) {
	user, err := uc.userRepo.GetWithFilter(ctx, filter)
	if err != nil {
		return domain.User{}, err
	}
	return user, nil
}
