package utils

import (
	"GH-Server/pkg/zaplog"
	"fmt"
	"os"
	"strconv"

	"golang.org/x/crypto/bcrypt"
)

var (
	passwordCost, _ = strconv.Atoi(os.Getenv("PASSWORD_COST"))
)

func GenerateFromPassword(original string) (string, error) {
	bytesPassword, err := bcrypt.GenerateFromPassword([]byte(original), passwordCost)
	if err != nil {
		zaplog.Zap.Error(fmt.Sprintf("generate from bcrypt failed: %v", err))
		return "", err
	}
	return string(bytesPassword), nil
}

func ComparedWithPassword(hash string, original string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(original))
	if err != nil {
		zaplog.Zap.Error(fmt.Sprintf("compare hash failed: %v", err))
		return err
	}
	return nil
}
