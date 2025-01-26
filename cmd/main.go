package main

import (
	"context"

	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/sashabaranov/go-openai"
	"github.com/velumlabs/hana/internal/twitter"
	"github.com/velumlabs/thor/llm"
	"github.com/velumlabs/thor/logger"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	// Initialize logger
	log, err := logger.New(&logger.Config{
		Level:      "info",
		TreeFormat: true,
		TimeFormat: "2006-01-02 15:04:05",
		UseColors:  true,
	})
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}

	// Create context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize database
	db, err := gorm.Open(postgres.Open(os.Getenv("DB_URL")), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Initialize LLM client
	llmClient, err := llm.NewLLMClient(llm.Config{
		ProviderType: llm.ProviderOpenAI,
		APIKey:       os.Getenv("OPENAI_API_KEY"),
		ModelConfig: map[llm.ModelType]string{
			llm.ModelTypeFast:     openai.GPT4oMini,
			llm.ModelTypeDefault:  openai.GPT4oMini,
			llm.ModelTypeAdvanced: openai.GPT4o,
		},
		Logger:  log.NewSubLogger("llm", &logger.SubLoggerOpts{}),
		Context: ctx,
	})

	// Create Twitter instance with options
	k, err := twitter.New(
		twitter.WithContext(ctx),
		twitter.WithLogger(log.NewSubLogger("thor", &logger.SubLoggerOpts{})),
		twitter.WithDatabase(db),
		twitter.WithLLM(llmClient),
		twitter.WithTwitterMonitorInterval(
			60*time.Second,  // min interval
			120*time.Second, // max interval
		),
		twitter.WithTwitterCredentials(
			os.Getenv("TWITTER_CT0"),
			os.Getenv("TWITTER_AUTH_TOKEN"),
			os.Getenv("TWITTER_USER"),
		),
	)
	if err != nil {
		log.Fatalf("Failed to create thor: %v", err)
	}

	// Start zen
	if err := k.Start(); err != nil {
		log.Fatalf("Failed to start thor: %v", err)
	}

	// Wait for interrupt signal
	<-ctx.Done()

	// Stop zen gracefully
	if err := k.Stop(); err != nil {
		log.Errorf("Error stopping thor: %v", err)
	}
}
