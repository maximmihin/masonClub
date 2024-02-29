package storage

import (
	"fmt"
	"testing"
	"time"
)

const dbPath = "/Users/gradagas/Desktop/masonClub/masons.db"

func TestStore_UpdateLastIncome(t *testing.T) {
	db, err := New(dbPath)
	if err != nil {
		t.Errorf("%s\n", err)
	}

	mason, err := db.UpdateLastIncome("morning star", time.Now())
	if err != nil {
		t.Errorf("%s\n", err)
	}
	fmt.Println(&mason)
}
