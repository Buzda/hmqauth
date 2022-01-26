package server

import (
	"authserver/store"
	"authserver/utils"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"

	"net/http"

	"github.com/gorilla/mux"
)

type StoreHandler struct {
	store store.UserPersistence
}

// SetStoreHandler sets handler to use store
func SetStoreHandler(store *store.UserPersistence) *StoreHandler {
	return &StoreHandler{
		store: *store,
	}
}

// Login processes a login request
func (me *StoreHandler) Login(w http.ResponseWriter, r *http.Request) {

	type LoginRequest struct {
		UserName string `json:"username"`
		Password string `json:"password"`
	}
	var login LoginRequest

	if r.Method == "POST" {
		us, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Println(err)
		}

		newerr := json.Unmarshal([]byte(string(us)), &login)
		if newerr != nil {
			log.Println(newerr)
		}
	}
	if r.Method == "GET" {
		login.Password = utils.GetSentValFromRequest(r, "password")
		login.UserName = utils.GetSentValFromRequest(r, "username")
	}
	if login.Password == "" || login.UserName == "" {
		utils.ReturnWithError(http.StatusUnauthorized, "Must provide both a user id and password", w)
		return
	}

	loggedInUser, loginError := me.store.Login(login.UserName, login.Password, true)
	if loginError != nil {
		utils.ReturnWithError(http.StatusUnauthorized, loginError.Error(), w)
		return
	}
	utils.ReturnOKWithData("ok", loggedInUser, loggedInUser.Token, w)
}

// GetUser returns a JSON object containing a user
func (me *StoreHandler) GetUser(w http.ResponseWriter, r *http.Request) {

	user, userError := me.GetUserFromRequest(r)
	if userError != nil {
		utils.ReturnWithError(http.StatusUnauthorized, userError.Error(), w)
		return
	}
	userfilter := mux.Vars(r)["userID"]

	if user.Admin == false && user.UserName != userfilter {
		utils.ReturnWithError(http.StatusUnauthorized, "Insufficient rights", w)
		return
	}

	type UserEntry struct {
		UserName string `json:"username"`
		IsAdmin  bool   `json:"admin"`
	}

	for _, v := range me.store.GetUsers() {
		if userfilter == "" || v.UserName == userfilter {
			utils.ReturnOKWithData("", UserEntry{UserName: v.UserName, IsAdmin: v.Admin}, user.Token, w)
			return
		}
	}
	utils.ReturnWithError(http.StatusNotFound, "User not found", w)
}

// ListUsers returns a JSON object containing a list of users
func (me *StoreHandler) ListUsers(w http.ResponseWriter, r *http.Request) {

	user, userError := me.GetUserFromRequest(r)
	if userError != nil {
		utils.ReturnWithError(http.StatusUnauthorized, userError.Error(), w)
		return
	}
	userfilter := utils.GetSentValFromRequest(r, "user")

	if user.Admin == false && user.UserName != userfilter {
		utils.ReturnWithError(http.StatusUnauthorized, "Insufficient rights", w)
		return
	}

	type UserListEntry struct {
		UserName string `json:"username"`
		IsAdmin  bool   `json:"admin"`
	}

	var ReturnArr []UserListEntry

	for _, v := range me.store.GetUsers() {
		if userfilter == "" || v.UserName == userfilter {
			ReturnArr = append(ReturnArr, UserListEntry{UserName: v.UserName, IsAdmin: v.Admin})
		}
	}
	utils.ReturnOKWithData("", ReturnArr, user.Token, w)
}

// GetAdminUserFromRequest returns the requester if an admin
func (me *StoreHandler) GetAdminUserFromRequest(r *http.Request) (store.User, error) {

	adminuser, userError := me.GetUserFromRequest(r)
	if userError != nil {
		log.Println("Error Getting User From Request:", userError.Error())
	}

	if adminuser.Admin == false {
		log.Println("GetAdminUserFromRequest failed as user not admin:", adminuser.UserName)
		var blankUser store.User
		return blankUser, errors.New("Not Admin")
	}
	return adminuser, nil
}

// GetUserFromRequest returns the user making the request
func (me *StoreHandler) GetUserFromRequest(r *http.Request) (store.User, error) {

	token := utils.GetTokenFromRequest(r)

	var thisUser store.User
	if token == "" {
		log.Println("token not provided")
		return thisUser, errors.New("No token provided")
	}

	thisUser, _ = me.store.GetUserByToken(token)
	if thisUser.UserName == "" {
		log.Println("token is not valid: ")
		return thisUser, errors.New("Invalid Token")
	}

	thisUser.Token = token
	return thisUser, nil
}

// EditUser handles a request to add a new user
func (me *StoreHandler) EditUser(w http.ResponseWriter, r *http.Request) {
	user, userError := me.GetAdminUserFromRequest(r)
	if userError != nil {
		utils.ReturnWithError(http.StatusUnauthorized, userError.Error(), w)
		return
	}

	var tempUser store.User

	if r.Method == "POST" {
		us, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Println(err)
		}
		newerr := json.Unmarshal([]byte(string(us)), &tempUser)
		if newerr != nil {
			log.Println(newerr)
		}
	}
	if r.Method == "GET" {
		newUserName := utils.GetSentValFromRequest(r, "username")
		newUserAdmin := utils.GetSentValFromRequest(r, "admin")
		newUserPassword := utils.GetSentValFromRequest(r, "password")
		if newUserAdmin == "true" {
			tempUser.Admin = true
		} else {
			tempUser.Admin = false
		}
		tempUser.UserName = newUserName
		tempUser.Password = newUserPassword
	}

	var er error
	er = me.store.EditUser(tempUser)

	if er != nil {
		utils.ReturnWithError(http.StatusBadRequest, "Error in editing user:"+er.Error(), w)
		return
	}
	utils.ReturnOK("User edited", user.Token, w)
	return
}

// AddUser handles a request to add a new user
func (me *StoreHandler) AddUser(w http.ResponseWriter, r *http.Request) {
	user, userError := me.GetUserFromRequest(r)
	if userError != nil {
		utils.ReturnWithError(http.StatusUnauthorized, userError.Error(), w)
		return
	}
	if user.Admin == false {
		utils.ReturnWithError(http.StatusUnauthorized, "Insufficient rights", w)
		return
	}

	var tempUser store.User

	if r.Method == "POST" {
		us, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Println(err)
		}
		newerr := json.Unmarshal([]byte(string(us)), &tempUser)
		if newerr != nil {
			log.Println(newerr)
		}
	}
	if r.Method == "GET" {
		newUserName := utils.GetSentValFromRequest(r, "username")
		newUserAdmin := utils.GetSentValFromRequest(r, "admin")
		newUserPassword := utils.GetSentValFromRequest(r, "password")
		if newUserAdmin == "true" {
			tempUser.Admin = true
		} else {
			tempUser.Admin = false
		}
		tempUser.UserName = newUserName
		tempUser.Password = newUserPassword
	}

	var er error
	er = me.store.AddUser(tempUser)

	if er != nil {
		utils.ReturnWithError(http.StatusBadRequest, "Error in adding user:"+er.Error(), w)
		return
	}
	utils.ReturnOK("User Added", user.Token, w)
	return
}

// DeleteUser deletes a user from the userlist
func (me *StoreHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	thisUser, userError := me.GetUserFromRequest(r)
	if userError != nil {
		utils.ReturnWithError(http.StatusUnauthorized, userError.Error(), w)
		return
	}
	if thisUser.Admin == false {
		utils.ReturnWithError(http.StatusUnauthorized, "Insufficient rights", w)
		return
	}
	userToDelete := mux.Vars(r)["userID"]
	if thisUser.UserName == userToDelete {
		utils.ReturnWithError(http.StatusBadRequest, "You cannot delete yourself", w)
		return
	}

	var er error
	er = me.store.DeleteUser(userToDelete)

	if er != nil {
		utils.ReturnWithError(http.StatusBadRequest, er.Error(), w)
		return
	}
	utils.ReturnOK("users deleted", thisUser.Token, w)
}
