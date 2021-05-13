package utils

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/badoux/checkmail"
	"github.com/joho/godotenv"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/CosminMocanu97/dissertationBackend/pkg/log"
)

const (
	MaxPasswordLength     = 6
	ActivationTokenLength = 50
	charset                 = "abcdefghijklmnopqrstuvwxyz" +
		"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

var (
	seededRand *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))
)

func GetEnvVars() {
	err := godotenv.Load("credentials.env")
	if err != nil {
		log.Fatal("Error loading the .env file")
	}
}

// Note: it accepts email format such as someting@domain
func ValidateEmail(email string) bool {
	err := checkmail.ValidateFormat(email)
	if err != nil {
		log.Error("Error validating the email %s: %s", email, err)
		return false
	}

	return true
}

func ValidatePhoneNumber(phoneNumber string) bool {
	if len(phoneNumber) < 10 {
		return false
	}
	re := regexp.MustCompile(`^(?:(?:\(?(?:00|\+)([1-4]\d\d|[1-9]\d?)\)?)?[\-\.\ \\\/]?)?((?:\(?\d{1,}\)?[\-\.\ \\\/]?){0,})(?:[\-\.\ \\\/]?(?:#|ext\.?|extension|x)[\-\.\ \\\/]?(\d+))?$`)
	phoneNumberIsValid := re.MatchString(phoneNumber)
	return phoneNumberIsValid
}

func ValidatePassword(password string) bool {
	return len(password) >= MaxPasswordLength
}

// GenerateAccountActivationToken generates a token in the form of <user_id>_<token>
func GenerateRawAccountActivationToken() string {
	var buffer bytes.Buffer
	for index := 0; index < ActivationTokenLength; index++ {
		buffer.WriteByte(charset[seededRand.Intn(len(charset))])
	}

	return buffer.String()
}

func BuildActivationTokenWithUserId(userID int64, rawActivationToken string) string {
	var buffer bytes.Buffer
	stringUserId := strconv.Itoa(int(userID))
	buffer.WriteString(stringUserId)
	buffer.WriteString("_")
	buffer.WriteString(rawActivationToken)

	return buffer.String()
}

func GetUserIDAndActivationTokenFromRawActivationToken(rawToken string) (int64, string, error) {
	split := strings.Split(rawToken, "_")
	if len(split) != 2 {
		errorMessage := fmt.Sprintf("Error splitting the activation token %s, %d parts found, and 2 were expected", rawToken, len(split))
		log.Error(errorMessage)
		return 0, "", errors.New(errorMessage)
	}

	userId, err := strconv.Atoi(split[0])
	if err != nil {
		errorMessage := fmt.Sprintf("Error convering the raw user ID %s to integer", split[0])
		log.Error(errorMessage)
		return 0, "", errors.New(errorMessage)
	}

	return int64(userId), split[1], nil
}