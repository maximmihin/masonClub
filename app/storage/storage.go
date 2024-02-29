package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/mattn/go-sqlite3"
	_ "github.com/mattn/go-sqlite3"
	"masonClub/app/entities"
	"time"
)

var (
	ErrMasonNotExist     = errors.New("такой масон не зарегистрирован")
	ErrMasonAlreadyExist = errors.New("такой масон уже зарегистрирован")
)

type Store struct {
	*sql.DB
}

func (s *Store) Close() {
	_ = s.DB.Close()
}

func New(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return &Store{}, fmt.Errorf("не получтлось открыть базу %s\n", err)
	}

	err = db.Ping()
	if err != nil {
		return &Store{}, fmt.Errorf("не получтлось пингануть базу %s\n", err)
	}

	return &Store{
		db,
	}, nil
}

func (s *Store) GetMasonByPseudonym(pseudonym string) (*entities.Mason, error) {
	stmt := "SELECT id, Pseudonym, JWY_token, Last_auth FROM Masons WHERE Pseudonym = ?"
	rows, err := s.DB.Query(stmt, pseudonym)
	if err != nil {
		return nil, fmt.Errorf("ошибка при выполнении запроса: %s", err)
	}
	defer rows.Close()

	var count uint
	mason := &entities.Mason{}
	for rows.Next() {
		count++

		err = rows.Scan(&mason.Id, &mason.Pseudonym, &mason.JwtToken, &mason.LastAuth)
		if err != nil {
			return nil, fmt.Errorf("ошибка при сканировании строк : %s", err)
		}
	}

	if count == 0 {
		return nil, ErrMasonNotExist
	}

	if count > 1 {
		return mason, fmt.Errorf("внутренняя ошибка: больше одного пользователя с таким ником в базе %s", err)
	}

	return mason, nil
}

func (s *Store) NewMason(mason *entities.Mason) (uint, error) {
	stmt := "INSERT INTO Masons (Pseudonym, JWY_token, Last_auth) VALUES (?, ?, ?);"

	res, err := s.DB.Exec(stmt, mason.Pseudonym, mason.JwtToken, mason.LastAuth)
	var sqlErr sqlite3.Error
	if errors.As(err, &sqlErr) {
		if sqlErr.Code == sqlite3.ErrConstraint && sqlErr.ExtendedCode == 2067 { // (2067) SQLITE_CONSTRAINT_UNIQUE https://www.sqlite.org/rescode.html#constraint_unique
			return 0, ErrMasonAlreadyExist
		}
		return 0, err
	}

	tagId, _ := res.LastInsertId()
	return uint(tagId), nil
}

func (s *Store) GetAllMasons() ([]entities.Mason, error) {
	stmt := "SELECT ID, Pseudonym, JWY_token, Last_auth FROM Masons"

	rows, err := s.DB.Query(stmt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mason entities.Mason
	var masons []entities.Mason

	for rows.Next() {
		err = rows.Scan(&mason.Id, &mason.Pseudonym, &mason.JwtToken, &mason.LastAuth)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, fmt.Errorf("не ни одного масона :(")
			}
			return nil, err
		}
		masons = append(masons, mason)
	}

	if len(masons) == 0 {
		return nil, fmt.Errorf("не ни одного масона :(")
	}
	return masons, nil
}

func (s *Store) UpdateLastIncome(pseudonym string, updateTime time.Time) (uint, error) {

	stmt := "UPDATE Masons SET Last_auth = ? WHERE Pseudonym = ?"

	res, err := s.DB.Exec(stmt, updateTime, pseudonym)
	if err != nil {
		return 0, err
	}

	masonId, _ := res.LastInsertId()
	return uint(masonId), nil
}
