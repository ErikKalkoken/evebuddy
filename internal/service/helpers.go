package service

import "github.com/mattn/go-sqlite3"

func isSqlite3ErrConstraint(err error) bool {
	switch e := err.(type) {
	case sqlite3.Error:
		if e.Code == sqlite3.ErrConstraint {
			return true
		}
	}
	return false
}
