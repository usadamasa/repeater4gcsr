package repeater4gcsr

import (
	"context"
	"fmt"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
)

func getPrivateKey(projectName string, name string) ([]byte, error) {
	ctx := context.Background()
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		sugar.Errorf("failed to create secretmanager client: %v", err)
		return nil, err
	}

	// Build the request.
	getSecretReq := &secretmanagerpb.AccessSecretVersionRequest{
		Name: fmt.Sprintf("projects/%s/secrets/%s/versions/latest", projectName, name),
	}

	// Call the API.
	result, err := client.AccessSecretVersion(ctx, getSecretReq)
	if err != nil {
		sugar.Errorf("failed to access secret version: %v", err)
		return nil, err
	}
	return result.Payload.Data, nil
}
