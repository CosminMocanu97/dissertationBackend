package database

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/CosminMocanu97/dissertationBackend/internal/auth"
	"github.com/CosminMocanu97/dissertationBackend/internal/types"
	"github.com/CosminMocanu97/dissertationBackend/pkg/log"
)

const (
	ERROR_USER_ALREADY_EXISTS  = "the email already exists in the database"
	ERROR_USER_NOT_ACTIVATED = 	"account is not activated"
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
func AddNewUser(db *sql.DB, registrationData types.RegistrationData, activationToken string) (bool, error) {
	userExists, err := UserExists(db, registrationData.Email)

	if err != nil {
		log.Error("Error verifying if the user with email %s exists: %s", registrationData.Email, err)
		return false, err
	} else {
		// if the email is not already in the database, try to add it
		if !userExists {
			passHash := auth.ComputePasswordHash(registrationData.Password)
			addUserStatement :=
				"INSERT INTO users(email, passHash, activationToken) VALUES($1, $2, $3);"

			_, err := db.Exec(addUserStatement, registrationData.Email, passHash, activationToken)
			if err != nil {
				log.Error("Error adding a new user: ", err)
				return false, err
			}

			log.Info("Successfully added the user with email %s", registrationData.Email)
			return true, nil
		} else {
			err = errors.New(ERROR_USER_ALREADY_EXISTS)
			return false, err
		}
	}
}
func UserExists(db *sql.DB, email string) (bool, error) {
	userExistsQuery :=
		"SELECT * FROM users WHERE email=$1;"
	res, err := db.Exec(userExistsQuery, email)
	if err != nil {
		log.Error("Error checking if the user with email %s exists: %s", email, err)
		return false, err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		log.Error("Error retrieving the number of rows matching the query %s, for email %s: %s", userExistsQuery, email, err)
		return false, err
	}

	if rowsAffected > 0 {
		return true, nil
	} else {
		return false, nil
	}
}

func GetActivationTokenForEmail(db *sql.DB, email string) (string, error) {
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
		return "", err
	}

	log.Info("Successfully retrieved activation token for email %s", email)
	return activationToken, nil
}

func GetUserDetailsForEmail(db *sql.DB, email string) (types.User, error) {
	var user types.User
	user.Email = email
	getUserIdForEmailQuery :=
		"SELECT id, passhash, isActivated, activationToken FROM users WHERE email=$1"

	row := db.QueryRow(getUserIdForEmailQuery, email)
	err := row.Scan(&user.ID, &user.Passhash, &user.IsActivated, &user.ActivationToken)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Error("Error finding any entry for the email %s: %s", email, err)
		} else {
			log.Error("Error getting the user details for email %s: %s", email, err)
		}
		return user, err
	}

	log.Info("Successfully retrieved the user details for email %s", email)
	return user, nil
}

// VerifyLoginCredentials returns true if there as a user with the email and password provided as parameter, false otherwise
func VerifyLoginCredentials(db *sql.DB, email, password string) (bool, error) {
	user, err := GetUserDetailsForEmail(db, email)
	if err != nil {
		log.Error("Error retrieving user details for email %s; %s", email, err)
		return false, err
	}

	// verify if the password matches
	passHash := auth.ComputePasswordHash(password)
	if user.Passhash == passHash {
		log.Info("The user %s has successfully logged in", user.Email)
		return true, nil
	} else {
		log.Info("The user %s tried to log in but the credentials are incorrect", user.Email)
		err = errors.New("invalid credentials")
		return false, err
	}
}

func VerifyActivationToken(db *sql.DB, userId int64, activationToken string) (bool, error) {
	// skip security check for the activationToken
	expectedActivationTokenQuery := "SELECT activationToken FROM users WHERE id=$1" //#nosec
	row := db.QueryRow(expectedActivationTokenQuery, userId)

	var expectedActivationToken string
	err := row.Scan(&expectedActivationToken)

	if err != nil {
		log.Error("Error retrieving the activation token for user with ID %d: %s", userId, err)
		return false, err
	}

	if expectedActivationToken == activationToken {
		log.Info("The provided activation token for the account with ID %d is correct", userId)
		return true, nil
	}
	return false, nil
}

func UserIsActivated(db *sql.DB, email string) (bool, error) {
	userIsVerifiedQuery := "SELECT isactivated FROM users WHERE email=$1"
	var isActivated bool
	row := db.QueryRow(userIsVerifiedQuery, email)
	err := row.Scan(&isActivated)
	if err != nil {
		log.Error("There's no account with the email %s: %s", email, err)
		return false, err
	}
	if !isActivated {
		err = errors.New(ERROR_USER_NOT_ACTIVATED)
		log.Error("The account %s is not activated: %s", email, err)
		return false, err
	}
	return true, nil
}

func ActivateAccount(db *sql.DB, userID int64) error {
	activateAccountQuery :=
		"UPDATE users SET isactivated=true WHERE id=$1"
	_, err := db.Exec(activateAccountQuery, userID)
	if err != nil {
		log.Error("Error activating the account for ID %d: %s", userID, err)
		return err
	}
	return nil
}

func RemoveUser(db *sql.DB, email string) error {
	removeUserQuery :=
		"DELETE FROM users WHERE email=$1"
	_, err := db.Exec(removeUserQuery, email)
	if err != nil {
		log.Error("Error removing user with email %s: %s", email, err)
		return err
	}
	return nil
}

func RenewActivationToken(db *sql.DB, userID int64, newActivationToken string) error {
	renewTokenQuery :=
		"UPDATE users SET activationtoken=$1 WHERE id=$2" //#nosec
	res, err := db.Exec(renewTokenQuery, newActivationToken, userID)
	if err != nil {
		log.Error("Error setting the activation token for user id %d: %s", userID, err)
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		log.Error("Error retrieving the number of rows for user id %d: %s", userID, err)
		return err
	}

	if rowsAffected != 1 {
		log.Error("More than one user with id %d was affected when setting the activation token", userID)
		return err
	}

	return nil
}

func UpdatePassword(db *sql.DB, userId int64, activationToken string, newPassword string) error {
	passHash := auth.ComputePasswordHash(newPassword)

	renewTokenQuery :=
		"UPDATE users SET passhash=$1 WHERE id=$2 and activationToken=$3" //#nosec
	res, err := db.Exec(renewTokenQuery, passHash, userId, activationToken)
	if err != nil {
		log.Error("Error setting the activation token for user id %d: %s", userId, err)
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		log.Error("Error retrieving the number of rows for user id %d: %s", userId, err)
		return err
	}

	if rowsAffected != 1 {
		errorMessage := fmt.Sprintf("More than one user with id %d was affected when trying to update the password", userId)
		log.Error("%s", errorMessage)
		return err
	}

	return nil
}
