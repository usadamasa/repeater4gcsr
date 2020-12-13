package repeater4gcsr

import "testing"

func TestSecrets(t *testing.T) {
	data, err := getPrivateKey("usadamasa-dev3", "repeater4gcsr-bitbucket-key")
	if err != nil {
		t.Errorf("getPrivateKey returned err %s", err)
		t.Fail()
	}
	if data == nil {
		t.Errorf("getPrivateKey data is null")
		t.Fail()
	}
}
