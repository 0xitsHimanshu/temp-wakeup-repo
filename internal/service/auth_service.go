package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"upbot-server-go/internal/models"
	"upbot-server-go/internal/repository"

	"github.com/golang-jwt/jwt/v5"
)

type AuthService interface {
	LoginWithGoogle(accessToken string) (*models.User, string, error)
}

type authService struct {
	repo      repository.TaskRepository // Using TaskRepo as it has User methods
	jwtSecret string
}

func NewAuthService(repo repository.TaskRepository, jwtSecret string) AuthService {
	return &authService{
		repo:      repo,
		jwtSecret: jwtSecret,
	}
}

type GoogleUserInfo struct {
	Email string `json:"email"`
}

func (s *authService) LoginWithGoogle(accessToken string) (*models.User, string, error) {
	// 1. Verify Google Token
	resp, err := http.Get("https://www.googleapis.com/oauth2/v1/tokeninfo?access_token=" + accessToken)
	if err != nil {
		return nil, "", fmt.Errorf("failed to call google api: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", errors.New("invalid google access token")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}

	var userInfo GoogleUserInfo
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return nil, "", err
	}

	// 2. Find or Create User
	user, err := s.repo.GetUserByEmail(userInfo.Email)
	if err != nil {
		// Assume error means not found, create new user
		newUser := &models.User{Email: userInfo.Email}
		if err := s.repo.CreateUser(newUser); err != nil {
			return nil, "", fmt.Errorf("failed to create user: %w", err)
		}
		user = newUser
	}

	// 3. Generate JWT
	token, err := s.generateJWT(user)
	if err != nil {
		return nil, "", err
	}

	return user, token, nil
}

func (s *authService) generateJWT(user *models.User) (string, error) {
	claims := jwt.MapClaims{
		"userId": user.ID,
		"email":  user.Email,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}
