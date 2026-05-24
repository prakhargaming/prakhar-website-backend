package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

const (
	embedURL    = "https://generativelanguage.googleapis.com/v1beta/models/gemini-embedding-2:embedContent?key=%s"
	generateURL = "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash:generateContent?key=%s"
)

type Client struct {
	apiKey string
	http   *http.Client
}

func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		http:   http.DefaultClient,
	}
}

type part struct {
	Text string `json:"text"`
}

type content struct {
	Parts []part `json:"parts"`
}

type embedRequest struct {
	Model    string  `json:"model"`
	Content  content `json:"content"`
	TaskType string  `json:"taskType"`
}

type embedResponse struct {
	Embedding struct {
		Values []float64 `json:"values"`
	} `json:"embedding"`
}

type generateRequest struct {
	Contents          []content `json:"contents"`
	SystemInstruction *content  `json:"systemInstruction,omitempty"`
}

type generateResponse struct {
	Candidates []struct {
		Content content `json:"content"`
	} `json:"candidates"`
}

func (c *Client) Embed(ctx context.Context, text string) ([]float64, error) {
	url := fmt.Sprintf(embedURL, c.apiKey)
	body, _ := json.Marshal(embedRequest{
		Model:    "models/text-embedding-004",
		Content:  content{Parts: []part{{Text: text}}},
		TaskType: "SEMANTIC_SIMILARITY",
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody map[string]any
		json.NewDecoder(resp.Body).Decode(&errBody)
		return nil, fmt.Errorf("gemini embed returned %d: %v", resp.StatusCode, errBody)
	}

	var result embedResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if len(result.Embedding.Values) == 0 {
		return nil, fmt.Errorf("gemini returned empty embedding")
	}
	return result.Embedding.Values, nil
}

func (c *Client) Generate(ctx context.Context, prompt, systemPrompt string) (string, error) {
	url := fmt.Sprintf(generateURL, c.apiKey)
	body, _ := json.Marshal(generateRequest{
		Contents:          []content{{Parts: []part{{Text: prompt}}}},
		SystemInstruction: &content{Parts: []part{{Text: systemPrompt}}},
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody map[string]any
		json.NewDecoder(resp.Body).Decode(&errBody)
		return "", fmt.Errorf("gemini generate returned %d: %v", resp.StatusCode, errBody)
	}

	var result generateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("empty response from Gemini")
	}
	return result.Candidates[0].Content.Parts[0].Text, nil
}
