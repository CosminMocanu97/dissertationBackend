package database

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/bwmarrin/snowflake"

	"github.com/CosminMocanu97/dissertationBackend/internal/auth"
	"github.com/CosminMocanu97/dissertationBackend/pkg/gserror"
	"github.com/CosminMocanu97/dissertationBackend/internal/types"
	"github.com/CosminMocanu97/dissertationBackend/pkg/log"
)

const (
	ERROR_USER_ALREADY_EXISTS             = "The email already exists in the database"
	ERROR_USER_ALREADY_EXISTS_TRANSLATION = "Exista deja un cont cu acest email"

	ERROR_INVALID_CREDENTIALS             = "Invalid credentials"
	ERROR_INVALID_CREDENTIALS_TRANSLATION = "Emailul sau parola nu corespund"

	ERROR_USER_NOT_ACTIVATED = "Contul nu este activat"
)

// todo: add role field
func CreateUsersTable(db *sql.DB) error {
	createUsersQuery :=
		"create table if not exists users (  id serial primary key,  email text not null, passHash text not null, " +
			"isActivated bool not null default false, activationToken text not null);"
	_, err := db.Query(createUsersQuery)
	if err != nil {
		log.Error("Error creating the users table: %s", err)
	}

	log.Info("Successfully created users table")

	return err
}

// returns true if the user was successfully created, false otherwise, alongside the reason of failure
// todo: add validation for email, password and phone number
func AddNewUser(db *sql.DB, registrationData types.RegistrationData, activationToken string, id snowflake.ID) (bool, gserror.GSError) {
	userExists, err := UserExists(db, registrationData.Email, id)

	if err != gserror.NoError {
		log.Error("UUID: %s; Error verifying if the user with email %s exists: %s", id, registrationData.Email, err)
		return false, err
	} else {
		// if the email is not already in the database, try to add it
		if !userExists {
			passHash := auth.ComputePasswordHash(registrationData.Password)
			addUserStatement :=
				"INSERT INTO users(email, passHash, activationToken) VALUES($1, $2, $3);"

			_, err := db.Exec(addUserStatement, registrationData.Email, passHash, activationToken)
			if err != nil {
				log.Error("UUID: %s; Error adding a new user: %s", id, err)
				return false, gserror.NewInternalGSError(err)
			}

			log.Info("UUID: %s; Successfully added the new user with email %s", id, registrationData.Email)
			return true, gserror.NoError
		} else {
			return false, gserror.NewGSError(nil, ERROR_USER_ALREADY_EXISTS, ERROR_USER_ALREADY_EXISTS_TRANSLATION)
		}
	}
}
func UserExists(db *sql.DB, email string, id snowflake.ID) (bool, gserror.GSError) {
	userExistsQuery :=
		"SELECT * FROM users WHERE email=$1;"
	res, err := db.Exec(userExistsQuery, email)
	if err != nil {
		log.Error("UUID: %s; Error checking if the user with email %s exists: %s", id, email, err)
		return false, gserror.NewInternalGSError(err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		log.Error("UUID: %s; Error retrieving the number of rows matching the query %s, for email %s: %s",
			id, userExistsQuery, email, err)
		return false, gserror.NewInternalGSError(err)
	}

	if rowsAffected > 0 {
		return true, gserror.NoError
	} else {
		return false, gserror.NoError
	}
}

func GetActivationTokenForEmail(db *sql.DB, email string) (string, gserror.GSError) {
	getUserActivationTokenForEmailQuery :=
		"SELECT activationToken FROM users WHERE email=$1;" //#nosec

	var activationToken string
	row := db.QueryRow(getUserActivationTokenForEmailQuery, email)
	err := row.Scan(&activationToken)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Error("Error finding any entry for the email %s: %s", email, err)
		} else {
			log.Error("Error getting the activation token for email %s: %s", email, err)
		}
		return "", gserror.NewInternalGSError(err)
	}

	log.Info("Successfully retrieved activation token for email %s", email)
	return activationToken, gserror.NoError
}

func GetUserDetailsForEmail(db *sql.DB, email string, id snowflake.ID) (types.User, gserror.GSError) {
	var user types.User
	user.Email = email
	getUserIdForEmailQuery :=
		"SELECT id, passhash, isActivated, activationToken FROM users WHERE email=$1"

	row := db.QueryRow(getUserIdForEmailQuery, email)
	err := row.Scan(&user.ID, &user.Passhash, &user.IsActivated, &user.ActivationToken)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Error("UUID: %s; Error finding any entry for the email %s: %s", id, email, err)
		} else {
			log.Error("UUID: %s; Error getting the user details for email %s: %s", id, email, err)
		}
		return user, gserror.NewInternalGSError(err)
	}

	log.Info("Successfully retrieved the user details for email %s", email)
	return user, gserror.NoError
}

// VerifyLoginCredentials returns true if there as a user with the email and password provided as parameter, false otherwise
func VerifyLoginCredentials(db *sql.DB, email, password string, id snowflake.ID) (bool, gserror.GSError) {
	user, err := GetUserDetailsForEmail(db, email, id)
	if err != gserror.NoError {
		log.Error("UUID: %s; Error retrieving user details for email %s; %s", id, email, err)
		return false, err
	}

	// verify if the password matches
	passHash := auth.ComputePasswordHash(password)
	if user.Passhash == passHash {
		log.Info("UUID: %s; The user %s has successfully logged in", id, user.Email)
		return true, gserror.NoError
	} else {
		log.Info("UUID %s: The user %s tried to log in but the credentials are incorrect", id, user.Email)
		return false, gserror.NewGSError(nil, ERROR_INVALID_CREDENTIALS, ERROR_INVALID_CREDENTIALS_TRANSLATION)
	}
}

func VerifyActivationToken(db *sql.DB, userId int64, activationToken string, id snowflake.ID) (bool, gserror.GSError) {
	// skip security check for the activationToken
	expectedActivationTokenQuery := "SELECT activationToken FROM users WHERE id=$1" //#nosec
	row := db.QueryRow(expectedActivationTokenQuery, userId)

	var expectedActivationToken string
	err := row.Scan(&expectedActivationToken)

	if err != nil {
		log.Error("UUID: %s; Error retrieving the activation token for user with ID %d: %s", id, userId, err)
		return false, gserror.NewInternalGSError(err)
	}

	if expectedActivationToken == activationToken {
		log.Info("UUID: %s; The provided activation token for the account with ID %d is correct", id, userId)
		return true, gserror.NoError
	}
	return false, gserror.NoError
}

func UserIsActivated(db *sql.DB, email string, id snowflake.ID) (bool, gserror.GSError) {
	userIsVerifiedQuery := "SELECT isactivated FROM users WHERE email=$1"
	var isActivated bool
	row := db.QueryRow(userIsVerifiedQuery, email)
	err := row.Scan(&isActivated)
	if err != nil {
		log.Error("UUID: %s; Error verifying if the user with email %s is activated: %s", id, email, err)
		return false, gserror.NewInternalGSError(err)
	}
	if !isActivated {
		log.Info("UUID: %s; Account is not activated: %s", id, email)
		return false, gserror.NoError
	}
	return true, gserror.NoError
}

func ActivateAccount(db *sql.DB, userID int64, id snowflake.ID) gserror.GSError {
	activateAccountQuery :=
		"UPDATE users SET isactivated=true WHERE id=$1"
	_, err := db.Exec(activateAccountQuery, userID)
	if err != nil {
		log.Error("UUID: %s; Error activating the account for ID %d: %s", id, userID, err)
		return gserror.NewInternalGSError(err)
	}
	return gserror.NoError
}

func RemoveUser(db *sql.DB, email string, id snowflake.ID) gserror.GSError {
	removeUserQuery :=
		"DELETE FROM users WHERE email=$1"
	_, err := db.Exec(removeUserQuery, email)
	if err != nil {
		log.Error("UUID: %s; Error removing user with email %s: %s", id, email, err)
		return gserror.NewInternalGSError(err)
	}
	return gserror.NoError
}

func RenewActivationToken(db *sql.DB, userID int64, newActivationToken string, id snowflake.ID) gserror.GSError {
	renewTokenQuery :=
		"UPDATE users SET activationtoken=$1 WHERE id=$2" //#nosec
	res, err := db.Exec(renewTokenQuery, newActivationToken, userID)
	if err != nil {
		log.Error("UUID: %s; Error setting the activation token for user id %d: %s", id, userID, err)
		return gserror.NewInternalGSError(err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		log.Error("UUID: %s; Error retrieving the number of rows for user id %d: %s", id, userID, err)
		return gserror.NewInternalGSError(err)
	}

	if rowsAffected != 1 {
		log.Error("UUID: %s; More than one user with id %d was affected when setting the activation token", id, userID)
		return gserror.NewInternalGSError(errors.New("unexpected number of rows affected"))
	}

	return gserror.NoError
}

func UpdatePassword(db *sql.DB, userId int64, activationToken string, newPassword string, id snowflake.ID) gserror.GSError {
	passHash := auth.ComputePasswordHash(newPassword)

	renewTokenQuery :=
		"UPDATE users SET passhash=$1 WHERE id=$2 and activationToken=$3" //#nosec
	res, err := db.Exec(renewTokenQuery, passHash, userId, activationToken)
	if err != nil {
		log.Error("UUID: %s; Error setting the activation token for user id %d: %s", id, userId, err)
		return gserror.NewInternalGSError(err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		log.Error("UUID: %s; Error retrieving the number of rows for user id %d: %s", id, userId, err)
		return gserror.NewInternalGSError(err)
	}

	if rowsAffected != 1 {
		errorMessage := fmt.Sprintf("More than one user with id %d was affected when trying to update the password", userId)
		log.Error("UUID: %s; %s", id, errorMessage)
		return gserror.NewInternalGSError(errors.New(errorMessage))
	}

	return gserror.NoError
}
