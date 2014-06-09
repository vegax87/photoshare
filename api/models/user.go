package models

import (
	"code.google.com/p/go.crypto/bcrypt"
	"database/sql"
	"errors"
	"github.com/coopernurse/gorp"
	"log"
	"regexp"
	"time"
)

var MissingLoginFields = errors.New("Missing login fields")

var emailRegex = regexp.MustCompile(".+@.+\\..+")

func validateEmail(email string) bool {
	return emailRegex.Match([]byte(email))
}

type Authenticator struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

func (auth *Authenticator) Identify() (*User, error) {

	if auth.Identifier == "" || auth.Password == "" {
		return nil, MissingLoginFields
	}
	return GetAuthenticatedUser(auth.Identifier, auth.Password)
}

type User struct {
	ID        int64     `db:"id" json:"id"`
	CreatedAt time.Time `db:"created_at" json:"createdAt"`
	Name      string    `db:"name" json:"name"`
	Password  string    `db:"password" json:"password,omitempty"`
	Email     string    `db:"email" json:"email"`
	IsAdmin   bool      `db:"admin" json:"isAdmin"`
	IsActive  bool      `db:"active" json:"isActive"`
}

func (user *User) PreInsert(s gorp.SqlExecutor) error {
	user.CreatedAt = time.Now()
	return nil
}

func (user *User) SetPassword(password string) error {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.Password = string(hashed)
	return nil
}

func (user *User) CheckPassword(password string) bool {
	if user.Password == "" {
		return false
	}
	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	return err == nil
}

func (user *User) Insert() error {
	return dbMap.Insert(user)
}

func (user *User) Validate() (*ValidationResult, error) {

	result := NewValidationResult()

	if user.Name == "" {
		result.Error("name", "Name is missing")
	} else {
		ok, err := user.isNameAvailable()
		if err != nil {
			return result, err
		}
		if !ok {
			result.Error("name", "Name already taken")
		}
	}

	if user.Email == "" {
		result.Error("email", "Email is missing")
	} else if !validateEmail(user.Email) {
		result.Error("email", "Invalid email address")
	} else {
		ok, err := user.isEmailAvailable()
		if err != nil {
			return result, err
		}
		if !ok {
			result.Error("email", "Email already taken")
		}

	}

	if user.Password == "" {
		result.Error("password", "Password is missing")
	}
	log.Println(result.Errors)

	return result, nil
}

func (user *User) isNameAvailable() (bool, error) {
	var (
		num int64
		err error
	)
	q := "SELECT COUNT(id) FROM users WHERE name=$1"
	if user.ID == 0 {
		num, err = dbMap.SelectInt(q, user.Name)
	} else {
		q += " AND id != $2"
		num, err = dbMap.SelectInt(q, user.Name, user.ID)
	}
	if err != nil {
		return false, err
	}
	return num == 0, nil
}

func (user *User) isEmailAvailable() (bool, error) {
	var (
		num int64
		err error
	)
	q := "SELECT COUNT(id) FROM users WHERE email=$1"
	if user.ID == 0 {
		num, err = dbMap.SelectInt(q, user.Email)
	} else {
		q += " AND id != $2"
		num, err = dbMap.SelectInt(q, user.Email, user.ID)
	}
	if err != nil {
		return false, err
	}
	return num == 0, nil
}

func GetActiveUser(userID int) (*User, error) {

	user := &User{}
	if err := dbMap.SelectOne(user, "SELECT * FROM users WHERE active=$1 AND id=$2", true, userID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return user, nil

}

func GetAuthenticatedUser(identifier string, password string) (*User, error) {
	user := &User{}
	if err := dbMap.SelectOne(user, "SELECT * FROM users WHERE active=$1 AND (email=$2 OR name=$2)", true, identifier); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if !user.CheckPassword(password) {
		return nil, nil
	}

	return user, nil
}