package ed25519

import (
	"bytes"
	"encoding/hex"
	"fmt"

	uuid "github.com/kthomas/go.uuid"
	"github.com/providenetwork/tendermint/crypto"
	"github.com/providenetwork/tendermint/crypto/tmhash"
	tmjson "github.com/providenetwork/tendermint/libs/json"
	"github.com/provideplatform/provide-go/api/ident"
	"github.com/provideplatform/provide-go/api/vault"
	"github.com/provideplatform/provide-go/common"
)

const VaultedPrivateKeyName = "tendermint/VaultedPrivKeyEd25519"
const VaultedPublicKeyName = "tendermint/VaultedPubKeyEd25519"

var _ crypto.PrivKey = &VaultedPrivateKey{}
var _ crypto.PubKey = &VaultedPublicKey{}

func init() {
	tmjson.RegisterType(VaultedPrivateKey{}, VaultedPrivateKeyName)
	tmjson.RegisterType(VaultedPublicKey{}, VaultedPublicKeyName)
}

func authorizeAccessToken(refreshToken *string) (*string, error) {
	resp, err := ident.CreateToken(*refreshToken, map[string]interface{}{
		"grant_type": "refresh_token",
	})
	if err != nil {
		return nil, err
	}

	return resp.AccessToken, nil
}

type VaultedPrivateKey struct {
	VaultID           uuid.UUID `json:"vault_id"`
	VaultKeyID        uuid.UUID `json:"vault_key_id"`
	VaultRefreshToken string    `json:"-"`

	PublicKey []byte `json:"public_key"`
}

func (k *VaultedPrivateKey) Bytes() []byte {
	return nil
}

func (k *VaultedPrivateKey) Equals(other crypto.PrivKey) bool {
	return bytes.Equal(k.PubKey().Bytes(), other.PubKey().Bytes())
}

func (k *VaultedPrivateKey) PubKey() crypto.PubKey {
	if k.PublicKey != nil && len(k.PublicKey) > 0 {
		return &VaultedPublicKey{
			VaultRefreshToken: k.VaultRefreshToken,
			VaultID:           k.VaultID,
			VaultKeyID:        k.VaultKeyID,
			PublicKey:         k.PublicKey,
		}
	}

	token, err := authorizeAccessToken(&k.VaultRefreshToken)
	if err != nil {
		return nil
	}

	resp, err := vault.FetchKey(
		*token,
		k.VaultID.String(),
		string(k.VaultKeyID.String()),
	)
	if err != nil {
		common.Log.Warningf("failed to fetch vault key: %s; %s", k.VaultKeyID, err.Error())
		return nil
	}

	k.PublicKey, _ = hex.DecodeString((*resp.PublicKey)[2:])

	return &VaultedPublicKey{
		VaultRefreshToken: k.VaultRefreshToken,
		VaultID:           k.VaultID,
		VaultKeyID:        k.VaultKeyID,
		PublicKey:         k.PublicKey,
	}
}

func (k *VaultedPrivateKey) Sign(msg []byte) ([]byte, error) {
	token, err := authorizeAccessToken(&k.VaultRefreshToken)
	if err != nil {
		return nil, err
	}

	resp, err := vault.SignMessage(
		*token,
		k.VaultID.String(),
		string(k.VaultKeyID.String()),
		hex.EncodeToString(msg),
		map[string]interface{}{},
	)
	if err != nil {
		common.Log.Warningf("failed to sign using vault key: %s; %s", k.VaultKeyID, err.Error())
		return nil, err
	}

	return hex.DecodeString(*resp.Signature)
}

func (k *VaultedPrivateKey) Type() string {
	return KeyType
}

type VaultedPublicKey struct {
	VaultID           uuid.UUID `json:"vault_id"`
	VaultKeyID        uuid.UUID `json:"vault_key_id"`
	VaultRefreshToken string    `json:"-"`

	PublicKey []byte `json:"public_key"`
}

func (k *VaultedPublicKey) Address() crypto.Address {
	addr := crypto.Address(tmhash.SumTruncated(k.Bytes()))
	return addr
}

func (k *VaultedPublicKey) Bytes() []byte {
	if k.PublicKey != nil && len(k.PublicKey) > 0 {
		return k.PublicKey
	}

	token, err := authorizeAccessToken(&k.VaultRefreshToken)
	if err != nil {
		return nil
	}

	resp, err := vault.FetchKey(
		*token,
		k.VaultID.String(),
		string(k.VaultKeyID.String()),
	)
	if err != nil {
		common.Log.Warningf("failed to fetch vault key: %s; %s", k.VaultKeyID, err.Error())
		return nil
	}

	k.PublicKey, _ = hex.DecodeString((*resp.PublicKey)[2:])
	return k.PublicKey
}

func (k *VaultedPublicKey) Equals(other crypto.PubKey) bool {
	return bytes.Equal(k.Bytes(), other.Bytes())
}

func (k *VaultedPublicKey) String() string {
	return fmt.Sprintf("PubKeyEd25519{%X}", k.Bytes())
}

func (k *VaultedPublicKey) Type() string {
	return KeyType
}

func (k *VaultedPublicKey) VerifySignature(msg, sig []byte) bool {
	token, err := authorizeAccessToken(&k.VaultRefreshToken)
	if err != nil {
		return false
	}

	resp, err := vault.VerifySignature(
		*token,
		k.VaultID.String(),
		k.VaultKeyID.String(),
		hex.EncodeToString(msg),
		hex.EncodeToString(sig),
		map[string]interface{}{},
	)
	if err != nil {
		common.Log.Warningf("failed to verify signature vault key: %s; %s", k.VaultKeyID, err.Error())
		return false
	}

	return resp.Verified
}
