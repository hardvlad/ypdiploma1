package auth

import (
	"database/sql"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

type Claims struct {
	jwt.RegisteredClaims
	UserID int
}

func CreateToken(tokenExpiration time.Duration, userID int, secretKey string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(tokenExpiration)),
		},
		UserID: userID,
	})

	tokenString, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func GetUserID(tokenString string, secretKey string) (int, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(secretKey), nil
	})

	if err != nil {
		return 0, err
	}

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

func CreateNewUser(db *sql.DB, key string) (int, string, error) {
	var userID int
	err := db.QueryRow("INSERT INTO users (created_at) values (now()) RETURNING id").Scan(&userID)
	if err != nil {
		return 0, "", err
	}

	token, err := CreateToken(time.Hour*24, int(userID), key)
	if err != nil {
		return 0, "", err
	}

	return int(userID), token, nil
}
