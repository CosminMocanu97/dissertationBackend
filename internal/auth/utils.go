package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"github.com/CosminMocanu97/dissertationBackend/pkg/log"
)

func ComputePasswordHash(password string) string {
	bytePassword := []byte(password)
	hasher := sha256.New()
	_, err := hasher.Write(bytePassword)
	if err != nil {
		log.Error("Error computing the hash of the password: %s", err)
	}
	hashedPassword := base64.URLEncoding.EncodeToString(hasher.Sum(nil))

	return hashedPassword
}
