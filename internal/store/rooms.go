package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"time"
)

type RoomStore struct {
	db *sql.DB
}

type Room struct {
	Id        int64     `json:"id,omitempty"`
	Name      string    `json:"name"`
	Author    *User     `json:"author"`
	Language  string    `json:"lang"`
	CreatedAt time.Time `json:"created_at"`
	Members   []Member  `json:"members"`
}

type Member struct {
	Id      int64     `json:"id"`
	Name    string    `json:"name"`
	Role    string    `json:"role"`
	AddedAt time.Time `json:"added_at"`
}

type RoomUser struct {
	RoomId int64 `json:"room_id"`
	UserId int64 `json:"user_id"`
	Role   int   `json:"role"`
}

func (r *RoomStore) Create(ctx context.Context, room *Room) error {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeOut)
	defer cancel()

	return withTx(r.db, ctx, func(tx *sql.Tx) error {
		query := `
			INSERT INTO rooms (name, author, language)
			VALUES ($1, $2, $3) RETURNING id, created_at
		`
		err := tx.QueryRowContext(ctx, query,
			room.Name,
			room.Author.Id,
			room.Language,
		).Scan(
			&room.Id,
			&room.CreatedAt,
		)

		if err != nil {
			return err
		}

		query = `
			INSERT INTO room_users (room_id, user_id, roleid)
			VALUES ($1, $2, $3)
		`
		_, err = tx.ExecContext(ctx, query, room.Id, room.Author.Id, 3)
		if err != nil {
			return err
		}

		return nil
	})

}

func (r *RoomStore) GetUserRooms(ctx context.Context, user *User) ([]Room, error) {
	query := `
		SELECT r.id, r.name, r.language, r.author, u.fname, u.lname
		FROM room_users ru
		JOIN rooms r ON r.id = ru.room_id
		JOIN users u ON u.id = r.author
		WHERE ru.user_id = $1
	`
	var output []Room
	rows, err := r.db.QueryContext(ctx, query, user.Id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var ro Room
		ro.Author = &User{}
		err := rows.Scan(&ro.Id, &ro.Name, &ro.Language,
			&ro.Author.Id, &ro.Author.FirstName, &ro.Author.LastName,
		)
		if err != nil {
			return nil, err
		}
		output = append(output, ro)
	}

	return output, nil

}

func (r *RoomStore) GetRoomById(ctx context.Context, roomID int64) (*Room, error) {
	query := `
		SELECT u.fname, u.lname, r.name, r.language
		FROM rooms r
		JOIN users u ON u.id = r.author
		WHERE r.id = $1
	`
	roomresp := Room{}
	roomresp.Author = &User{}
	err := r.db.QueryRowContext(ctx, query, roomID).Scan(
		&roomresp.Author.FirstName,
		&roomresp.Author.LastName,
		&roomresp.Name,
		&roomresp.Language,
	)
	if err != nil {
		return nil, err
	}

	members, err := r.getMembers(ctx, roomID)
	if err != nil {
		return nil, err
	}
	roomresp.Members = members

	return &roomresp, nil
}

func (r *RoomStore) getMembers(ctx context.Context, roomID int64) ([]Member, error) {
	var members []Member
	query := `
		SELECT u.id, u.fname, roles.name, added_at
		FROM room_users
		JOIN users u ON u.id = user_id
		LEFT JOIN roles ON roles.roleid = room_users.roleid
		WHERE room_id = $1 AND u.active = TRUE
	`

	rows, err := r.db.QueryContext(ctx, query, roomID)
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			return members, nil
		default:
			return nil, err
		}
	}
	defer rows.Close()
	for rows.Next() {
		var m Member
		err := rows.Scan(&m.Id, &m.Name, &m.Role, &m.AddedAt)
		if err != nil {
			return nil, err
		}
		members = append(members, m)
	}

	return members, nil

}

func (r *RoomStore) AddMember(ctx context.Context, tx *sql.Tx, roomID, userID int64, roleid int64) error {
	query := `
		INSERT INTO room_users (room_id, user_id, roleid)
		VALUES ($1, $2, $3)
	`

	_, err := r.db.ExecContext(ctx, query, roomID, userID, roleid)
	if err != nil {
		switch {
		case err.Error() == `pq: insert or update on table "room_users" violates foreign key constraint "room_users_roleid_fkey"`:
			return ErrInvalidRole
		default:
			return err
		}
	}

	return nil
}

func (r *RoomStore) CreateNewJoinToken(ctx context.Context,
	exp time.Duration, roomid, userid, roleid int64, token string) error {
	return withTx(r.db, ctx, func(tx *sql.Tx) error {
		query := `
			INSERT INTO join_tokens (room_id, userid, roleid, token, expiry)
			VALUES($1, $2, $3, $4, $5)
		`
		ctx, cancel := context.WithTimeout(ctx, QueryTimeOut)
		defer cancel()

		_, err := tx.ExecContext(ctx, query, roomid, userid, roleid, token, time.Now().Add(exp))
		if err != nil {
			return err
		}

		return nil
	})
}

func (r *RoomStore) authorise(ctx context.Context, tx *sql.Tx, token string,
	expiry time.Time) (*RoomUser, error) {
	query := `
		SELECT jt.user_id, jt.room_id, jt.role 
		FROM join_tokens jt
		JOIN users u ON u.id = jt.user_id
		WHERE jt.token = $1 AND jt.expiry > $2 AND u.active = true
	`
	room_user := &RoomUser{}
	hashtoken := sha256.Sum256([]byte(token))
	hash := hex.EncodeToString(hashtoken[:])
	err := tx.QueryRowContext(ctx, query, hash, expiry).Scan(
		&room_user.UserId,
		&room_user.RoomId,
		&room_user.Role,
	)
	if err != nil {
		return nil, err
	}
	return room_user, nil
}

func (r *RoomStore) AcceptJoinRequest(ctx context.Context, token string, expiry time.Time) error {
	return withTx(r.db, ctx, func(tx *sql.Tx) error {

		ctx, cancel := context.WithTimeout(ctx, QueryTimeOut)
		defer cancel()
		room_user, err := r.authorise(ctx, tx, token, expiry)
		if err != nil {
			switch err {
			case sql.ErrNoRows:
				return ErrTokenExpired
			default:
				return err
			}
		}

		err = r.AddMember(ctx, tx, room_user.RoomId,
			room_user.UserId, int64(room_user.Role))
		if err != nil {
			return err
		}

		query := `
			DELETE FROM join_tokens
			WHERE user_id = $1 AND room_id = $2
		`
		_, err = tx.ExecContext(ctx, query, room_user.UserId, room_user.RoomId)
		if err != nil {
			return err
		}

		return nil
	})
}
