package utils

import (
	"encoding/json"
	"log"
	"strings"

	"net/http"
)

// utils.GetSentValFromRequest is a simple helper function to return a parameter from
// a request - it first checks to see if the parameter is set as a header, then as a query param, then as a form element
// if it doesn't find the parameter it returns an empty string
func GetSentValFromRequest(r *http.Request, sentval string) string {
	param := ""
	// Always prefer the header
	param = r.Header.Get(strings.Replace(sentval, "_", "-", -1))
	// Then look for a GET
	if param == "" {
		param = r.URL.Query().Get(sentval)
	}
	// Now look for a POST (ie form)
	if param == "" {
		err := r.ParseForm() // Parses the request body
		if err != nil {
			log.Println(err)
		}
		param = r.Form.Get(sentval) // x will be "" if parameter is not set

	}
	//fmt.Println(sentval, "=", param)
	return param
}

func GetTokenFromRequest(r *http.Request) string {
	var tmptoken string
	tmptoken = r.Header.Get("X-API-KEY")
	if tmptoken != "" {
		return tmptoken
	}
	tmptoken = r.URL.Query().Get("token")
	if tmptoken != "" {
		return tmptoken
	}

	tmptoken = r.Header.Get("wf-tkn")
	if tmptoken == "" {
		tmptoken = r.URL.Query().Get("wf_tkn")
	}

	return tmptoken
}

// ReturnWithError is a simple helper function to return a standardised error to the client
func ReturnWithError(ErrorType int, ErrorMessage string, w http.ResponseWriter) {
	type ret struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	}
	log.Println("Return with Error called with message = ", ErrorMessage)
	var retval ret
	retval.Status = "error"
	retval.Message = ErrorMessage

	outbytes, outerr := json.Marshal(retval)
	if outerr != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(outerr.Error()))

	} else {
		w.WriteHeader(ErrorType)
		w.Write(outbytes)
	}
}

// ReturnWithError is a simple helper function to return a standardised error to the client
func ReturnOK(Message string, Token string, w http.ResponseWriter) {
	type ret struct {
		Status  string `json:"status"`
		Message string `json:"message"`
		Token   string `json:"token"`
	}
	var retval ret
	retval.Status = "ok"
	retval.Message = Message
	retval.Token = Token

	outbytes, outerr := json.Marshal(retval)
	if outerr != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(outerr.Error()))
		return

	}
	w.Write(outbytes)

}

// ReturnWithError is a simple helper function to return a standardised error to the client
func ReturnOKWithData(Message string, Data interface{}, Token string, w http.ResponseWriter) {
	type ret struct {
		Status  string      `json:"status"`
		Message string      `json:"message"`
		Data    interface{} `json:"data"`
		Token   string      `json:"token"`
	}
	var retval ret
	retval.Status = "ok"
	retval.Message = Message
	retval.Token = Token
	retval.Data = Data

	outbytes, outerr := json.MarshalIndent(retval, " ", " ")
	if outerr != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(outerr.Error()))

	}
	w.Write(outbytes)

}
