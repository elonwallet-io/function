package datastore

import (
	"encoding/json"
	"fmt"
	"github.com/Leantar/elonwallet-function/models"
	"os"
	"sync"
)

type JsonFile struct {
	rootPath string
	mut      sync.Mutex
}

func NewJsonFile(rootPath string) *JsonFile {
	return &JsonFile{
		rootPath: rootPath,
	}
}

func (j *JsonFile) GetUser() (user models.User, err error) {
	path := fmt.Sprintf("%s/user_data.json", j.rootPath)

	err = j.loadData(path, &user)
	return
}

func (j *JsonFile) SaveUser(user models.User) error {
	path := fmt.Sprintf("%s/user_data.json", j.rootPath)

	return j.saveData(path, &user)
}

func (j *JsonFile) loadData(path string, output any) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
	}()

	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()

	return decoder.Decode(output)
}

func (j *JsonFile) saveData(path string, data any) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
	}()

	return json.NewEncoder(file).Encode(data)
}
