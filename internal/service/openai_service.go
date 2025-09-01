package service

import (
	bytes "bytes"
	json "encoding/json"
	fmt "fmt"
	ioutil "io/ioutil"
	nethttp "net/http"
	os "os"
	time "time"
)

// OpenAIChatMessage represents a message for OpenAI chat completion
type OpenAIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenAIChatRequest is the request payload for OpenAI chat completions
type OpenAIChatRequest struct {
	Model       string              `json:"model"`
	Temperature float64             `json:"temperature"`
	Messages    []OpenAIChatMessage `json:"messages"`
}

// OpenAIChatResponseChoice is a single choice in the response
type OpenAIChatResponseChoice struct {
	Message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"message"`
}

// OpenAIChatResponse is the response from OpenAI chat completions
type OpenAIChatResponse struct {
	Choices []OpenAIChatResponseChoice `json:"choices"`
}

// CallOpenAIChatCompletion calls the OpenAI chat completions API
func CallOpenAIChatCompletion(model string, temperature float64, messages []OpenAIChatMessage) (string, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("OPENAI_API_KEY not set")
	}
	url := "https://api.openai.com/v1/chat/completions"
	// Always use temperature 1 for supported models
	payload := OpenAIChatRequest{
		Model:       model,
		Temperature: temperature,
		Messages:    messages,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("error marshaling OpenAI payload: %w", err)
	}
	fmt.Printf("[OpenAI] Request payload: %s\n", string(body))

	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		req, err := nethttp.NewRequest("POST", url, bytes.NewBuffer(body))
		if err != nil {
			lastErr = fmt.Errorf("error creating OpenAI request: %w", err)
			break
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)
		client := &nethttp.Client{}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("error sending OpenAI request: %w", err)
			fmt.Printf("[OpenAI] Attempt %d failed: %v\n", attempt, lastErr)
			time.Sleep(2 * time.Second)
			continue
		}
		defer resp.Body.Close()
		fmt.Printf("[OpenAI] Response status: %s\n", resp.Status)
		respBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			lastErr = fmt.Errorf("error reading OpenAI response body: %w", err)
			fmt.Printf("[OpenAI] Attempt %d failed: %v\n", attempt, lastErr)
			time.Sleep(2 * time.Second)
			continue
		}
		fmt.Printf("[OpenAI] Full response body: %s\n", string(respBody))
		if resp.StatusCode != 200 {
			lastErr = fmt.Errorf("OpenAI API error: status %d, body: %s", resp.StatusCode, string(respBody))
			fmt.Printf("[OpenAI] Attempt %d failed: %v\n", attempt, lastErr)
			time.Sleep(2 * time.Second)
			continue
		}
		var openaiResp OpenAIChatResponse
		if err := json.Unmarshal(respBody, &openaiResp); err != nil {
			lastErr = fmt.Errorf("error unmarshaling OpenAI response: %w; body: %s", err, string(respBody))
			fmt.Printf("[OpenAI] Attempt %d failed: %v\n", attempt, lastErr)
			time.Sleep(2 * time.Second)
			continue
		}
		if len(openaiResp.Choices) == 0 {
			lastErr = fmt.Errorf("no choices returned from OpenAI; response body: %s", string(respBody))
			fmt.Printf("[OpenAI] Attempt %d failed: %v\n", attempt, lastErr)
			time.Sleep(2 * time.Second)
			continue
		}
		// Success
		return openaiResp.Choices[0].Message.Content, nil
	}
	return "", lastErr
}
