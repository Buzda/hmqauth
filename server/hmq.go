package server

import (
	"authserver/utils"
	"log"
	"net/http"
	"strings"
)

// AuthHandler authenticates the mqtt client trying to connect to hmq broker
func (me *StoreHandler) AuthHandler(w http.ResponseWriter, r *http.Request) {

	error := r.ParseForm()
	if error != nil {
		log.Fatalln(error)
	}

	password := r.Form["password"][0]
	username := r.Form["username"][0]

	_, loginErr := me.store.Login(username, password, false)
	if loginErr != nil {
		utils.ReturnWithError(http.StatusUnauthorized, "Invalid login", w)
		return
	}
	return
}

// ACLHandler verifies the client has the write to pub/sub to the topic
func (me *StoreHandler) ACLHandler(w http.ResponseWriter, r *http.Request) {

	access := utils.GetSentValFromRequest(r, "access")
	topic := utils.GetSentValFromRequest(r, "topic")
	username := utils.GetSentValFromRequest(r, "username")

	thisUser, getUserError := me.store.GetUserByUsername(username)
	if getUserError != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	userPub, userSub, CheckErr := thisUser.CheckTopicAuth(topic)
	if CheckErr != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	hasHash := strings.Index(topic, "#")
	hasPlus := strings.Index(topic, "+")

	switch access {
	case "1":
		if userSub == true {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusNoContent)
		}
	case "2":
		if userPub == true && hasHash < 0 && hasPlus < 0 {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusNoContent)
		}
	}
}

// SuperUserHandler is unfinished - we really need to address this one
func (me *StoreHandler) SuperUserHandler(w http.ResponseWriter, r *http.Request) {

	utils.ReturnWithError(http.StatusInternalServerError, "Not implemented", w)
	return
	/*
		fmt.Println("Superuser Request")
		//fmt.Println(r.Method)
		//fmt.Println(r)
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println(string(body))

		}*/
}
