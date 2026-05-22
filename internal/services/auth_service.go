package services

import (
	"errors"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID  int    `json:"user_id"`
	Account string `json:"account"`
	IsSuper int    `json:"is_super"`
	jwt.RegisteredClaims
}

func GenerateToken(userID int, account string, isSuper int, secret string, expireHours int) (string, error) {
	claims := Claims{
		UserID:  userID,
		Account: account,
		IsSuper: isSuper,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expireHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func ParseToken(tokenString string, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("无效的 token")
	}

	return claims, nil
}

func (c *Claims) GetUserID() string {
	return strconv.Itoa(c.UserID)
}
