package embedding

import (
	"context"
	"encoding/json"
	"fmt"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"

	"github.com/maraichr/codegraph/internal/config"
)

const maxBatchSize = 96 // Cohere embed API limit

// Client wraps the AWS Bedrock runtime for embedding generation.
type Client struct {
	bedrock *bedrockruntime.Client
	modelID string
}

// NewClient creates a new Bedrock embedding client.
func NewClient(cfg config.BedrockConfig) (*Client, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(cfg.Region),
	)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	client := bedrockruntime.NewFromConfig(awsCfg)
	return &Client{bedrock: client, modelID: cfg.ModelID}, nil
}

// cohereEmbedRequest is the Cohere Embed v4 API request format.
type cohereEmbedRequest struct {
	Texts     []string `json:"texts"`
	InputType string   `json:"input_type"`
}

// cohereEmbedResponse is the Cohere Embed v4 API response format.
type cohereEmbedResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
}

// EmbedBatch generates embeddings for a batch of texts.
// Automatically splits into sub-batches of maxBatchSize.
func (c *Client) EmbedBatch(ctx context.Context, texts []string, inputType string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	var allEmbeddings [][]float32

	for i := 0; i < len(texts); i += maxBatchSize {
		end := min(i+maxBatchSize, len(texts))
		batch := texts[i:end]

		embeddings, err := c.embedSingle(ctx, batch, inputType)
		if err != nil {
			return nil, fmt.Errorf("embed batch %d: %w", i/maxBatchSize, err)
		}
		allEmbeddings = append(allEmbeddings, embeddings...)
	}

	return allEmbeddings, nil
}

func (c *Client) embedSingle(ctx context.Context, texts []string, inputType string) ([][]float32, error) {
	reqBody, err := json.Marshal(cohereEmbedRequest{
		Texts:     texts,
		InputType: inputType,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.bedrock.InvokeModel(ctx, &bedrockruntime.InvokeModelInput{
		ModelId:     &c.modelID,
		ContentType: strPtr("application/json"),
		Body:        reqBody,
	})
	if err != nil {
		return nil, fmt.Errorf("invoke model: %w", err)
	}

	var result cohereEmbedResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return result.Embeddings, nil
}

// ModelID returns the Bedrock model identifier.
func (c *Client) ModelID() string { return c.modelID }

func strPtr(s string) *string { return &s }
