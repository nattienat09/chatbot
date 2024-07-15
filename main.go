package main

import (
	"fmt"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"text/template"
	"chatbot/db"
	"chatbot/handler"
	"chatbot/session"
	"github.com/joho/godotenv"
	"github.com/gorilla/sessions"
	"github.com/gorilla/mux"
	openai "github.com/sashabaranov/go-openai"
)


func main() {
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
	client := openai.NewClient(MY_OPENAI_KEY)

	r := mux.NewRouter()

	r.HandleFunc("/", IndexHandler).Methods("GET")
	r.HandleFunc("/confirmDelivery", WithSession(ConfirmDeliveryHandler)).Methods("POST")
	r.HandleFunc("/chat", handler.ChatHandler(client)).Methods("POST")

	fmt.Println("Server is running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}

func WithSession(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, err := session.Store.Get(r, "chatbot-session")
		if err != nil {
			http.Error(w, "Error retrieving session", http.StatusInternalServerError)
			return
		}
		session.Options = &sessions.Options{
			Path:     "/",
			MaxAge:   300,
			HttpOnly: true,
		}
		if session.IsNew {
			jsonBytes, err := json.Marshal([]openai.ChatCompletionMessage{})
        if err != nil {
               log.Printf("Failed to marshal chat history: %v\n", err)
               return
       }

    // Save JSON bytes to session
    session.Values["chat_history"] = jsonBytes
			session.Values["review_collected"] = false
		}
		sessions.Save(r, w)
		next.ServeHTTP(w, r)
	}
}

func ConfirmDeliveryHandler(w http.ResponseWriter, r *http.Request) {
	type ConfirmDeliveryRequest struct {
		CustomerID int `json:"customerId"`
		ProductID  int `json:"productId"`
	}

	var request ConfirmDeliveryRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		log.Printf("Invalid request payload: %v\n", err)
		return
	}

	// Store customer ID and product ID in the session
	session, err := session.Store.Get(r, "chatbot-session")
	if err != nil {
		http.Error(w, "Error retrieving session", http.StatusInternalServerError)
		log.Printf("Error retrieving session: %v\n", err)
		return
	}

	session.Values["customer_id"] = request.CustomerID
	session.Values["product_id"] = request.ProductID
	jsonBytes, err := json.Marshal([]openai.ChatCompletionMessage{})
	if err != nil {
               log.Printf("Failed to marshal chat history: %v\n", err)
	       return
       }

    // Save JSON bytes to session
    session.Values["chat_history"] = jsonBytes
	session.Values["review_collected"] = false


	// Retrieve product name from database
	productName, err := dbUtils.GetProductNameFromProductId(request.ProductID)
	if err != nil {
		log.Printf("Failed to get product name: %v\n", err)
		http.Error(w, "Failed to get product name", http.StatusInternalServerError)
		return
	}

	session.Values["product_name"] = productName
	// Save the session
	err = session.Save(r, w)
	if err != nil {
		log.Printf("Failed to save session: %v\n", err)
		http.Error(w, "Failed to save session", http.StatusInternalServerError)
		return
	}

	// Return the product name as part of the response
	response := struct {
		ProductName string `json:"productName"`
	}{
		ProductName: productName,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("index.html")
	if err != nil {
		http.Error(w, "Could not load template.", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}


// move things to init func
