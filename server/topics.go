package server

import (
	"authserver/store"
	"authserver/utils"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

// AddUserTopic adds an authorisation for a user to a given topic
func (me *StoreHandler) AddUserTopic(w http.ResponseWriter, r *http.Request) {

	user, userError := me.GetAdminUserFromRequest(r)
	if userError != nil {
		utils.ReturnWithError(http.StatusUnauthorized, userError.Error(), w)
		return
	}

	userToAddTopicTo := mux.Vars(r)["userID"]
	_, targetUserError := me.store.GetUserByUsername(userToAddTopicTo)
	if targetUserError != nil {
		utils.ReturnWithError(http.StatusNotFound, "User not found", w)
		return
	}

	var newTopic store.Topic

	if r.Method == "POST" {
		tp, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Println("Error Reading Body", err.Error())
			utils.ReturnWithError(http.StatusBadRequest, "Could not read request body", w)
			return
		}
		newerr := json.Unmarshal([]byte(string(tp)), &newTopic)
		if newerr != nil {
			utils.ReturnWithError(http.StatusInternalServerError, "Could not marshal request body", w)
			return
		}
	}
	if r.Method == "GET" {
		topicString := utils.GetSentValFromRequest(r, "topicstring")
		pubString := utils.GetSentValFromRequest(r, "pub")
		subString := utils.GetSentValFromRequest(r, "sub")

		pub := false
		sub := false
		if pubString == "1" || pubString == "true" {
			pub = true
		}
		if subString == "1" || subString == "true" {
			sub = true
		}
		if pub == false && sub == false {
			utils.ReturnWithError(http.StatusBadRequest, "Pub and Sub cannot both be false", w)
			return
		}
		if topicString == "" {
			utils.ReturnWithError(http.StatusBadRequest, "Topic cannot be blank", w)
			return
		}
		if string(topicString[len(topicString)-1]) == "/" {
			utils.ReturnWithError(http.StatusBadRequest, "Topic cannot end with a /", w)
			return
		}

		newTopic.Pub = pub
		newTopic.Sub = sub
		newTopic.TopicString = topicString
	}

	addTopicError := me.store.AddTopicToUser(userToAddTopicTo, newTopic)
	if addTopicError != nil {
		utils.ReturnWithError(http.StatusBadRequest, addTopicError.Error(), w)
		return
	}
	utils.ReturnOK("Topic added", user.Token, w)
}

// EditUserTopic edits an authorisation for a user to a given topic
func (me *StoreHandler) EditUserTopic(w http.ResponseWriter, r *http.Request) {

	user, userError := me.GetAdminUserFromRequest(r)
	if userError != nil {
		utils.ReturnWithError(http.StatusUnauthorized, userError.Error(), w)
		return
	}

	userToAddTopicTo := mux.Vars(r)["userID"]
	_, targetUserError := me.store.GetUserByUsername(userToAddTopicTo)
	if targetUserError != nil {
		utils.ReturnWithError(http.StatusNotFound, "User not found", w)
		return
	}

	var newTopic store.Topic
	if r.Method == "POST" {
		tp, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Println(err)
			utils.ReturnWithError(http.StatusBadRequest, "Could not read request body", w)
		}
		newerr := json.Unmarshal([]byte(string(tp)), &newTopic)
		if newerr != nil {
			log.Println(newerr)
			utils.ReturnWithError(http.StatusInternalServerError, "Could not marshal request body", w)
		}
	}
	if r.Method == "GET" {
		topicString := utils.GetSentValFromRequest(r, "topicstring")
		pubString := utils.GetSentValFromRequest(r, "pub")
		subString := utils.GetSentValFromRequest(r, "sub")

		pub := false
		sub := false
		if pubString == "1" || pubString == "true" {
			pub = true
		}
		if subString == "1" || subString == "true" {
			sub = true
		}
		if pub == false && sub == false {
			utils.ReturnWithError(http.StatusBadRequest, "Pub and Sub cannot both be false", w)
			return
		}
		if topicString == "" {
			utils.ReturnWithError(http.StatusBadRequest, "Topic cannot be blank", w)
			return
		}
		if string(topicString[len(topicString)-1]) == "/" {
			utils.ReturnWithError(http.StatusBadRequest, "Topic cannot end with a /", w)
			return
		}

		newTopic.Pub = pub
		newTopic.Sub = sub
		newTopic.TopicString = topicString
	}

	EditTopicError := me.store.EditTopicForUser(userToAddTopicTo, newTopic)
	if EditTopicError != nil {
		utils.ReturnWithError(http.StatusNotFound, EditTopicError.Error(), w)
		return
	}
	utils.ReturnOK("Topic modified", user.Token, w)
}

// DeleteTopic removes a user authorisation to a topic / topic pattern
// Permissions - at this stage a user cannot change their authorisation level - even to remove it
func (me *StoreHandler) DeleteTopic(w http.ResponseWriter, r *http.Request) {

	user, userError := me.GetAdminUserFromRequest(r)
	if userError != nil {
		utils.ReturnWithError(http.StatusUnauthorized, userError.Error(), w)
		return
	}

	var topicToDelete string
	var userToDeleteTopicFrom string

	if r.Method == "POST" {
		utils.ReturnWithError(http.StatusNotImplemented, "Delete topic via post request not implemented", w)
		return
	}
	if r.Method == "GET" {
		topicToDelete = utils.GetSentValFromRequest(r, "topic")
		userToDeleteTopicFrom = utils.GetSentValFromRequest(r, "username")
	}

	if topicToDelete == "" || userToDeleteTopicFrom == "" {
		utils.ReturnWithError(http.StatusBadRequest, "Must provide a username and a topic", w)
		return
	}

	deleteTopicError := me.store.DeleteTopicFromUser(userToDeleteTopicFrom, topicToDelete)
	if deleteTopicError != nil {
		utils.ReturnWithError(http.StatusNotFound, deleteTopicError.Error(), w)
		return
	}
	utils.ReturnOK("Topics deleted", user.Token, w)
}

// CheckUserTopics checks to see whether a topic is in the User's acl
// Any user can check their own topics, only admin users can check another user's topics
func (me *StoreHandler) CheckUserTopics(w http.ResponseWriter, r *http.Request) {

	user, userError := me.GetUserFromRequest(r)
	if userError != nil {
		utils.ReturnWithError(http.StatusUnauthorized, "Could not authorise user", w)
		return
	}

	userIDToGetTopicsFor := mux.Vars(r)["userID"]
	// Here we check - if the user is NOT an Admin AND the user is not the same user as the user for which the
	// topic info is requested then we have an authorisation error
	if user.Admin == false && user.UserName != userIDToGetTopicsFor {
		if userError != nil {
			utils.ReturnWithError(http.StatusUnauthorized, "Not Authorised", w)
			return
		}
	}

	searchTopic := utils.GetSentValFromRequest(r, "topic")
	userToGetTopicsFor, GetUserError := me.store.GetUserByUsername(userIDToGetTopicsFor)
	if GetUserError != nil {
		utils.ReturnWithError(http.StatusNotFound, "Could not get that user", w)
		return
	}
	if searchTopic == "" {
		type topicsReturn struct {
			Username string        `json:"username"`
			Topics   []store.Topic `json:"topics"`
		}
		var ttr topicsReturn
		ttr.Topics = userToGetTopicsFor.Topics
		ttr.Username = userToGetTopicsFor.UserName

		utils.ReturnOKWithData("ok", ttr, user.Token, w)
		return
	}
	for _, v := range userToGetTopicsFor.Topics {
		if v.TopicString == searchTopic {
			type topicReturn struct {
				Username string      `json:"username"`
				Topic    store.Topic `json:"topic"`
			}
			var tr topicReturn
			tr.Topic = v
			tr.Username = userToGetTopicsFor.UserName
			utils.ReturnOKWithData("ok", tr, user.Token, w)
			return
		}
	}
	utils.ReturnWithError(http.StatusNotFound, "topic not found", w)
	return
}

// CheckTopicAuth checks to see whether a user is authorised on a given topic
func (me *StoreHandler) CheckTopicAuth(w http.ResponseWriter, r *http.Request) {
	user, userError := me.GetUserFromRequest(r)
	if userError != nil {
		utils.ReturnWithError(http.StatusUnauthorized, userError.Error(), w)
		return
	}

	var usernameToCheck string
	var topicToCheck string
	var access string

	type topicCheck struct {
		Username string `json:"username"`
		Topic    string `json:"topic"`
		Access   string `json:"access"`
	}

	var tpCheck topicCheck

	if r.Method == "POST" {
		tp, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Println(err)
		}
		newerr := json.Unmarshal([]byte(string(tp)), &tpCheck)
		if newerr != nil {
			log.Println(newerr)
		}
		usernameToCheck = tpCheck.Username
		topicToCheck = tpCheck.Topic
		access = tpCheck.Access
	}

	if r.Method == "GET" {
		usernameToCheck = utils.GetSentValFromRequest(r, "username")
		topicToCheck = utils.GetSentValFromRequest(r, "topic")
		access = utils.GetSentValFromRequest(r, "access")
	}

	if usernameToCheck == "" || topicToCheck == "" || access == "" {
		utils.ReturnWithError(http.StatusBadRequest, "You must provide a username, topic, and access type to check", w)
		return
	}

	// Here we check - if the user is NOT an Admin AND the user is not the same user as the user for which the
	// topic info is requested then we have an authorisation error
	if user.Admin == false && user.UserName != usernameToCheck {

		utils.ReturnWithError(http.StatusUnauthorized, "Insufficient rights", w)
		return
	}
	if access != "pub" && access != "sub" {
		utils.ReturnWithError(http.StatusBadRequest, "The access type must be 'pub' or 'sub'", w)
		return
	}
	userToCheck, userToCheckError := me.store.GetUserByUsername(usernameToCheck)
	if userToCheckError != nil {
		utils.ReturnWithError(http.StatusNotFound, "Could not fetch user to check", w)
		return
	}
	userPub, userSub, CheckErr := userToCheck.CheckTopicAuth(topicToCheck)
	if CheckErr != nil {
		utils.ReturnWithError(http.StatusNotFound, "Could not get auth", w)
		return
	}
	if access == "pub" {
		utils.ReturnOKWithData("ok", userPub, user.Token, w)
		return
	}
	utils.ReturnOKWithData("ok", userSub, user.Token, w)
}
