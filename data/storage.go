package data

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/goccy/go-json"

	"mtgo/tools"
)

func setStorage(path string) *Storage {
	output := new(Storage)

	data := tools.GetJSONRawMessage(path)
	if err := json.UnmarshalNoEscape(data, output); err != nil {
		msg := tools.CheckParsingError(data, err)
		log.Fatalln(msg)
	}

	return output
}

func GetStorageByID(uid string) (*Storage, error) {
	profile, err := GetProfileByUID(uid)
	if err != nil {
		return nil, err
	}

	if profile.Storage != nil {
		return profile.Storage, nil
	}

	return nil, fmt.Errorf(storageNotExist, uid)
}

func (storage *Storage) CreateStorage() *Storage {
	return &Storage{
		Suites: make([]string, 0),
		Builds: &Builds{
			EquipmentBuilds: make([]*EquipmentBuild, 0),
			WeaponBuilds:    make([]*WeaponBuild, 0),
			MagazineBuilds:  make([]*struct{}, 0),
		},
		Insurance: make([]any, 0),
		Mailbox:   make([]*Notification, 0),
	}
}

func (storage *Storage) SaveStorage(sessionID string) error {
	storageFilePath := filepath.Join(profilesPath, sessionID, "storage.json")

	if err := tools.WriteToFile(storageFilePath, storage); err != nil {
		return fmt.Errorf(storageNotSaved, sessionID, err)
	}
	log.Println("Storage saved")
	return nil
}

const (
	storageNotSaved string = "Account for %s was not saved: %s"
	storageNotExist string = "Storage for UID %s does not exist"
)
