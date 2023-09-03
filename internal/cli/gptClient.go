package cli

import (
	"context"
	"fmt"

	"github.com/pkoukk/tiktoken-go"
	"github.com/sashabaranov/go-openai"
)

type GptClient struct {
	ApiKey string
}

var apiClient *openai.Client

func (gpt *GptClient) getClient() *openai.Client {
	if apiClient != nil {
		return apiClient
	}

	apiClient = openai.NewClient(gpt.ApiKey)
	return apiClient
}

func (gpt *GptClient) GetSummarization(text string, ctx context.Context) (*string, error) {
	client := gpt.getClient()

	resp, err := client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: openai.GPT4,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: fmt.Sprintf("Make a bullet points summarization of this speech trascription: %v", text),
				},
			},
		},
	)

	if err != nil {
		return nil, err
	}

	return &resp.Choices[0].Message.Content, nil
}

// Used to split the text according to the max tokens usage required per request by the model
func (gpt *GptClient) TextPerTokenSplit(text string, maxTokenNumber int, model string) ([]string, error) {
	result := make([]string, 0)

	// Count Tokens used
	tkm, err := tiktoken.EncodingForModel(model)
	if err != nil {
		return nil, err
	}

	tokens := tkm.Encode(text, nil, nil)
	tokenNumber := len(tokens)

	// Build the returned slice
	if tokenNumber <= maxTokenNumber {
		result = append(result, text)
		return result, nil
	}

	// Split text by length
	splitNumber := (tokenNumber + (tokenNumber % maxTokenNumber)) / maxTokenNumber
	textRunes := []rune(text)
	itemLen := len(textRunes) / splitNumber
	for i := 0; i < splitNumber; i++ {

		currentRunes := textRunes[i*itemLen : i*itemLen+itemLen]
		result = append(result, string(currentRunes))
	}

	return result, nil
}
