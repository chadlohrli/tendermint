package privval

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	uuid "github.com/kthomas/go.uuid"
	"github.com/providenetwork/tendermint/crypto"
	"github.com/providenetwork/tendermint/crypto/ed25519"
	tmbytes "github.com/providenetwork/tendermint/libs/bytes"
	tmjson "github.com/providenetwork/tendermint/libs/json"
	tmos "github.com/providenetwork/tendermint/libs/os"
	tmproto "github.com/providenetwork/tendermint/proto/tendermint/types"
	"github.com/providenetwork/tendermint/types"
)

const ValidatorName = "tendermint/Validator"

var _ types.PrivValidator = (*Validator)(nil)

func init() {
	tmjson.RegisterType(Validator{}, ValidatorName)
}

type Validator struct {
	Address types.Address              `json:"address"`
	PubKey  *ed25519.VaultedPublicKey  `json:"pub_key"`
	PrivKey *ed25519.VaultedPrivateKey `json:"priv_key"`

	ConfigPath    string         `json:"config_path"`
	StatePath     string         `json:"state_path"`
	LastSignState *LastSignState `json:"-"`

	VaultID           uuid.UUID `json:"vault_id"`
	VaultKeyID        uuid.UUID `json:"vault_key_id"`
	VaultRefreshToken string    `json:"-"`
}

type LastSignState struct {
	Height    int64            `json:"height"`
	Round     int32            `json:"round"`
	Step      int8             `json:"step"`
	Signature []byte           `json:"signature,omitempty"`
	SignBytes tmbytes.HexBytes `json:"signbytes,omitempty"`

	StatePath string `json:"-"`
}

func LoadOrGenValidator(configRoot, vaultRefreshToken string, vaultID uuid.UUID, vaultKeyID *uuid.UUID) (*Validator, error) {
	var v *Validator

	cfgPath := fmt.Sprintf("%s%svalidator.json", configRoot, string(os.PathSeparator))
	statePath := fmt.Sprintf("%s%svalidator-state.json", configRoot, string(os.PathSeparator))

	if tmos.FileExists(cfgPath) {
		raw, err := ioutil.ReadFile(cfgPath)
		if err != nil {
			return nil, err
		}
		err = tmjson.Unmarshal(raw, &v)
		if err != nil {
			return nil, err
		}
		v.ConfigPath = cfgPath

		raw, err = ioutil.ReadFile(statePath)
		if err == nil {
			err = tmjson.Unmarshal(raw, &v.LastSignState)
			if err != nil {
				return nil, err
			}
			v.LastSignState.StatePath = statePath
		}
	} else {
		var privKey *ed25519.VaultedPrivateKey
		if vaultKeyID != nil {
			privKey = ed25519.LoadVaultedPrivKey(vaultRefreshToken, vaultID, *vaultKeyID)
		} else {
			privKey = ed25519.GenVaultedPrivKey(vaultRefreshToken, vaultID)
		}

		val, err := ValidatorFactory(cfgPath, statePath, vaultRefreshToken, vaultID, privKey.VaultKeyID)
		if err != nil {
			return nil, err
		}
		v = val
		v.Save()
	}

	if v.LastSignState == nil {
		v.LastSignState = &LastSignState{
			StatePath: statePath,
		}
	}

	return v, nil
}

func ValidatorFactory(cfgPath, statePath, token string, vaultID, vaultKeyID uuid.UUID) (*Validator, error) {
	v := &Validator{
		ConfigPath:        cfgPath,
		StatePath:         statePath,
		VaultID:           vaultID,
		VaultKeyID:        vaultKeyID,
		VaultRefreshToken: token,
	}

	v.PrivKey = &ed25519.VaultedPrivateKey{
		VaultID:           v.VaultID,
		VaultKeyID:        v.VaultKeyID,
		VaultRefreshToken: v.VaultRefreshToken,
	}

	v.PubKey = v.PrivKey.PubKey().(*ed25519.VaultedPublicKey)
	v.Address = v.PubKey.Address()

	return v, nil
}

func (v *Validator) GetPubKey() (crypto.PubKey, error) {
	return &ed25519.VaultedPublicKey{
		VaultID:           v.VaultID,
		VaultKeyID:        v.VaultKeyID,
		VaultRefreshToken: v.VaultRefreshToken,
	}, nil
}

func (v *Validator) SignProposal(chainID string, proposal *tmproto.Proposal) error {
	height, round, step := proposal.Height, proposal.Round, stepPropose

	lss := v.LastSignState

	sameHRS, err := lss.CheckHRS(height, round, step)
	if err != nil {
		return err
	}

	signBytes := types.ProposalSignBytes(chainID, proposal)

	// We might crash before writing to the wal,
	// causing us to try to re-sign for the same HRS.
	// If signbytes are the same, use the last signature.
	// If they only differ by timestamp, use last timestamp and signature
	// Otherwise, return error
	if sameHRS {
		if bytes.Equal(signBytes, lss.SignBytes) {
			proposal.Signature = lss.Signature
		} else if timestamp, ok := checkProposalsOnlyDifferByTimestamp(lss.SignBytes, signBytes); ok {
			proposal.Timestamp = timestamp
			proposal.Signature = lss.Signature
		} else {
			err = fmt.Errorf("conflicting data")
		}
		return err
	}

	// It passed the checks. Sign the proposal
	sig, err := v.PrivKey.Sign(signBytes)
	if err != nil {
		return err
	}
	v.saveSigned(height, round, step, signBytes, sig)
	proposal.Signature = sig
	return nil
}

func (v *Validator) SignVote(chainID string, vote *tmproto.Vote) error {
	height, round, step := vote.Height, vote.Round, voteToStep(vote)

	lss := v.LastSignState

	sameHRS, err := lss.CheckHRS(height, round, step)
	if err != nil {
		return err
	}

	signBytes := types.VoteSignBytes(chainID, vote)

	// We might crash before writing to the wal,
	// causing us to try to re-sign for the same HRS.
	// If signbytes are the same, use the last signature.
	// If they only differ by timestamp, use last timestamp and signature
	// Otherwise, return error
	if sameHRS {
		if bytes.Equal(signBytes, lss.SignBytes) {
			vote.Signature = lss.Signature
		} else if timestamp, ok := checkVotesOnlyDifferByTimestamp(lss.SignBytes, signBytes); ok {
			vote.Timestamp = timestamp
			vote.Signature = lss.Signature
		} else {
			err = fmt.Errorf("conflicting data")
		}
		return err
	}

	// It passed the checks. Sign the vote
	sig, err := v.PrivKey.Sign(signBytes)
	if err != nil {
		return err
	}
	v.saveSigned(height, round, step, signBytes, sig)
	vote.Signature = sig
	return nil
}

func (v *Validator) Save() {
	jsonBytes, err := tmjson.MarshalIndent(v, "", "  ")
	if err != nil {
		panic(err)
	}
	err = ioutil.WriteFile(v.ConfigPath, jsonBytes, 0600)
	if err != nil {
		panic(err)
	}
}

func (v *Validator) saveSigned(
	height int64,
	round int32,
	step int8,
	signBytes []byte,
	sig []byte,
) {
	v.LastSignState.Height = height
	v.LastSignState.Round = round
	v.LastSignState.Step = step
	v.LastSignState.Signature = sig
	v.LastSignState.SignBytes = signBytes
	v.LastSignState.Save()
}

func (lss *LastSignState) CheckHRS(height int64, round int32, step int8) (bool, error) {
	if lss.Height > height {
		return false, fmt.Errorf("height regression. Got %v, last height %v", height, lss.Height)
	}

	if lss.Height == height {
		if lss.Round > round {
			return false, fmt.Errorf("round regression at height %v. Got %v, last round %v", height, round, lss.Round)
		}

		if lss.Round == round {
			if lss.Step > step {
				return false, fmt.Errorf(
					"step regression at height %v round %v. Got %v, last step %v",
					height,
					round,
					step,
					lss.Step,
				)
			} else if lss.Step == step {
				if lss.SignBytes != nil {
					if lss.Signature == nil {
						panic("pv: Signature is nil but SignBytes is not!")
					}
					return true, nil
				}
				return false, errors.New("no SignBytes found")
			}
		}
	}
	return false, nil
}

func (lss *LastSignState) Save() {
	jsonBytes, err := tmjson.MarshalIndent(lss, "", "  ")
	if err != nil {
		panic(err)
	}
	err = ioutil.WriteFile(lss.StatePath, jsonBytes, 0600)
	if err != nil {
		panic(err)
	}
}
