package utils

import (
	"log"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// Config holds the application's configuration values.
type Config struct {
	CORSOrigin     string
	Environment    string
	HTTPPort       int
	HTTPHost       string
	NgrokAuthToken string
	S3AccessKey    string
	S3BucketName   string
	S3Endpoint     string
	S3SecretKey    string
	S3Region       string
}

// LoadConfig initializes and returns the configuration using environment variables.
func LoadConfig() (*Config, error) {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found or unable to load it.")
	}

	// Set up default values
	viper.SetDefault("CORS_ORIGIN", "*")
	viper.SetDefault("NODE_ENV", "dev")
	viper.SetDefault("PORT", 8080)
	viper.SetDefault("HOST", "http://localhost")
	viper.SetDefault("S3_ACCESS_KEY_ID", "")
	viper.SetDefault("S3_BUCKET_NAME", "")
	viper.SetDefault("S3_ENDPOINT", "")
	viper.SetDefault("S3_SECRET_ACCESS_KEY", "")
	viper.SetDefault("S3_REGION", "us-east-1")

	// Bind environment variables
	viper.BindEnv("CORS_ORIGIN")
	viper.BindEnv("NODE_ENV")
	viper.BindEnv("PORT")
	viper.BindEnv("HOST")
	viper.BindEnv("S3_ACCESS_KEY_ID")
	viper.BindEnv("S3_BUCKET_NAME")
	viper.BindEnv("S3_ENDPOINT")
	viper.BindEnv("S3_SECRET_ACCESS_KEY")
	viper.BindEnv("S3_REGION")

	// Read environment variables
	viper.AutomaticEnv()

	// Initialize config values from environment or defaults
	config := &Config{
		CORSOrigin:     viper.GetString("CORS_ORIGIN"),
		Environment:    viper.GetString("NODE_ENV"),
		HTTPPort:       viper.GetInt("PORT"),
		HTTPHost:       viper.GetString("HOST"),
		NgrokAuthToken: viper.GetString("NGROK_AUTHTOKEN"),
		S3AccessKey:    viper.GetString("S3_ACCESS_KEY_ID"),
		S3BucketName:   viper.GetString("S3_BUCKET_NAME"),
		S3Endpoint:     viper.GetString("S3_ENDPOINT"),
		S3SecretKey:    viper.GetString("S3_SECRET_ACCESS_KEY"),
		S3Region:       viper.GetString("S3_REGION"),
	}

	// Validate required fields (example: checking if S3 keys are set)
	if config.S3AccessKey == "" || config.S3SecretKey == "" || config.S3BucketName == "" {
		log.Println("Warning: Some required S3 configurations are missing.")
	}

	return config, nil
}
