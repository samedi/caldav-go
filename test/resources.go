package test

import (
	"os"

	"github.com/ngradwohl/caldav-go/data"
	"github.com/ngradwohl/caldav-go/global"
)

// Creates a fake storage to be used in unit tests.
// TODO: for now it's just creating a storage based  of the default file storage.
// Would be better having an in-memory storage instead and make use of a stubbed fake storage to make unit tests faster.
func NewFakeStorage() FakeStorage {
	return FakeStorage{global.Storage}
}

type FakeStorage struct {
	data.Storage
}

func (s FakeStorage) AddFakeResource(collection, name, data string) {
	pwd, _ := os.Getwd()
	err := os.MkdirAll(pwd+collection, os.ModePerm)
	panicerr(err)
	f, err := os.Create(pwd + collection + name)
	panicerr(err)
	f.WriteString(data)
}
