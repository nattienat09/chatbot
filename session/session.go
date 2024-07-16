package session

import (
        "encoding/json"
        "log"
        "net/http"
        "github.com/gorilla/sessions"
        openai "github.com/sashabaranov/go-openai"

)

var (
        Store = sessions.NewCookieStore([]byte("secret-key"))
)


func ReviewHasBeenCollected(session *sessions.Session) {
        session.Values["review_collected"] = true
}

func AddMessageToSession(session *sessions.Session, message openai.ChatCompletionMessage) {
    history := GetMessagesFromSession(session)
    history = append(history, message)

    // Convert history to JSON bytes
    jsonBytes, err := json.Marshal(history)
    if err != nil {
        log.Printf("Failed to marshal chat history: %v\n", err)
        return
    }

    // Save JSON bytes to session
    session.Values["chat_history"] = jsonBytes
}

func GetMessagesFromSession(session *sessions.Session) []openai.ChatCompletionMessage {
    var history []openai.ChatCompletionMessage

    // Retrieve JSON bytes from session
    jsonBytes, ok := session.Values["chat_history"].([]byte)
    if !ok {
        return history
    }

    // Unmarshal JSON bytes to []openai.ChatCompletionMessage
    err := json.Unmarshal(jsonBytes, &history)
    if err != nil {
        log.Printf("Failed to unmarshal chat history: %v\n", err)
    }

    return history
}


func WithSession(next http.HandlerFunc) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
                session, err := Store.Get(r, "chatbot-session")
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

