package server

import (
	"authserver/config"
	"authserver/store"
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"sync"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

var Server *MyServer

// MyServer is a container for the Server stuff
type MyServer struct {
	http.Server
	shutdownReq chan bool
	WG          sync.WaitGroup
}

// NewServer - this is the init function for the server process
func NewServer(port string, store *store.UserPersistence) *MyServer {

	// prepare handler to use store
	storeHandler := SetStoreHandler(store)
	// create server - this version creates a server that listens on any address
	s := &MyServer{
		Server: http.Server{
			Addr:         "127.0.0.1:" + port,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
		},
		shutdownReq: make(chan bool),
	}

	router := mux.NewRouter()

	// Swagger
	sh := http.StripPrefix("/mqtt/swaggerui/", http.FileServer(http.Dir("assets/swaggerui/")))
	router.PathPrefix("/mqtt/swaggerui/").Handler(sh)

	// CORS stuff
	headersOk := handlers.AllowedHeaders([]string{"X-Requested-With", "X-API-KEY", "X-Request-Token", "Content-Type"})
	originsOk := handlers.AllowedOrigins([]string{"*"})
	methodsOk := handlers.AllowedMethods([]string{"GET", "HEAD", "POST", "PUT", "DELETE", "OPTIONS"})
	s.Handler = handlers.CORS(headersOk, originsOk, methodsOk)(router)

	// hmq handlers
	router.HandleFunc("/mqtt/auth", storeHandler.AuthHandler)
	router.HandleFunc("/mqtt/acl", storeHandler.ACLHandler)
	router.HandleFunc("/mqtt/superuser", storeHandler.SuperUserHandler)

	// http users handlers
	router.HandleFunc("/mqtt/login", storeHandler.Login)
	router.HandleFunc("/mqtt/listusers", storeHandler.ListUsers)
	router.HandleFunc("/mqtt/getuser/{userID}", storeHandler.GetUser)
	router.HandleFunc("/mqtt/adduser", storeHandler.AddUser)
	router.HandleFunc("/mqtt/edituser", storeHandler.EditUser)
	router.HandleFunc("/mqtt/deleteuser/{userID}", storeHandler.DeleteUser)

	// http topics handlers
	router.HandleFunc("/mqtt/addusertopic/{userID}", storeHandler.AddUserTopic)
	router.HandleFunc("/mqtt/editusertopic/{userID}", storeHandler.EditUserTopic)
	router.HandleFunc("/mqtt/deletetopic", storeHandler.DeleteTopic)
	router.HandleFunc("/mqtt/topics/{userID}", storeHandler.CheckUserTopics)
	router.HandleFunc("/mqtt/checkTopicAuth", storeHandler.CheckTopicAuth)

	return s
}

func (s *MyServer) WaitShutdown() {
	irqSig := make(chan os.Signal, 1)
	signal.Notify(irqSig, syscall.SIGINT, syscall.SIGTERM)

	//Wait interrupt or shutdown request through /shutdown
	select {
	case sig := <-irqSig:
		log.Printf("Shutdown request (signal: %v)", sig)
	case sig := <-s.shutdownReq:
		log.Printf("Shutdown request (/shutdown %v)", sig)
	}
	log.Printf("Stopping API server ...")
	close(config.Done)
	//Create shutdown context with 10 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	//shutdown the server
	err := s.Shutdown(ctx)
	if err != nil {
		log.Printf("Shutdown request error: %v", err)
	}
	log.Println("Waiting for waitgroup to clear")

	s.WG.Wait()
}

func (s *MyServer) RootHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("HTTPAUTH - see /httpauth/V01/swaggerui/ for documentation\n"))
}
