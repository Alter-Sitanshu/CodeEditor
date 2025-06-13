package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var ErrNotFound error = errors.New("resource not found")

type UserStore struct {
	db *sql.DB
}

type User struct {
	Id        int64    `json:"id,omitempty"`
	FirstName string   `json:"first_name"`
	LastName  string   `json:"last_name,omitempty"`
	Password  password `json:"-"`
	Email     string   `json:"email,omitempty"`
	Age       int      `json:"age,omitempty"`
	Active    bool     `json:"active,omitempty"`
}

type password struct {
	text *string
	hash []byte
}

type UserFromToken struct {
	id        int64
	FirstName string
	LastName  string
	email     string
	active    bool
}

func (pass *password) Encrypt(text string) error {
	pass.text = &text
	hashed_pass, err := bcrypt.GenerateFromPassword([]byte(text), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	pass.hash = hashed_pass
	return nil
}

func (pass *password) Verify(text string) bool {
	err := bcrypt.CompareHashAndPassword(pass.hash, []byte(text))
	return err == nil
}

func (u *UserStore) GetUserById(ctx context.Context, id int64) (*User, error) {
	query := `
		SELECT id, fname, lname, email, age, active
		FROM users
		WHERE id = $1
	`
	var user User
	err := u.db.QueryRowContext(ctx, query, id).Scan(
		&user.Id,
		&user.FirstName,
		&user.LastName,
		&user.Email,
		&user.Age,
		&user.Active,
	)
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			return nil, ErrNotFound
		default:
			return nil, err
		}
	}

	return &user, nil
}

func (u *UserStore) create(ctx context.Context, tx *sql.Tx, user *User) error {
	query := `
		INSERT INTO users (fname, lname, password, email, age)
		VALUES($1, $2, $3, $4, $5) RETURNING id
	`
	ctx, cancel := context.WithTimeout(ctx, QueryTimeOut)
	defer cancel()

	err := tx.QueryRowContext(ctx, query,
		user.FirstName,
		user.LastName,
		user.Password.hash,
		user.Email,
		user.Age,
	).Scan(
		&user.Id,
	)
	if err != nil {
		switch {
		case err.Error() == `pq: duplicate key value violates unique constraint "users_email_key"`:
			return ErrDupliMail
		}
		return err
	}
	return nil
}

func (u *UserStore) CreateAndInvite(ctx context.Context, user *User,
	token string, expiry time.Duration) error {
	return withTx(u.db, ctx, func(tx *sql.Tx) error {
		err := u.create(ctx, tx, user)
		if err != nil {
			return err
		}
		err = CreateNewUserToken(ctx, tx, expiry, user.Id, token)
		if err != nil {
			return err
		}

		return nil
	})
}

func CreateNewUserToken(ctx context.Context, tx *sql.Tx,
	exp time.Duration, userid int64, token string) error {

	query := `
		INSERT INTO user_tokens (userid, token, expiry)
		VALUES($1, $2, $3)
	`
	ctx, cancel := context.WithTimeout(ctx, QueryTimeOut)
	defer cancel()

	_, err := tx.ExecContext(ctx, query, userid, token, time.Now().Add(exp))
	if err != nil {
		return err
	}

	return nil
}
func (u *UserStore) authorise(ctx context.Context, tx *sql.Tx, token string,
	expiry time.Time) (*UserFromToken, error) {
	query := `
		SELECT u.id, u.fname, u.lname, u.email, u.active
		FROM users u
		JOIN user_tokens ut ON ut.userid = u.id
		WHERE ut.token = $1 AND ut.expiry > $2 AND u.active = false
	`
	user := &UserFromToken{}
	hashtoken := sha256.Sum256([]byte(token))
	hash := hex.EncodeToString(hashtoken[:])
	err := tx.QueryRowContext(ctx, query, hash, expiry).Scan(
		&user.id,
		&user.FirstName,
		&user.LastName,
		&user.email,
		&user.active,
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (u *UserStore) ActivateUser(ctx context.Context, token string, expiry time.Time) error {
	return withTx(u.db, ctx, func(tx *sql.Tx) error {

		ctx, cancel := context.WithTimeout(ctx, QueryTimeOut)
		defer cancel()

		user, err := u.authorise(ctx, tx, token, expiry)
		if err != nil {
			switch err {
			case sql.ErrNoRows:
				return ErrTokenExpired
			default:
				return err
			}
		}

		query := `
			UPDATE users
			SET active = true
			WHERE id = $1
		`
		_, err = tx.ExecContext(ctx, query, user.id)
		if err != nil {
			return err
		}

		query = `
			DELETE FROM user_tokens
			WHERE userid = $1
		`
		_, err = tx.ExecContext(ctx, query, user.id)
		if err != nil {
			return err
		}

		return nil
	})
}

func (u *UserStore) DeleteUser(ctx context.Context, user *User) error {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeOut)
	defer cancel()

	return withTx(u.db, ctx, func(tx *sql.Tx) error {
		// The role check will be handled in the middleware
		query := `
			DELETE FROM users
			WHERE id = $1
		`
		_, err := tx.ExecContext(ctx, query, user.Id)
		if err != nil {
			return err
		}

		return nil
	})
}
