// embedtest makes a single OpenRouter embedding request using config from env (and .env if present).
// Run from project root: go run ./cmd/embedtest
// Verbose (raw HTTP): go run ./cmd/embedtest -verbose
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/maraichr/codegraph/internal/config"
	"github.com/maraichr/codegraph/internal/embedding"
)

func main() {
	verbose := flag.Bool("verbose", false, "print raw HTTP request and response")
	modelOverride := flag.String("model", "", "override OPENROUTER_MODEL for this run (e.g. openai/text-embedding-3-small)")
	flag.Parse()

	_ = godotenv.Load(".env") // ignore error if .env missing

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	if cfg.OpenRouter.APIKey == "" {
		log.Fatal("OPENROUTER_API_KEY is not set (set it in .env or environment)")
	}

	baseURL := cfg.OpenRouter.BaseURLEmbeddings
	if baseURL == "" {
		baseURL = cfg.OpenRouter.BaseURL
	}
	if baseURL == "" {
		baseURL = "https://openrouter.ai/api/v1/embeddings"
	}
	baseURL = strings.TrimRight(baseURL, "/")
	if baseURL == "https://openrouter.ai" || baseURL == "https://openrouter.ai/api/v1" {
		baseURL = "https://openrouter.ai/api/v1/embeddings"
	}

	model := cfg.OpenRouter.Model
	if *modelOverride != "" {
		model = *modelOverride
	}
	if model == "" {
		model = "openai/text-embedding-3-small"
	}
	dims := cfg.OpenRouter.Dimensions
	if dims <= 0 {
		dims = 1024
	}

	if *verbose {
		doVerbose(cfg.OpenRouter.APIKey, baseURL, model, dims)
		return
	}

	orc := cfg.OpenRouter
	orc.Model = model
	client, err := embedding.NewOpenRouterClient(orc)
	if err != nil {
		log.Fatalf("openrouter client: %v", err)
	}

	fmt.Printf("Model: %s\n", client.ModelID())
	fmt.Printf("URL: %s\n", baseURL)
	fmt.Println("Sending one embedding request...")

	ctx := context.Background()
	texts := []string{"The quick brown fox jumps over the lazy dog."}
	vecs, err := client.EmbedBatch(ctx, texts, "search_document")
	if err != nil {
		fmt.Fprintf(os.Stderr, "EmbedBatch error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("OK: got %d embedding(s), dims=%d\n", len(vecs), len(vecs[0]))
}

func doVerbose(apiKey, baseURL, model string, dimensions int) {
	body := map[string]any{
		"model":           model,
		"input":           []string{"The quick brown fox jumps over the lazy dog."},
		"encoding_format": "float",
		"provider":        map[string]bool{"allow_fallbacks": true},
	}
	if strings.HasPrefix(model, "openai/") || strings.HasPrefix(model, "qwen/") {
		body["dimensions"] = dimensions
	}
	raw, _ := json.MarshalIndent(body, "", "  ")
	fmt.Println("--- Request ---")
	fmt.Printf("POST %s\n", baseURL)
	fmt.Printf("Body:\n%s\n", raw)

	req, err := http.NewRequest(http.MethodPost, baseURL, bytes.NewReader(raw))
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("HTTP request: %v", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	fmt.Println("--- Response ---")
	fmt.Printf("Status: %d %s\n", resp.StatusCode, resp.Status)
	fmt.Printf("Body:\n%s\n", respBody)
	if resp.StatusCode != http.StatusOK {
		os.Exit(1)
	}
}
