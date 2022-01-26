package store

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"

	"github.com/rs/xid"
	"golang.org/x/crypto/bcrypt"
)

// InitJSON returns the store object that uses a json file
func InitJSON(fname string) *UserJSONCollection {
	log.Println("Storage type is json")
	UsersJSON.Fname = fname
	return &UsersJSON
}

// Load loads the users along with their topics from a json file
func (me *UserJSONCollection) Load() error {

	fname := me.Fname
	if fname == "" {
		fname = "assets/users.json"
	}

	content, err := ioutil.ReadFile(fname)
	if err != nil {
		log.Println("Error with reading: ", err)
		return err
	}
	var jsonUsers []User
	userMarshalError := json.Unmarshal(content, &jsonUsers)
	if userMarshalError != nil {
		log.Println("Error with unmarshalling: ", userMarshalError.Error())
	}

	me.Lock()
	me.Users = jsonUsers
	me.Fname = fname
	me.Unlock()
	return nil
}

// Login logs the user in and generates a token for the session if required
func (me *UserJSONCollection) Login(username string, password string, requesttoken bool) (User, error) {

	userLoggingIn, getUserError := me.GetUserByUsername(username)
	if getUserError != nil {
		return userLoggingIn, errors.New("User not found")
	}

	PasswordValid := bcrypt.CompareHashAndPassword([]byte(userLoggingIn.Password), []byte(password))
	if PasswordValid != nil {
		var blankUser User
		return blankUser, errors.New("Passwords don't match")
	}

	if requesttoken == true {
		// Create a token for the session
		token := xid.New().String()
		userLoggingIn.Token = token
		tokenUpdateError := me.UpdateUserToken(username, token)
		if tokenUpdateError != nil {
			var blankUser User
			return blankUser, tokenUpdateError
		}
	}
	return userLoggingIn, nil
}

// AddUser adds a new user to the collection
func (me *UserJSONCollection) AddUser(user User) error {
	// Validate the user
	// if the username and/or the password are blank then reject
	if user.UserName == "" || user.Password == "" {
		return errors.New("Username and password must both be non-blank")
	}
	// Add the User to the collection
	me.Lock()

	for _, v := range me.Users {
		if v.UserName == user.UserName {
			me.Unlock()
			return errors.New("User already exists")
		}
	}

	hashPWD, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Println("Cannot create password hash")
		return errors.New("Cannot create password hash")
	}

	user.Password = string(hashPWD)
	me.Users = append(me.Users, user)
	me.Unlock()

	//Save the collection
	return me.Save("")
}

// EditUser edits an existing user
func (me *UserJSONCollection) EditUser(user User) error {
	// Validate the user
	// if the username is blank then reject
	if user.UserName == "" {
		return errors.New("Username and password must both be non-blank")
	}

	hashPWD, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Println("Cannot create password hash")
		return errors.New("Cannot create password hash")
	}

	if user.Password != "" {
		user.Password = string(hashPWD)
	}
	//Add the User to the collection
	me.Lock()
	found := false
	foundindex := 0
	for k, v := range me.Users {
		if v.UserName == user.UserName {
			found = true
			foundindex = k
			break
		}
	}
	me.Unlock()
	if !found {
		return errors.New("User not found")
	}

	me.Lock()
	me.Users[foundindex].Admin = user.Admin
	if user.Password != "" {
		me.Users[foundindex].Password = user.Password
	}

	me.Unlock()
	//Save the collection
	return me.Save("")
}

// UpdateUser accepts a user object and updates the relevant user
// users cannot change their name - so we can rely upon username as a key
func (me *UserJSONCollection) UpdateUser(user User) error {

	me.Lock()
	for k, v := range me.Users {
		if v.UserName == user.UserName {
			if v.Password != user.Password {
				hashPWD, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
				if err != nil {
					me.Unlock()
					log.Println("Cannot create password hash")
					return errors.New("Cannot create password hash")
				}
				user.Password = string(hashPWD)
			}
			me.Users[k] = user
			me.Unlock()
			// fmt.Println("We need to save the update to Postgres too- remembering both user and topics")
			return me.Save("")
		}
	}
	me.Unlock()
	return errors.New("Could not find user")
}

// DeleteUser removes a user from the collection, using the username as a key
func (me *UserJSONCollection) DeleteUser(username string) error {

	me.Lock()
	for k, v := range me.Users {
		if v.UserName == username {
			me.Users[k] = me.Users[len(me.Users)-1]
			me.Users = me.Users[:len(me.Users)-1]
			me.Unlock()
			me.Save("")
			return nil
		}
	}
	me.Unlock()
	return errors.New("User Not Found")
}

// GetUserByToken returns a user from the collection using the token as a key
func (me *UserJSONCollection) GetUserByToken(token string) (User, error) {

	if len(me.Users) == 0 {
		me.Load()
	}
	me.RLock()
	defer me.RUnlock()

	for _, v := range me.Users {
		if v.Token == token {
			return v, nil
		}
	}
	var blankUser User
	return blankUser, errors.New("User not found")
}

// GetUserByUsername returns a user from the collection using username as a key
func (me *UserJSONCollection) GetUserByUsername(username string) (User, error) {

	me.RLock()
	defer me.RUnlock()

	for _, v := range me.Users {
		if v.UserName == username {
			return v, nil
		}
	}
	var blankUser User
	return blankUser, errors.New("User not found")
}

// GetUsers returns all the users from the collection
func (me *UserJSONCollection) GetUsers() []User {
	me.RLock()
	defer me.RUnlock()
	return me.Users
}

// AddTopicToUser adds a new topic to an existing user
func (me *UserJSONCollection) AddTopicToUser(username string, topic Topic) error {

	targetUser, getTargetUserError := me.GetUserByUsername(username)
	if getTargetUserError != nil {
		return errors.New("Could not find user")
	}

	for _, v := range targetUser.Topics {
		if v.TopicString == topic.TopicString {
			return errors.New("Topic already exists")
		}
	}

	targetUser.Topics = append(targetUser.Topics, topic)
	return me.UpdateUser(targetUser)
}

// EditTopicForUser edits and existing topic for an existing user in the collection
func (me *UserJSONCollection) EditTopicForUser(username string, topic Topic) error {

	targetUser, getTargetUserError := me.GetUserByUsername(username)
	if getTargetUserError != nil {
		return errors.New("Could not find user")
	}

	found := false
	me.RLock()
	for k, v := range targetUser.Topics {
		if v.TopicString == topic.TopicString {
			targetUser.Topics[k] = topic
			found = true
			break
		}
	}
	me.RUnlock()
	if !found {
		return errors.New("Topic not found")
	}
	return me.UpdateUser(targetUser)
}

// DeleteTopicFromUser removes a topic permission for that user if the topic does not exist it returns an error
func (me *UserJSONCollection) DeleteTopicFromUser(username string, topicString string) error {

	targetUser, getTargetUserError := me.GetUserByUsername(username)
	if getTargetUserError != nil {
		return errors.New("Could not find user")
	}

	for k, v := range targetUser.Topics {
		if v.TopicString == topicString {
			targetUser.Topics[k] = targetUser.Topics[len(targetUser.Topics)-1]
			targetUser.Topics = targetUser.Topics[:len(targetUser.Topics)-1]

			return me.UpdateUser(targetUser)
		}
	}
	return errors.New("Topic not found")
}

// UpdateUserToken updates the token for an existing user upon login
func (me *UserJSONCollection) UpdateUserToken(username string, newtoken string) error {

	me.Lock()
	for k, v := range me.Users {
		if v.UserName == username {
			v.Token = newtoken
			me.Users[k] = v
			me.Unlock()
			saveError := me.Save("")
			return saveError
		}
	}
	me.Unlock()
	return errors.New("User not found")
}

// Save saves the users collection in a json file
func (me *UserJSONCollection) Save(fname string) error {

	me.RLock()
	defer me.RUnlock()
	if fname == "" {
		fname = me.Fname
	}

	// fmt.Println("me.users: ", me.Users)
	b, err := json.MarshalIndent(me.Users, " ", " ")
	if err != nil {
		log.Println("Could not unmarshall ", err)
		return err
	}

	err = ioutil.WriteFile(fname, b, 0644)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}
