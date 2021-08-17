package p2p

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"

	uuid "github.com/kthomas/go.uuid"
	"github.com/providenetwork/tendermint/crypto"
	"github.com/providenetwork/tendermint/crypto/ed25519"
	tmjson "github.com/providenetwork/tendermint/libs/json"
	tmos "github.com/providenetwork/tendermint/libs/os"
	"github.com/provideplatform/provide-go/api/ident"
	"github.com/provideplatform/provide-go/api/vault"
)

// ID is a hex-encoded crypto.Address
type ID string

// IDByteLength is the length of a crypto.Address. Currently only 20.
// TODO: support other length addresses ?
const IDByteLength = crypto.AddressSize

//------------------------------------------------------------------------------
// Persistent peer ID
// TODO: encrypt on disk

// NodeKey is the persistent peer key.
// It contains the nodes private key for authentication.
type NodeKey struct {
	PrivKey crypto.PrivKey `json:"priv_key"` // our priv key
}

// ID returns the peer's canonical ID - the hash of its public key.
func (nodeKey *NodeKey) ID() ID {
	return PubKeyToID(nodeKey.PubKey())
}

// PubKey returns the peer's PubKey
func (nodeKey *NodeKey) PubKey() crypto.PubKey {
	return nodeKey.PrivKey.PubKey()
}

// PubKeyToID returns the ID corresponding to the given PubKey.
// It's the hex-encoding of the pubKey.Address().
func PubKeyToID(pubKey crypto.PubKey) ID {
	return ID(hex.EncodeToString(pubKey.Address()))
}

// GenNodeKey generates a new node key.
func GenNodeKey() NodeKey {
	privKey := ed25519.GenPrivKey()
	return NodeKey{
		// ID:      p2p.NodeIDFromPubKey(privKey.PubKey()),
		PrivKey: privKey,
	}
}

// GenVaultedNodeKey generates a new node key using the configured vault.
func GenVaultedNodeKey(vaultRefreshToken string, vaultID uuid.UUID) NodeKey {
	privKey := ed25519.GenVaultedPrivKey(vaultRefreshToken, vaultID)
	return NodeKey{
		// ID:      p2p.NodeIDFromPubKey(privKey.PubKey()),
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
		// ID:      NodeIDFromPubKey(privKey.PubKey()),
		PrivKey: privKey,
	}
}

// LoadOrGenNodeKey attempts to load the NodeKey from the given filePath. If
// the file does not exist, it generates and saves a new NodeKey.
func LoadOrGenNodeKey(filePath, vaultRefreshToken string, vaultID, vaultKeyID *uuid.UUID) (*NodeKey, error) {
	if tmos.FileExists(filePath) {
		nodeKey, err := LoadNodeKey(filePath)
		if err != nil {
			return &NodeKey{}, err
		}
		return nodeKey, nil
	}

	var nodeKey NodeKey

	if vaultRefreshToken != "" && vaultID != nil && vaultKeyID == nil {
		nodeKey = GenVaultedNodeKey(vaultRefreshToken, *vaultID)
	} else if vaultRefreshToken != "" && vaultID != nil && vaultKeyID != nil {
		nk := FetchVaultedNodeKey(vaultRefreshToken, *vaultID, *vaultKeyID)
		if nk == nil {
			return &NodeKey{}, errors.New("failed to fetch vaulted node key")
		}
		nodeKey = *nk
	} else {
		nodeKey = GenNodeKey()

		if err := nodeKey.SaveAs(filePath); err != nil {
			return &NodeKey{}, err
		}
	}

	return &nodeKey, nil
}

// LoadNodeKey loads NodeKey located in filePath.
func LoadNodeKey(filePath string) (*NodeKey, error) {
	jsonBytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	nodeKey := new(NodeKey)
	err = tmjson.Unmarshal(jsonBytes, nodeKey)
	if err != nil {
		return nil, err
	}
	return nodeKey, nil
}

// SaveAs persists the NodeKey to filePath.
func (nodeKey *NodeKey) SaveAs(filePath string) error {
	jsonBytes, err := tmjson.Marshal(nodeKey)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filePath, jsonBytes, 0600)
	if err != nil {
		return err
	}
	return nil
}

//------------------------------------------------------------------------------

// MakePoWTarget returns the big-endian encoding of 2^(targetBits - difficulty) - 1.
// It can be used as a Proof of Work target.
// NOTE: targetBits must be a multiple of 8 and difficulty must be less than targetBits.
func MakePoWTarget(difficulty, targetBits uint) []byte {
	if targetBits%8 != 0 {
		panic(fmt.Sprintf("targetBits (%d) not a multiple of 8", targetBits))
	}
	if difficulty >= targetBits {
		panic(fmt.Sprintf("difficulty (%d) >= targetBits (%d)", difficulty, targetBits))
	}
	targetBytes := targetBits / 8
	zeroPrefixLen := (int(difficulty) / 8)
	prefix := bytes.Repeat([]byte{0}, zeroPrefixLen)
	mod := (difficulty % 8)
	if mod > 0 {
		nonZeroPrefix := byte(1<<(8-mod) - 1)
		prefix = append(prefix, nonZeroPrefix)
	}
	tailLen := int(targetBytes) - len(prefix)
	return append(prefix, bytes.Repeat([]byte{0xFF}, tailLen)...)
}
