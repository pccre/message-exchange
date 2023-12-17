package storage

import (
	"os"
	"sync"

	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigFastest

type LocalStorage struct {
  Filename string
  Items    map[string][]interface{}
  Mut *sync.RWMutex
}

func (s *LocalStorage) Load() {
  s.Mut = &sync.RWMutex{}
  data, err := os.ReadFile(s.Filename)
  if err != nil {
    panic("Error while loading LocalStorage; " + err.Error())
  }

  err = json.Unmarshal(data, &s.Items)
  if err != nil {
    panic("Error while loading LocalStorage; " + err.Error())
  }
}

func (s *LocalStorage) GetRelationships(userhash string) (data []interface{}, err error) {
  s.Mut.RLock()
  data, ok := s.Items[userhash]
  s.Mut.RUnlock()
  if !ok {
    err = ErrNotFound
  }
  return
}

func (s *LocalStorage) Save() error {
  s.Mut.RLock()
  data, err := json.Marshal(s.Items)
  s.Mut.RUnlock()
  if err != nil {
    return err
  }
  return os.WriteFile(s.Filename, data, 0644)
}

func (s *LocalStorage) AddRelationship(userhash string, data interface{}) error {
  s.Mut.Lock()
  s.Items[userhash] = append(s.Items[userhash], data)
  s.Mut.Unlock()
  return s.Save()
}
