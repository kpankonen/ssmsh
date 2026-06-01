package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

func LoadConfig(region, profile string) aws.Config {
	opts := []func(*config.LoadOptions) error{
		config.WithRegion(region),
	}

	// Only force a shared config profile when one was explicitly requested.
	// Forcing a profile (e.g. "default") makes the SDK resolve credentials
	// from that profile before it considers credentials supplied via
	// environment variables. That breaks credential injection from tools like
	// Teleport (tsh aws --exec), which set AWS_ACCESS_KEY_ID/AWS_SECRET_ACCESS_KEY
	// directly. Leaving the profile unset lets the SDK's normal chain prefer
	// those env credentials.
	if profile != "" {
		opts = append(opts, config.WithSharedConfigProfile(profile))
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(), opts...)
	if err != nil {
		panic("unable to load AWS config: " + err.Error())
	}
	return cfg
}
