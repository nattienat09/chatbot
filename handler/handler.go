package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"text/template"
	"chatbot/db"
	"chatbot/session"
	"chatbot/utils"
	"chatbot/analyzer"
	openai "github.com/sashabaranov/go-openai"
)

type ChatRequest struct {
	Message string `json:"message"`
}

type ChatResponse struct {
	Response string `json:"response"`
}

type ConfirmDeliveryRequest struct {
	CustomerID int `json:"customerId"`
	ProductID  int `json:"productId"`
}

type ConfirmDeliveryResponse struct {
	ProductName string `json:"productName"`
}


var mu sync.Mutex

func ChatHandler(client *openai.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sess, err := session.Store.Get(r, "chatbot-session")
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

		customerId := sess.Values["customer_id"].(int)
		productId := sess.Values["product_id"].(int)
		productName := sess.Values["product_name"].(string)
		if err != nil {
			log.Printf("Failed to get product name: %v\n", err)
			http.Error(w, "Failed to get product name", http.StatusInternalServerError)
			return
		}

		// Lock to ensure safe concurrent access to shared resources
		mu.Lock()
		defer mu.Unlock()

		// Retrieve or initialize user-specific messages
		userMessages := session.GetMessagesFromSession(sess)
		userMessages = append(userMessages, openai.ChatCompletionMessage{
			Role:    "user",
			Content: chatRequest.Message,
		})

		fmt.Printf("review collected: %v",sess.Values["review_collected"])

		if sess.Values["review_collected"] == true {
			messages = append([]openai.ChatCompletionMessage{
                                {
                                        Role:    "system",
                                        Content: fmt.Sprintf(utils.PROMPT_AFTER_REVIEW, productName),
                                },
                        }, openai.ChatCompletionMessage{
                                Role:    "user",
                                Content: chatRequest.Message,
                        })//userMessages...)}
		}else{
			messageAnalysis, err := analyzer.AnalyzeMessages(client, userMessages, productName)
			if err != nil {
				log.Printf("Message analysis error: %v\n", err)
				http.Error(w, "OpenAI failed to analyse message", http.StatusInternalServerError)
				return
			}
			if messageAnalysis.Confidence <= 0.8 {
				messages = append([]openai.ChatCompletionMessage{
					{	Role:    "system",
                                        Content: fmt.Sprintf(utils.PROMPT_BEFORE_REVIEW, productName, productName),
                                },
                        }, openai.ChatCompletionMessage{
                                Role:    "user",
                                Content: chatRequest.Message,
                        })
                        } else {
				err := dbUtils.SaveCustomerRating(customerId, productId, messageAnalysis.Review)
                                if err != nil {
                                        log.Printf("Couldn't save user review to db: %v\n", err)
                                        http.Error(w, "Failed to save user review to db.", http.StatusInternalServerError)
                                        return
                                }
                              session.ReviewHasBeenCollected(sess)
			      messages = append([]openai.ChatCompletionMessage{
                                {
                                        Role:    "system",
                                        Content: fmt.Sprintf(utils.PROMPT_AFTER_REVIEW, productName),
                                },
                              }, openai.ChatCompletionMessage{
                                Role:    "user",
                                Content: chatRequest.Message,
                              })
			}
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

		session.AddMessageToSession(sess,openai.ChatCompletionMessage{
			Role:    "user",
			Content: chatRequest.Message,
		})
		session.AddMessageToSession(sess,openai.ChatCompletionMessage{
			Role:    "assistant",
			Content: resp.Choices[0].Message.Content,
		})

		err = sess.Save(r, w)


		chatResponse := ChatResponse{Response: resp.Choices[0].Message.Content}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(chatResponse)
	}
}

func ConfirmDeliveryHandler(w http.ResponseWriter, r *http.Request) {
	var request ConfirmDeliveryRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		log.Printf("Invalid request payload: %v\n", err)
		return
	}

	// Store customer ID and product ID in the session
	sess, err := session.Store.Get(r, "chatbot-session")
	if err != nil {
		http.Error(w, "Error retrieving session", http.StatusInternalServerError)
		log.Printf("Error retrieving session: %v\n", err)
		return
	}

	sess.Values["customer_id"] = request.CustomerID
	sess.Values["product_id"] = request.ProductID
	jsonBytes, err := json.Marshal([]openai.ChatCompletionMessage{})
	if err != nil {
		log.Printf("Failed to marshal chat history: %v\n", err)
		return
	}

	// Save JSON bytes to session
	sess.Values["chat_history"] = jsonBytes
	sess.Values["review_collected"] = false


	// Retrieve product name from database
	productName, err := dbUtils.GetProductNameFromProductId(request.ProductID)
	if err != nil {
		log.Printf("Failed to get product name: %v\n", err)
		http.Error(w, "Failed to get product name", http.StatusInternalServerError)
		return
	}

	sess.Values["product_name"] = productName
	// Save the session
	err = sess.Save(r, w)
	if err != nil {
		log.Printf("Failed to save session: %v\n", err)
		http.Error(w, "Failed to save session", http.StatusInternalServerError)
		return
	}

	// Return the product name as part of the response
	response := ConfirmDeliveryResponse{
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

