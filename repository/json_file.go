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

func (j *JsonFile) GetUser() (models.User, error) {
	path := fmt.Sprintf("%s/user_data.json", j.rootPath)

	var user models.User
	err := j.loadData(path, &user)
	if err != nil {
		return models.User{}, fmt.Errorf("failed to get user data: %w", err)
	}

	return user, nil
}

func (j *JsonFile) UpsertUser(user models.User) error {
	path := fmt.Sprintf("%s/user_data.json", j.rootPath)

	err := j.saveData(path, &user)
	if err != nil {
		return fmt.Errorf("failed to update user data: %w", err)
	}

	return nil
}

func (j *JsonFile) SaveSigningKey(signingKey models.SigningKey) error {
	path := fmt.Sprintf("%s/signing_key.json", j.rootPath)

	err := j.saveData(path, &signingKey)
	if err != nil {
		return fmt.Errorf("failed to save signing key: %w", err)
	}

	return nil
}

func (j *JsonFile) GetSigningKey() (models.SigningKey, error) {
	path := fmt.Sprintf("%s/signing_key.json", j.rootPath)

	var signingKey models.SigningKey
	err := j.loadData(path, &signingKey)
	if err != nil {
		return models.SigningKey{}, fmt.Errorf("failed to get signing key: %w", err)
	}

	return signingKey, nil
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
