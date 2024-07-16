package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"chatbot/db"
	"chatbot/handler"
	"chatbot/session"
	"github.com/joho/godotenv"
	"github.com/gorilla/mux"
	openai "github.com/sashabaranov/go-openai"
)

var (
	client *openai.Client
)


func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	MY_OPENAI_KEY := os.Getenv("MY_OPENAI_KEY")
	DBUSER := os.Getenv("DBUSER")
	DBPASS := os.Getenv("DBPASS")
	dbUtils.InitDb(DBUSER, DBPASS)
	if MY_OPENAI_KEY == "" {
		log.Fatal("OPENAI_API_KEY environment variable is not set")
	}
	client = openai.NewClient(MY_OPENAI_KEY)
}


func main() {
	r := mux.NewRouter()

	r.HandleFunc("/", handler.IndexHandler).Methods("GET")
	r.HandleFunc("/confirmDelivery", session.WithSession(handler.ConfirmDeliveryHandler)).Methods("POST")
	r.HandleFunc("/chat", handler.ChatHandler(client)).Methods("POST")

	fmt.Println("Server is running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
