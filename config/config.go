package config

// Copyright 2017 Gary Barnett. All rights reserved.
// Use of this source code is governed by the same BSD-style
// license that the Golang Team uses.
//
// This implements a simple configuration file, which is stored as a json object
// the configuration is intended to support being updated at run-time with the
// option to save the configuration so it is reloaded on restart, or just to
// be applied for this runtime session

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"sync"
)

// Config is a global containing the current configuration info
var Config Configuration

// Done is a global chan which is closed when the app is shutting down
var Done chan bool

// WG is a global waitgroup used in application shutdown
var WG sync.WaitGroup

//Configuration holds the runtime config info
type Configuration struct {
	Connstring      string
	Port            string
	StorageType     string // json or postgres
	StorageFileName string
	sync.RWMutex
}

// GetConnString returns the DB connection string as defined in the config.json
func (s *Configuration) GetConnString() string {
	s.RLock()
	defer s.RUnlock()
	return s.Connstring
}

// SetConnString allows the connection string to be set
func (s *Configuration) SetConnString(newval string) {
	s.Lock()
	defer s.Unlock()
	s.Connstring = newval
	return
}

// GetPort returns the port number the app is meant to listen on
func (s *Configuration) GetPort() string {
	s.RLock()
	defer s.RUnlock()
	return s.Port
}

// SetPort sets the port that the application is meant to listen on
func (s *Configuration) SetPort(newval string) {
	s.Lock()
	defer s.Unlock()
	s.Port = newval
	return
}

// GetStorageType returns the type of data storage, json or postgres
func (s *Configuration) GetStorageType() string {
	s.RLock()
	defer s.RUnlock()
	return s.StorageType
}

// GetStorageFileName returns the name of the storage file in case json is used
func (s *Configuration) GetStorageFileName() string {
	s.RLock()
	defer s.RUnlock()
	return s.StorageFileName
}

// SaveToFile saves the configuration
func (s *Configuration) SaveToFile(fname string) {
	if fname == "" {
		fname = "assets/config1.json"
	}
	s.Lock()
	defer s.Unlock()
	b, err := json.Marshal(s)
	if err != nil {
		log.Println("Could not marshal configuration to save it: ", err)
		return
	}
	err = ioutil.WriteFile(fname, b, 0644)
	if err != nil {
		log.Println("Save file: ", err)
	}
}

// GetJSON returns the JSON encoding of the configuration
func (s *Configuration) GetJSON() string {

	s.Lock()
	defer s.Unlock()
	b, err := json.Marshal(s)
	if err != nil {
		log.Println("Error in marshalling: ", err)
		return "could not convert config to JSON"
	}
	return string(b)
}

// LoadFromFile loads the configuration from a file
func (s *Configuration) LoadFromFile(fname string) error {
	if fname == "" {
		fname = "assets/config.json"
	}
	file, openerr := os.Open(fname)
	if openerr != nil {
		return openerr
	}
	decoder := json.NewDecoder(file)

	err := decoder.Decode(&s)
	if err != nil {
		log.Println("Error in decoding: ", err)
		return err
	}
	return nil
}
