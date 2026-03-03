package main

import (
	"faas_goida/handlers"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/run", handlers.RunGoidaHandler)

	port := ":8080"
	log.Printf("Сервер запущен на %s. Перейдите на http://localhost:8080/run", port)

	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatal(err)
	}
}
