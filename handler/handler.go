package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
"chatbot/db"
"chatbot/session"
	"chatbot/analyzer"
	"github.com/gorilla/sessions"
	openai "github.com/sashabaranov/go-openai"
)

type ChatRequest struct {
	Message string `json:"message"`
}

type ChatResponse struct {
	Response string `json:"response"`
}


func addMessageToSession(session *sessions.Session, message openai.ChatCompletionMessage) {
    history := getMessagesFromSession(session)
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

func getMessagesFromSession(session *sessions.Session) []openai.ChatCompletionMessage {
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

func reviewHasBeenCollected(session *sessions.Session) {
	session.Values["review_collected"] = true
}

var mu sync.Mutex
const PROMPT_BEFORE_REVIEW = `You are a review chatbot you must ask the customer for a review of the %s they purchased from my shop recently. you will ask them to provide a review in the form of a number from 1 to 5. Do not greet the user with hello. Jump straight to the review process. Persist until they give you a 1 to 5 rating. Keep asking for it.

                                        Be friendly and helpful in your interactions.

                                        Feel free to ask customers about their preferences, recommend products, and inform them about any ongoing promotions.
                                        do not answer any question irrelevant to the %s politely return to the topic of the product review. I am also providing you a history of the chat.

                                        Make the shopping experience enjoyable and encourage customers to reach out if they have any questions or need assistance. If you have already collected a review from the user do not ask for another one.`

const PROMPT_AFTER_REVIEW = `you just received a review for the %s.react accordingly to the review you received. Thank the user and don't forget to ask them specifics about their review. What they liked and what they didn't like. Be friendly and helpful in your interactions. Provide any other info they may ask about the %s. 
                                        Never ask for a review again !!! If the user does not want to give any more comments then thank them and say bye.`


func ChatHandler(client *openai.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		//figure out how to send the first message
		session, err := session.Store.Get(r, "chatbot-session")
		if err != nil {
			http.Error(w, "Error retrieving session", http.StatusInternalServerError)
			return
		}

		var chatRequest ChatRequest
		var messages []openai.ChatCompletionMessage

		if err := json.NewDecoder(r.Body).Decode(&chatRequest); err != nil {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			log.Printf("Invalid request payload: %v\n", err)
			return
		}

		customerId := session.Values["customer_id"].(int)
		productId := session.Values["product_id"].(int)
		productName := session.Values["product_name"].(string)
		if err != nil {
			log.Printf("Failed to get product name: %v\n", err)
			http.Error(w, "Failed to get product name", http.StatusInternalServerError)
			return
		}

		// Lock to ensure safe concurrent access to shared resources
		mu.Lock()
		defer mu.Unlock()

		// Retrieve or initialize user-specific messages
		userMessages := getMessagesFromSession(session)
		userMessages = append(userMessages, openai.ChatCompletionMessage{
                        Role:    "user",
                        Content: chatRequest.Message,
                })


		fmt.Printf("review collected %v\n",session.Values["review_collected"])
		if session.Values["review_collected"] == false {
			fmt.Printf("i clearly have not got a review\n", session.Values["review_collected"])
		messageAnalysis, err := analyzer.AnalyzeMessages(client, userMessages, productName)
		if err != nil {
			log.Printf("Message analysis error: %v\n", err)
			http.Error(w, "OpenAI failed to analyse message", http.StatusInternalServerError)
			return
		}

		if messageAnalysis.Confidence <= 0.8 { // add flag to signal that the review is already there
			// Prepare messages to send to OpenAI API
			messages = append([]openai.ChatCompletionMessage{
				{
					Role:    "system",
					Content: fmt.Sprintf(PROMPT_BEFORE_REVIEW, productName, productName),
				},
			}, openai.ChatCompletionMessage{
                        Role:    "user",
                        Content: chatRequest.Message,
                })
			//userMessages...) //do i really need to pass it everything
		} else {
			if session.Values["review_collected"] == false {
				err := dbUtils.SaveCustomerRating(productId, customerId, messageAnalysis.Review)
				if err != nil {
					log.Printf("Couldn't save user review to db: %v\n", err)
					http.Error(w, "Failed to save user review to db.", http.StatusInternalServerError)
					return
				}
				reviewHasBeenCollected(session)
			}

			// add code to save messageAnalysis.Review, also save the text in additional reviews.
			// Prepare messages to send to OpenAI API
			messages = append([]openai.ChatCompletionMessage{
				{
					Role:    "system",
					Content: fmt.Sprintf(PROMPT_AFTER_REVIEW, productName),
				},
			}, openai.ChatCompletionMessage{
                        Role:    "user",
                        Content: chatRequest.Message,
                })//userMessages...)
		}} else {
			messages = append([]openai.ChatCompletionMessage{
                                {
                                        Role:    "system",
                                        Content: fmt.Sprintf(PROMPT_AFTER_REVIEW,productName),
                                },
                        }, openai.ChatCompletionMessage{
                        Role:    "user",
                        Content: chatRequest.Message,
                })
		}

		resp, err := client.CreateChatCompletion(r.Context(), openai.ChatCompletionRequest{
			Model:    "gpt-3.5-turbo",
			Messages: messages,
		})

		if err != nil {
			log.Printf("ChatCompletion error: %v\n", err)
			http.Error(w, "Failed to get response from OpenAI", http.StatusInternalServerError)
			return
		}

		addMessageToSession(session,openai.ChatCompletionMessage{
                        Role:    "user",
                        Content: chatRequest.Message,
                })
		addMessageToSession(session,openai.ChatCompletionMessage{
                        Role:    "assistant",
                        Content: resp.Choices[0].Message.Content,
                })



		chatResponse := ChatResponse{Response: resp.Choices[0].Message.Content}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(chatResponse)
	}
}

