package analyzer

import (
	"context"
	"fmt"
	"strings"
	"chatbot/utils"
	openai "github.com/sashabaranov/go-openai"
)

type AnalysisResult struct {
	Review     int     `json:"review"`
	Confidence float64 `json:"confidence"`
}


func concatenateMessages(messages []openai.ChatCompletionMessage) string {
	messageContents := make([]string, len(messages))
	for i,msg := range messages {
		if msg.Role == "user"{
			messageContents[i] = "User:" + msg.Content
		}else {
			messageContents[i] = "Chatbot:" + msg.Content
		}
	}
	return strings.Join(messageContents,"\n")
}


func AnalyzeMessages(client *openai.Client, user_messages []openai.ChatCompletionMessage, productName string) (*AnalysisResult, error) {
	prompt := fmt.Sprintf(utils.ANALYZER_PROMPT, productName)

	messages := []openai.ChatCompletionMessage{
		{
			Role:    "assistant",
			Content: prompt,
		},
		{
			Role: "user",
			Content: concatenateMessages(user_messages),
		},
	}

	fmt.Printf(prompt, concatenateMessages(user_messages))

	// Call OpenAI API to get response
	resp, err := client.CreateChatCompletion(context.Background(), openai.ChatCompletionRequest{
		Model:    "gpt-3.5-turbo",
		Messages: messages,
	})

	if err != nil {
		return nil, err
	}

	// Extract user-specific messages based on the response
	userResponse := resp.Choices[0].Message.Content

	//fmt.Printf("Bot response %s",userResponse)

	// Example parsing logic: assuming response format "Review: 4. Confidence: 0.95"
	var review int
	var confidence float64


	// Extract review and confidence from the response (assuming fixed format for simplicity)
	fmt.Sscanf(userResponse, "Review: %d. Confidence: %f", &review, &confidence)
	if ((review > 5) || (review < 1)) {
		confidence = 0 //we wont be saving it
	}

	result := &AnalysisResult{
		Review:     review,
		Confidence: confidence,
	}
	fmt.Printf("Review: %d. Confidence: %f", review, confidence)

	return result, nil
}
