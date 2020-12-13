package repeater4gcsr

import (
	"context"
	"fmt"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	ssh2 "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"golang.org/x/crypto/ssh"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
)

func GetGCSRAuth(projectName string, gcsrSshUser string) (*ssh2.PublicKeys, error) {
	return getAuth(projectName, "repeater4gcsr-gcsr-key", gcsrSshUser)
}

func GetBitBcuketAuth(projectName string) (*ssh2.PublicKeys, error) {
	return getAuth(projectName, "repeater4gcsr-bitbucket-key", "git")
}

func getAuth(projectName string, keyname string, sshUser string) (*ssh2.PublicKeys, error) {
	key, err := getPrivateKey(projectName, keyname)
	if err != nil {
		sugar.Errorf("getPrivateKey keyname:%s, %s\n", keyname, err)
		return nil, err
	}

	signer, _ := ssh.ParsePrivateKey(key)
	auth := &ssh2.PublicKeys{
		User:   sshUser,
		Signer: signer,
		HostKeyCallbackHelper: ssh2.HostKeyCallbackHelper{
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		},
	}
	return auth, nil
}

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
