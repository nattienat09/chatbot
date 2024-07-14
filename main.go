package main

import (
        "fmt"
        "log"
        "net/http"
        "os"
	"text/template"
        "chatbot/db"
        "chatbot/handler"
        "github.com/joho/godotenv"
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
	productName,err := dbUtils.GetProductNameFromProductId(productId)
	if err != nil {
		log.Fatal("Error getting product name from database")
	}

session := &handler.Session{
        UserId: customerId,
        Messages: []openai.ChatCompletionMessage{},
        ReviewCollected: false,
}

        http.HandleFunc("/chat", handler.ChatHandler(client, productId, customerId, session))
        http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.ParseFiles("index.html")
		if err != nil {
			http.Error(w, "Could not load template.", http.StatusInternalServerError)
			return
		}
		data := map[string]interface{}{
			"ProductName": productName,
		}
		tmpl.Execute(w, data)
                //http.ServeFile(w, r, "index.html")
        })
        fmt.Println("Server is running on http://localhost:8080")
        log.Fatal(http.ListenAndServe(":8080", nil))
}
