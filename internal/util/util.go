package util

import (
	"math/rand"
	"strconv"

	"golang.org/x/crypto/bcrypt"
)

func GenerateRandomString(length int) string {
	charset := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := 0; i < length; i++ {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b[:])
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func CheckNumberLuhn(s string) bool {
	if len(s) < 3 {
		return false
	}

	checkDigit := s[len(s)-1]
	numberPart := s[:len(s)-1]

	number, err := strconv.Atoi(numberPart)
	if err != nil {
		return false
	}

	checkNumber := CalcChecksumLuhn(number)

	if checkNumber != 0 {
		checkNumber = 10 - checkNumber
	}

	return checkDigit == byte('0'+checkNumber)
}

func CalcChecksumLuhn(number int) int {
	var luhn int

	for i := 0; number > 0; i++ {
		cur := number % 10

		if i%2 == 0 {
			cur = cur * 2
			if cur > 9 {
				cur = cur%10 + cur/10
			}
		}

		luhn += cur
		number = number / 10
	}
	return luhn % 10
}
