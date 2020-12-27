package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/usadamasa/repeater4gcsr"
)

func main() {
	http.HandleFunc("/", repeater4gcsr.Index)
	http.HandleFunc("/webhook", repeater4gcsr.Webhook)
	http.HandleFunc("/driver", repeater4gcsr.Driver)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	err := http.ListenAndServe(fmt.Sprintf(":%s", port), nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
