package data

import (
	"database/sql"
	"errors"
)

// Define custom errors. We'll return these from our Get() and Update() methods.
var (
	ErrRecordNotFound = errors.New("record not found")
	ErrEditConflict   = errors.New("edit conflict")
)

// Create a Models struct which wraps the BookModel.
type Models struct {
	Books       BookModel
	Permissions PermissionModel
	Tokens      TokenModel
	Users       UserModel
}

// For ease of use, we also add a New() method which returns a Models struct containing
// the initialized BookModel.
func NewModels(db *sql.DB) Models {
	return Models{
		Books:       BookModel{DB: db},
		Permissions: PermissionModel{DB: db},
		Tokens:      TokenModel{DB: db},
		Users:       UserModel{DB: db},
	}
}
