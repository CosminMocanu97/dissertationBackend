package types

type Email struct {
	Recipients []string `json:"recipients" binding:""`
	Payload    string   `json:"payload" binding:"required"`
	Subject    string   `json:"subject" binding:"optional"`
}

type LoginCredentials struct {
	Email    string `form:"email"`
	Password string `form:"password"`
}

type LoginResponse struct {
	Id    int64  `json:"id"`
	Token string `json:"token"`
}

type RegistrationData struct {
	Email       string `form:"email"`
	Password    string `form:"password"`
}

type User struct {
	ID              int64
	Email           string
	Passhash        string
	IsActivated     bool
	IsAdmin 		bool
	ActivationToken string
}