package models

import "errors"

var (
	ErrNotFound           = errors.New("not found")
	ErrEmailExists        = errors.New("email already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
)
