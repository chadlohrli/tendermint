package types

import (
	"errors"
	"io/ioutil"

	uuid "github.com/kthomas/go.uuid"
	"github.com/providenetwork/tendermint/crypto"
	"github.com/providenetwork/tendermint/crypto/ed25519"
	tmjson "github.com/providenetwork/tendermint/libs/json"
	tmos "github.com/providenetwork/tendermint/libs/os"
	"github.com/provideplatform/provide-go/api/ident"
	"github.com/provideplatform/provide-go/api/vault"
)

//------------------------------------------------------------------------------
// Persistent peer ID
// TODO: encrypt on disk

// NodeKey is the persistent peer key.
// It contains the nodes private key for authentication.
type NodeKey struct {
	// Canonical ID - hex-encoded pubkey's address (IDByteLength bytes)
	ID NodeID `json:"id"`
	// Private key
	PrivKey crypto.PrivKey `json:"priv_key"`
}

// PubKey returns the peer's PubKey
func (nodeKey NodeKey) PubKey() crypto.PubKey {
	return nodeKey.PrivKey.PubKey()
}

// SaveAs persists the NodeKey to filePath.
func (nodeKey NodeKey) SaveAs(filePath string) error {
	jsonBytes, err := tmjson.Marshal(nodeKey)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filePath, jsonBytes, 0600)
}

// LoadOrGenNodeKey attempts to load the NodeKey from the given filePath. If
// the file does not exist, it generates and saves a new NodeKey.
func LoadOrGenNodeKey(filePath, vaultRefreshToken string, vaultID, vaultKeyID *uuid.UUID) (NodeKey, error) {
	if tmos.FileExists(filePath) {
		nodeKey, err := LoadNodeKey(filePath)
		if err != nil {
			return NodeKey{}, err
		}
		return nodeKey, nil
	}

	var nodeKey NodeKey

	if vaultRefreshToken != "" && vaultID != nil && vaultKeyID == nil {
		nodeKey = GenVaultedNodeKey(vaultRefreshToken, *vaultID)
	} else if vaultRefreshToken != "" && vaultID != nil && vaultKeyID != nil {
		nk := FetchVaultedNodeKey(vaultRefreshToken, *vaultID, *vaultKeyID)
		if nk == nil {
			return NodeKey{}, errors.New("failed to fetch vaulted node key")
		}
		nodeKey = *nk
	} else {
		nodeKey = GenNodeKey()

		if err := nodeKey.SaveAs(filePath); err != nil {
			return NodeKey{}, err
		}
	}

	return nodeKey, nil
}

// GenNodeKey generates a new node key.
func GenNodeKey() NodeKey {
	privKey := ed25519.GenPrivKey()
	return NodeKey{
		ID:      NodeIDFromPubKey(privKey.PubKey()),
		PrivKey: privKey,
	}
}

// GenVaultedNodeKey generates a new node key using the configured vault.
func GenVaultedNodeKey(vaultRefreshToken string, vaultID uuid.UUID) NodeKey {
	privKey := ed25519.GenVaultedPrivKey(vaultRefreshToken, vaultID)
	return NodeKey{
		ID:      NodeIDFromPubKey(privKey.PubKey()),
		PrivKey: privKey,
	}
}

// FetchVaultedNodeKey fetches an existing node key using the configured vault and vault key id
func FetchVaultedNodeKey(vaultRefreshToken string, vaultID, vaultKeyID uuid.UUID) *NodeKey {
	token, err := ident.CreateToken(vaultRefreshToken, map[string]interface{}{
		"grant_type": "refresh_token",
	})
	if err != nil {
		return nil
	}

	resp, err := vault.FetchKey(*token.AccessToken, vaultID.String(), vaultKeyID.String())
	if err != nil {
		return nil
	}

	privKey := &ed25519.VaultedPrivateKey{
		VaultID:           vaultID,
		VaultKeyID:        resp.ID,
		VaultRefreshToken: vaultRefreshToken,
	}

	return &NodeKey{
		ID:      NodeIDFromPubKey(privKey.PubKey()),
		PrivKey: privKey,
	}
}

// LoadNodeKey loads NodeKey located in filePath.
func LoadNodeKey(filePath string) (NodeKey, error) {
	jsonBytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		return NodeKey{}, err
	}
	nodeKey := NodeKey{}
	err = tmjson.Unmarshal(jsonBytes, &nodeKey)
	if err != nil {
		return NodeKey{}, err
	}
	nodeKey.ID = NodeIDFromPubKey(nodeKey.PubKey())
	return nodeKey, nil
}
