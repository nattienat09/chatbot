package main

import (
        "fmt"
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
        customerId := 1
        productId := 5

/*session := &handler.Session{
        UserId: customerId,
        Messages: []openai.ChatCompletionMessage{},
        ReviewCollected: false,
}*/
r := mux.NewRouter()

        r.HandleFunc("/chat", WithSession(handler.ChatHandler(client, productId, customerId))).Methods("GET","POST")
        r.HandleFunc("/", IndexHandler).Methods("GET", "POST")

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
		session.Options = &sessions.Options {
			Path: "/",
			MaxAge: 300,
			HttpOnly: true,
		}
		if session.IsNew {
			session.Values["chat_history"] = []openai.ChatCompletionMessage{}
			session.Values["review_collected"] = false
		}
		sessions.Save(r,w)
		next.ServeHTTP(w, r)
	}
}

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("index.html")
	if err != nil {
		http.Error(w, "Could not load template.", http.StatusInternalServerError)
		return
	}
	productName, err := dbUtils.GetProductNameFromProductId(5)
	if err!= nil {
		log.Printf("Failed to get product name from product id %v\n", err)
		http.Error(w, "Failed to get product name", http.StatusInternalServerError)
		return
	}
	data := map[string]interface{}{
		"ProductName":productName,
	}
	tmpl.Execute(w, data)
}
