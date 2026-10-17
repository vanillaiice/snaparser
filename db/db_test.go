package db

import "testing"

func TestOpen(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}

	t.Run("close", func(t *testing.T) {
		if err = db.Close(); err != nil {
			t.Error(err)
		}
	})
}
