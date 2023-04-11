package common

import (
	"errors"
	"github.com/Leantar/elonwallet-function/models"
)

var (
	ErrNotFound = errors.New("element does not exist")
)

type Repository interface {
	GetUser() (models.User, error)
	UpsertUser(u models.User) error
}
