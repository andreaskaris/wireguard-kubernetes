package wireguard

import (
	"fmt"
	"path"
	"testing"
)

func TestEnsureWireguardKeys(t *testing.T) {
	tempDir := t.TempDir()

	tcs := []struct {
		pubKey      string
		privKey     string
		expectError bool
	}{
		{
			pubKey:      "public",
			privKey:     "private",
			expectError: false,
		}, {
			pubKey:      "subdir/public",
			privKey:     "subdir/private",
			expectError: true,
		},
	}

	for _, tc := range tcs {
		pubKey := path.Join(tempDir, tc.pubKey)
		privKey := path.Join(tempDir, tc.privKey)

		err := EnsureWireguardKeys(pubKey, privKey)
		if !tc.expectError && err != nil {
			t.Fatal(fmt.Sprintf("EnsureWireguardKeys(%s, %s): Got error %s", pubKey, privKey, err))
		}
		if tc.expectError && err == nil {
			t.Fatal(fmt.Sprintf("EnsureWireguardKeys(%s, %s): Should return an error, but got nil", pubKey, privKey))
		}
	}
}
