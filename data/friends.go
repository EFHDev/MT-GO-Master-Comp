package data

import (
	"fmt"
	"log"
	"mtgo/tools"
	"path/filepath"

	"github.com/goccy/go-json"
)

const (
	friendsNotExist string = "Friends for %s do not exist"
	friendsNotSaved string = "Friends for %s was not saved: %s"
)

func setFriends(path string) *Friends {
	output := new(Friends)
	data := tools.GetJSONRawMessage(path)
	if err := json.Unmarshal(data, &output); err != nil {
		log.Println(err)
	}

	return output
}

func (friends *Friends) CreateFriends() *Friends {
	return &Friends{
		Friends:      make([]FriendRequest, 0),
		Ignore:       make([]string, 0),
		InIgnoreList: make([]string, 0),
		Matching: Matching{
			LookingForGroup: false,
		},
		FriendRequestInbox:  make([]any, 0),
		FriendRequestOutbox: make([]any, 0),
	}
}

func GetFriendsByID(uid string) (*Friends, error) {
	profile, err := GetProfileByUID(uid)
	if err != nil {
		log.Println(err)
	}

	if profile.Friends != nil {
		return profile.Friends, nil
	}
	return nil, fmt.Errorf(friendsNotExist, uid)
}

func (friends *Friends) SaveFriends(sessionID string) error {
	friendsFilePath := filepath.Join(profilesPath, sessionID, "friends.json")

	if err := tools.WriteToFile(friendsFilePath, friends); err != nil {
		return fmt.Errorf(friendsNotSaved, sessionID, err)
	}
	log.Println("Friends saved")
	return nil
}
