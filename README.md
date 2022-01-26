# hmqauth

## Function:

This project authenticates MQTT clients and gives them permissions, by working with the hmq (broker) project 

## Working princple:

This is essentially a HTTP server that listens to HTTP requests coming from the hmq MQTT broker. The latter sends these requests when it receives a conenction from a MQTT client. hmqauth then process the requests and responds with ok/reject response, and consquently, hmq broker approves or rejects the MQTT client connection.

The current implementation uses two ways of storing the users and their MQTT topics:

* postgresQL
* JSON file

## Config file example:

{
    "Connstring": "host=<ip> port=5432 user=<username> password=<password> dbname=<dbname> sslmode=disable",
    "Port": "9090",
    "StorageTypeJSON": "json",
    "StorageFileName": "assets/users.json"
}