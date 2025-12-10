package auth

import (
	"errors"
	"time"

	"app_backend/internal/domain"
	"app_backend/internal/ports"

	"github.com/golang-jwt/jwt/v5"
)

type JWTService struct {
	secret []byte
}

func NewJWT(secret string) ports.TokenService {
	return &JWTService{secret: []byte(secret)}
}

func (j *JWTService) GenerateUserToken(id domain.UserID) (string, error) {
	return j.generateToken(string(id), "user")
}

func (j *JWTService) GenerateProviderToken(id domain.ProviderID) (string, error) {
	return j.generateToken(string(id), "provider")
}

func (j *JWTService) generateToken(id, typ string) (string, error) {
	claims := jwt.MapClaims{
		"sub":  id,
		"type": typ,
		"exp":  time.Now().Add(72 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.secret)
}

func (j *JWTService) Parse(tokenString string) (string, string, error) {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return j.secret, nil
	})
	if err != nil || !token.Valid {
		return "", "", errors.New("invalid token")
	}

	claims := token.Claims.(jwt.MapClaims)
	return claims["sub"].(string), claims["type"].(string), nil
}
