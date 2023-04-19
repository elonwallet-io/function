package repository

import (
	"encoding/json"
	"fmt"
	"github.com/Leantar/elonwallet-function/models"
	"github.com/Leantar/elonwallet-function/server/common"
	"os"
	"sync"
)

type JsonFile struct {
	rootPath string
	mu       sync.Mutex
}

func NewJsonFile() *JsonFile {
	return &JsonFile{
		rootPath: "/data",
		mu:       sync.Mutex{},
	}
}

func (j *JsonFile) GetUser() (user models.User, err error) {
	path := fmt.Sprintf("%s/user_data.json", j.rootPath)

	err = j.loadData(path, &user)
	return
}

func (j *JsonFile) UpsertUser(user models.User) error {
	path := fmt.Sprintf("%s/user_data.json", j.rootPath)

	return j.saveData(path, &user)
}

func (j *JsonFile) SaveSigningKey(signingKey models.SigningKey) error {
	path := fmt.Sprintf("%s/signing_key.json", j.rootPath)

	return j.saveData(path, &signingKey)
}

func (j *JsonFile) GetSigningKey() (signingKey models.SigningKey, err error) {
	path := fmt.Sprintf("%s/signing_key.json", j.rootPath)

	err = j.loadData(path, &signingKey)
	return
}

func (j *JsonFile) loadData(path string, output any) error {
	j.mu.Lock()
	defer j.mu.Unlock()

	file, err := os.Open(path)
	if os.IsNotExist(err) {
		return common.ErrNotFound
	}
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
	j.mu.Lock()
	defer j.mu.Unlock()

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
	}()

	return json.NewEncoder(file).Encode(data)
}
