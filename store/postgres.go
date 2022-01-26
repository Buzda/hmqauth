package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/rs/xid"
	"golang.org/x/crypto/bcrypt"
)

// InitPostgres returns the store object that uses postgresql connection
func InitPostgres(connString string) *UserPostgresCollection {

	log.Println("Storage type is postgres")
	UsersPostgres.DB, UsersPostgres.DBerr = pgxpool.Connect(context.Background(), connString)

	log.Println("DB Connected")
	if UsersPostgres.DBerr != nil {
		log.Println("Unabled to Create DB Connection", UsersPostgres.DBerr)
	}
	return &UsersPostgres
}

// Load loads the users along with their topics from the db
func (me *UserPostgresCollection) Load() error {

	var usersOut []User
	LoadUserQuery := "SELECT username,pwd,token,admin,topics FROM hmqusers"
	UserRows, UserRowsError := me.DB.Query(context.Background(), LoadUserQuery)
	if UserRowsError != nil {
		log.Println(UserRowsError)
		return UserRowsError
	}
	defer UserRows.Close()

	for UserRows.Next() {
		var dbUser User
		var dbUserName sql.NullString
		var dbPassword sql.NullString
		var dbToken sql.NullString
		var dbAdmin sql.NullBool

		scanner := UserRows.Scan(&dbUserName, &dbPassword, &dbToken, &dbAdmin, &dbUser.Topics)
		if scanner != nil {
			log.Println("scanner error: ", scanner)
		}
		dbUser.UserName = dbUserName.String
		dbUser.Password = dbPassword.String
		dbUser.Token = dbToken.String
		dbUser.Admin = dbAdmin.Bool
		usersOut = append(usersOut, dbUser)
	}
	me.Lock()
	me.Users = usersOut
	me.Unlock()
	return nil
}

// Login logs the user in and generates a token for the session if required
func (me *UserPostgresCollection) Login(username string, password string, requesttoken bool) (User, error) {

	userLoggingIn, getUserError := me.GetUserByUsername(username)
	if getUserError != nil {
		log.Println(getUserError)
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
func (me *UserPostgresCollection) AddUser(user User) error {
	// Validate the user
	// if the username and/or the password are blank then reject
	if user.UserName == "" || user.Password == "" {
		return errors.New("Username and password must both be non-blank")
	}
	//Add the User to the collection
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

	insertSQL := "INSERT INTO hmqusers (username, pwd, admin) VALUES ($1, $2, $3)"
	_, result := me.DB.Exec(context.Background(), insertSQL, user.UserName, user.Password, user.Admin)
	if result != nil {
		log.Println("Error in adding a user", result)
	}
	return result
}

// EditUser edits an existing user
func (me *UserPostgresCollection) EditUser(user User) error {
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

	insertSQL := "UPDATE hmqusers SET pwd=$1, admin=$2 WHERE username = $3"
	_, result := me.DB.Exec(context.Background(), insertSQL, user.Password, user.Admin, user.UserName)
	if result != nil {
		log.Println("Error in editing user: ", result)
	}
	return result
}

// UpdateUser accepts a user object and updates the relevant user
// users cannot change their name - so we can rely upon username as a key
func (me *UserPostgresCollection) UpdateUser(user User) error {

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

			insertSQL := "UPDATE hmqusers SET pwd=$1, admin=$2, topics=$3 WHERE username = $4"
			_, result := me.DB.Exec(context.Background(), insertSQL, user.Password, user.Admin, user.Topics, user.UserName)
			if result != nil {
				log.Println("Error in updating user: ", result)
			}
			return result
		}
	}
	me.Unlock()
	return errors.New("Could not find user")
}

// DeleteUser removes a user from the collection, using the username as a key
func (me *UserPostgresCollection) DeleteUser(username string) error {

	me.Lock()
	for k, v := range me.Users {
		if v.UserName == username {
			me.Users[k] = me.Users[len(me.Users)-1]
			me.Users = me.Users[:len(me.Users)-1]
			me.Unlock()
			insertSQL := "DELETE FROM hmqusers WHERE username=$1;"
			_, result := me.DB.Exec(context.Background(), insertSQL, username)
			if result != nil {
				log.Println("Error in deleting user: ", result)
			}
			return result
		}
	}
	me.Unlock()
	return errors.New("User Not Found")
}

// GetUserByToken returns a user from the collection using the token as a key
func (me *UserPostgresCollection) GetUserByToken(token string) (User, error) {

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
func (me *UserPostgresCollection) GetUserByUsername(username string) (User, error) {

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
func (me *UserPostgresCollection) GetUsers() []User {
	me.RLock()
	defer me.RUnlock()
	return me.Users
}

// AddTopicToUser adds a new topic to an existing user
func (me *UserPostgresCollection) AddTopicToUser(username string, topic Topic) error {

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
func (me *UserPostgresCollection) EditTopicForUser(username string, topic Topic) error {

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
func (me *UserPostgresCollection) DeleteTopicFromUser(username string, topicString string) error {

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
func (me *UserPostgresCollection) UpdateUserToken(username string, newtoken string) error {

	me.Lock()
	for k, v := range me.Users {
		if v.UserName == username {
			fmt.Println("v.Token: ", v.Token)
			v.Token = newtoken
			me.Users[k] = v
			me.Unlock()
			fmt.Println("newtoken: ", newtoken)
			insertSQL := "UPDATE hmqusers SET token = $1 WHERE username = $2;"
			_, result := me.DB.Exec(context.Background(), insertSQL, newtoken, username)
			if result != nil {
				log.Println("Error in updating token: ", result)
			}
			return result
		}
	}
	me.Unlock()
	return errors.New("User not found")
}
