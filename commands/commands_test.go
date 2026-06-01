package commands

import (
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

func TestFormatPlainCleansSDKPointerFields(t *testing.T) {
	result := []types.Parameter{
		{
			Name:    aws.String("/develop/wiz/eks-client-id"),
			Type:    types.ParameterTypeSecureString,
			Value:   aws.String("client-id"),
			Version: 2,
		},
	}

	cleaned, err := cleanResult(result)
	if err != nil {
		t.Fatal(err)
	}
	got := formatPlain(cleaned, 0)

	for _, want := range []string{
		`Name: "/develop/wiz/eks-client-id"`,
		`Type: "SecureString"`,
		`Value: "client-id"`,
		`Version: 2`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected formatted output to contain %q, got %s", want, got)
		}
	}
	if strings.Contains(got, "0x") {
		t.Fatalf("expected formatted output not to contain pointer addresses, got %s", got)
	}
}
