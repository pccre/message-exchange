package storage

import "errors"

type Storage interface {
  Load()
  GetRelationships(string) ([]interface{}, error)
  AddRelationship(string, interface{}) error
}

var ErrNotFound = errors.New("user not found")
