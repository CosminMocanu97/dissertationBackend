package auth

import (
	"fmt"
	"os"
	"time"

	"github.com/CosminMocanu97/dissertationBackend/pkg/log"
	"github.com/dgrijalva/jwt-go"
)

var (
	NoJWTTokenProvidedError = fmt.Errorf("the jwt token was not provided")
	TokenIsExpired = fmt.Errorf("the jwt token has expired")
)

type LoginService interface {
	LoginUser(email string, password string) bool
}

type LoginInformation struct {
	email    string
	password string
}

func (info *LoginInformation) LoginUser(email string, password string) bool {
	return info.email == email && info.password == password
}

type JWTService interface {
	GenerateToken(id int64, email string) string
	ValidateToken(token string) (*jwt.Token, error)
}

// the data embedded in JWT
type AuthCustomClaims struct {
	Id          int64  `json:"id"`
	Email       string `json:"name"`
	IsActivated bool   `json:"isActivated"`
	jwt.StandardClaims
}

type RefreshAuthCustomClaims struct {
	Email string `json:"email"`
	jwt.StandardClaims
}

type jwtServices struct {
	secretKey string
	issuer    string
}

//auth-jwt
func JWTAuthService() *jwtServices {
	return &jwtServices{
		secretKey: getSecretKey(),
		issuer:    "Dissertation",
	}
}

func getSecretKey() string {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "secret"
	}
	return secret
}

func (service *jwtServices) GenerateToken(id int64, email string, isActivated bool) map[string]string {
	claims := &AuthCustomClaims{
		id,
		email,
		isActivated,
		jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Hour * 24).Unix(),
			Issuer:    service.issuer,
			IssuedAt:  time.Now().Unix(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	//encoded string
	t, err := token.SignedString([]byte(service.secretKey))
	if err != nil {
		panic(err)
	}

	rtClaims := &RefreshAuthCustomClaims{
		email,
		jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Hour * 48).Unix(),
			Issuer:    service.issuer,
			IssuedAt:  time.Now().Unix(),
		},
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, rtClaims)

	//encoded string
	rt, err := refreshToken.SignedString([]byte(service.secretKey))
	if err != nil {
		panic(err)
	}

	return map[string]string{
		"access_token":  t,
		"refresh_token": rt,
	}
}

func (service *jwtServices) ValidateToken(encodedToken string) (*jwt.Token, error) {
	if len(encodedToken) == 0 {
		return nil, NoJWTTokenProvidedError
	}
	var customClaims AuthCustomClaims
	/*return jwt.ParseWithClaims(encodedToken, &customClaims, func(token *jwt.Token) (interface{}, error) {
		if _, isvalid := token.Method.(*jwt.SigningMethodHMAC); !isvalid {
			return nil, fmt.Errorf("invalid token %s", token.Header["alg"])

		}  
		if customClaims.ExpiresAt < time.Now().Unix() {
			fmt.Println("A expirat")
			return []byte(service.secretKey), TokenIsExpired
		}

		return []byte(service.secretKey), nil
	})
	*/
	token, err := jwt.ParseWithClaims(encodedToken, &customClaims, func(token *jwt.Token) (interface{}, error) {
		if _, isvalid := token.Method.(*jwt.SigningMethodHMAC); !isvalid {
			return nil, fmt.Errorf("invalid token %s", token.Header["alg"])

		}  
		return []byte(service.secretKey), nil
	})

	claims := token.Claims.(*AuthCustomClaims) 
	if claims.ExpiresAt < time.Now().Unix() {
		return nil, TokenIsExpired
	}

	if err != nil {
		log.Error("%s", err)
		return nil, err
	}

	return token, nil
}