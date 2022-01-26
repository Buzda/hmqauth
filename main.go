package main

import (
	"authserver/config"
	"authserver/server"
	"authserver/store"
	"authserver/utils"
	"fmt"
	"log"
)

func main() {

	// load the configuration
	configLoadErr := config.Config.LoadFromFile("assets/config.json")
	if configLoadErr != nil {
		log.Println("Could not get configuration", configLoadErr)
		return
	}
	utils.Logging()

	store := store.NewStorage(config.Config.GetStorageType())
	storeLoadErr := store.Load()
	if storeLoadErr != nil {
		log.Println("Error in loading data")
	}

	server.Server = server.NewServer(config.Config.GetPort(), &store)

	fmt.Printf("Starting Server on port %v\n", server.Server.Addr)
	log.Printf("Starting Server on port %v\n", server.Server.Addr)
	go func() {
		// This starts the HTTP server
		err := server.Server.ListenAndServe()

		if err != nil {
			log.Fatalln("Cannot Start Server, exiting:", err.Error())
		}
	}()

	//wait shutdown
	server.Server.WaitShutdown()
	config.WG.Wait()

	log.Printf("Service Exiting")
}
