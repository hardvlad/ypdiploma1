// Package auth создание и проверка JWT токенов
package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// Claims добавление в стандартный набор клеймов JWT токена клейма UserID
type Claims struct {
	jwt.RegisteredClaims
	UserID int
}

// CreateToken функция создания JWT токена
// возвращает или ошибку или созданный токен
func CreateToken(tokenExpiration time.Duration, userID int, secretKey string) (string, error) {
	// создание токена с нужными клеймами
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(tokenExpiration)),
		},
		UserID: userID,
	})

	// подписание токена и возвращение его в виде строки
	tokenString, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// GetUserID проверка токена и получение из него клейма UserID
// возвращает или ошибку или валидный UserID
func GetUserID(tokenString string, secretKey string) (int, error) {
	// проверка токена и парсинг содержащихся в нем клеймов
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(secretKey), nil
	})

	if err != nil {
		return 0, err
	}

	// проверка метода шифрование, недопущение none алгоритма и проверка валидности токена
	if token.Method.Alg() != "HS256" {
		return 0, jwt.ErrTokenMalformed
	}

	if !token.Valid {
		return 0, jwt.ErrTokenUnverifiable
	}

	if claims.UserID == 0 {
		return 0, jwt.ErrTokenMalformed
	}

	return claims.UserID, nil
}
