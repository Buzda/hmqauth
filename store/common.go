package store

import (
	"authserver/config"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"strings"
	"sync"

	pgx "github.com/jackc/pgx/v4/pgxpool"
)

// User Persistence manages the cache and updates to
type UserPersistence interface {
	Load() error
	Login(username string, password string, requesttoken bool) (User, error)
	AddUser(user User) error
	EditUser(user User) error
	DeleteUser(username string) error
	GetUserByToken(token string) (User, error)
	GetUserByUsername(username string) (User, error)
	GetUsers() []User
	AddTopicToUser(username string, topic Topic) error
	EditTopicForUser(username string, topic Topic) error
	DeleteTopicFromUser(username string, topicString string) error
}

func NewStorage(storageType string) UserPersistence {
	switch storageType {
	case "json":
		return InitJSON(config.Config.GetStorageFileName())
	case "postgres":
		return InitPostgres(config.Config.GetConnString())
	default:
		return InitJSON(config.Config.GetStorageFileName())
	}
}

type UserJSONCollection struct {
	Users []User
	sync.RWMutex
	Fname string
}

var UsersJSON UserJSONCollection

type UserPostgresCollection struct {
	Users []User
	sync.RWMutex
	DB    *pgx.Pool
	DBerr error
}

var UsersPostgres UserPostgresCollection

type User struct {
	UserName string     `json:"username"`
	Password string     `json:"password"`
	Admin    bool       `json:"admin"`
	CreateTS string     `json:"createTS"`
	UpdateTS string     `json:"updateTS"`
	Token    string     `json:"token"`
	Topics   TopicArray `json:"topics"`
}

type Topic struct {
	TopicString string `json:"topicstring"`
	Pub         bool   `json:"pub"`
	Sub         bool   `json:"sub"`
}

type TopicArray []Topic

// Scan implements the sql.Scanner interface
func (me *TopicArray) Scan(value interface{}) error {
	var i sql.NullString
	if err := i.Scan(value); err != nil {
		return err
	}
	if i.Valid == false {
		return nil
	}
	return json.Unmarshal([]byte(i.String), &me)
}

// Value implements the driver.Valuer interface
func (me TopicArray) Value() (driver.Value, error) {
	var nullString sql.NullString
	if me != nil {
		addressBytes, err := json.Marshal(me)
		if err != nil {
			return nil, err
		}
		nullString.Valid = true
		nullString.String = string(addressBytes)
	}
	return nullString.Value()
}

// CheckTopicAuthSub checks to see whether the user has Sub rights on a topic
func (me User) CheckTopicAuthSub(topic string) (bool, error) {
	_, sub, err := me.CheckTopicAuth(topic)
	return sub, err
}

// CheckTopicAuthPub checks to see whether the user has Pub rights on a topic
func (me User) CheckTopicAuthPub(topic string) (bool, error) {
	pub, _, err := me.CheckTopicAuth(topic)
	return pub, err
}

// CheckTopicAuth returns 2 boolean values, one showing whether the user has pub rights on a topic, the second
// showing whether the user has sub rights on the topic
func (me User) CheckTopicAuth(topic string) (pub bool, sub bool, err error) {
	pub = false
	sub = false
	matched := false
	for _, v := range me.Topics {
		if topicMatch(topic, v.TopicString) == true {
			matched = true
			if v.Pub == true {
				pub = true
			}
			if v.Sub == true {
				sub = true
			}
		}
	}
	if !matched {
		err = errors.New("Topic not found")
	}
	return
}

// topicMatch compares two topics, and returns a true if they are related (one is part of the other)
func topicMatch(SetStoreHandler string, permittedTopic string) bool {
	// For safety we remove any trailing forward slash - as this isn't
	// actually valid as a topic but is a common user error in checking or setting permissions
	if string(SetStoreHandler[len(SetStoreHandler)-1]) == "/" {
		SetStoreHandler = SetStoreHandler[:len(SetStoreHandler)-1]
	}
	if string(permittedTopic[len(permittedTopic)-1]) == "/" {
		permittedTopic = permittedTopic[:len(permittedTopic)-1]
	}
	if SetStoreHandler == permittedTopic || permittedTopic == "#" {
		return true
	}

	topicComponents := strings.Split(SetStoreHandler, "/")
	filterComponents := strings.Split(permittedTopic, "/")

	currentpos := 0
	filterComponentsLength := len(filterComponents)
	currentFilterComponent := ""
	if filterComponentsLength > 0 {
		currentFilterComponent = filterComponents[currentpos]
	}
	for _, topicVal := range topicComponents {

		if currentFilterComponent == "" {
			return false
		}
		if currentFilterComponent == "#" {
			return true
		}
		if currentFilterComponent != "+" && currentFilterComponent != topicVal {
			return false
		}
		currentpos++
		if filterComponentsLength > currentpos {
			currentFilterComponent = filterComponents[currentpos]
		} else {
			currentFilterComponent = ""
		}
	}

	// If we have got here but we've still not parsed the full length of the permission then
	// its deffo not a match - Is there a rule we can apply up front that says
	// "if the length of the permission is greater than the check topic then we're deffo not matching"
	if len(filterComponents) > currentpos {
		return false
	}
	return true
}
