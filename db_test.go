package boltdb_go

import (
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
)

func TestDB_Open(t *testing.T) {
	WithDB(func(db *DB, path string) {
		err := db.Open(path, 0666)
		assert.NoError(t, err)
	})
}
func WithDB(fn func(*DB, string)) {
	f, _ := ioutil.TempFile("", "bolt-")
	path := f.Name()
	f.Close()
	//os.Remove(path)
	//defer os.RemoveAll(path)
	db := NewDB()
	fn(db, path)
}
