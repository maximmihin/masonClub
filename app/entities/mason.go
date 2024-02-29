package entities

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

type Mason struct {
	Id        uint
	Pseudonym string
	JwtToken  string
	LastAuth  time.Time
}

func (m Mason) Validate() error {
	problems := make(map[string]string, 0)

	if !utf8.ValidString(m.Pseudonym) {
		problems["pseudonym"] = "псевдоним масона может состоять только из utf8 символов"
	}

	cRunes := utf8.RuneCountInString(m.Pseudonym)
	if cRunes == 0 {
		problems["pseudonym"] = "псевдоним масона должен содержать хотя бы 1 символ"
	} else if cRunes > 256 {
		problems["pseudonym"] = "псевдоним масона может состоять максимум 256 символов"
	}

	if len(problems) > 0 {
		var strProblems strings.Builder

		for k, v := range problems {
			strProblems.WriteString(k + " : " + v + "\n")
		}

		return fmt.Errorf(strProblems.String())
	}

	return nil
}
