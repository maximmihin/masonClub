package use_cases

import (
	"errors"
	"github.com/golang-jwt/jwt/v5"
	"masonClub/app/entities"
	"masonClub/app/storage"
	"time"
)

type Validator interface {
	Valid() error
}

var (
	ErrMasonAlreadyInitiation = errors.New("такой масон уже инициирован")
)

func InitiationIntoTheMasons(store *storage.Store, jwtSecret, pseudonym string, firstAuth time.Time) (*entities.Mason, error) {

	bearerToken, err := createJwtToken(jwtSecret, pseudonym, firstAuth)
	if err != nil {
		return nil, err
	}

	mason := &entities.Mason{
		Pseudonym: pseudonym,
		JwtToken:  bearerToken,
		LastAuth:  firstAuth,
	}
	err = mason.Validate()
	if err != nil {
		return nil, err
	}

	mason.Id, err = store.NewMason(mason)
	if err != nil {
		if errors.Is(err, storage.ErrMasonAlreadyExist) {
			return mason, ErrMasonAlreadyInitiation
		}
		return mason, err
	}

	return mason, nil
}

func createJwtToken(jwtSecret, pseudonym string, firstAuth time.Time) (string, error) {
	jwtKey := []byte(jwtSecret)

	payload := jwt.MapClaims{
		"sub": pseudonym,
		"iat": firstAuth.Unix(),
	}

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)

	bearerToken, err := jwtToken.SignedString(jwtKey)
	if err != nil {
		return "", err
	}
	return bearerToken, nil
}

func GetListMasons(store *storage.Store) ([]string, error) {
	masons, err := store.GetAllMasons()
	if err != nil {
		return nil, err
	}

	masonsList := make([]string, 0, len(masons))
	for _, mason := range masons {
		masonsList = append(masonsList, mason.Pseudonym)
	}

	return masonsList, nil
}
