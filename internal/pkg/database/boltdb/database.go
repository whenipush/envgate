package database

import "go.etcd.io/bbolt"

func MustConnect(dbPath string) *bbolt.DB {
	db, err := bbolt.Open(dbPath, 0600, nil)
	if err != nil {
		panic(err)
	}
	return db
}
