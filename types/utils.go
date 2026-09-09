package types

import (
	"encoding/hex"
	"fmt"
	"strings"

	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/evm/crypto/ethsecp256k1"

	sdkerrors "cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/crypto/types/multisig"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
)

const prefix = "uptick"

// IsSupportedKey returns true if the pubkey type is supported by the chain
// (i.e eth_secp256k1, amino multisig, ed25519).
// NOTE: Nested multisigs are not supported.
//
// Deprecated: not wired into signature verification; app/sigverify.go's
// SigVerificationGasConsumer is the source of truth (and rejects ed25519).
// Kept only for the existing unit test; do not call from production code.
//
// ed25519 is only used for consensus-validator key verification, not for
// transaction signing; this function is retained solely for compatibility
// with existing tests and must not be called from new code.
func IsSupportedKey(pubkey cryptotypes.PubKey) bool {
	switch pubkey := pubkey.(type) {
	case *ethsecp256k1.PubKey, *ed25519.PubKey:
		return true
	case multisig.PubKey:
		if len(pubkey.GetPubKeys()) == 0 {
			return false
		}

		for _, pk := range pubkey.GetPubKeys() {
			switch pk.(type) {
			case *ethsecp256k1.PubKey, *ed25519.PubKey:
				continue
			default:
				// Nested multisigs are unsupported
				return false
			}
		}

		return true
	default:
		return false
	}
}

// GetUptickAddressFromBech32 returns the sdk.Account address of given address,
// while also changing bech32 human readable prefix (HRP) to the value set on
// the global sdk.Config (eg: `uptick`).
// The function fails if the provided bech32 address is invalid.
func GetUptickAddressFromBech32(address string) (sdk.AccAddress, error) {
	bech32Prefix := strings.SplitN(address, "1", 2)[0]
	if bech32Prefix == address {
		return nil, sdkerrors.Wrapf(errortypes.ErrInvalidAddress, "invalid bech32 address: %s", address)
	}

	addressBz, err := sdk.GetFromBech32(address, bech32Prefix)
	if err != nil {
		return nil, sdkerrors.Wrapf(errortypes.ErrInvalidAddress, "invalid address %s, %s", address, err.Error())
	}

	// safety check: shouldn't happen
	if err := sdk.VerifyAddressFormat(addressBz); err != nil {
		return nil, err
	}

	return sdk.AccAddress(addressBz), nil
}

func ConvertAddressCosmos2Evm(cosmosAddress string) (string, error) {

	rawBytes, err := sdk.GetFromBech32(cosmosAddress, prefix)
	if err != nil {
		return "", err
	}
	if len(rawBytes) != common.AddressLength {
		return "", fmt.Errorf("invalid address length: %d", len(rawBytes))
	}
	evmAddress := "0x" + hex.EncodeToString(rawBytes)
	return evmAddress, nil
}

func ConvertAddressEvm2Cosmos(evmAddress string) (string, error) {
	if !strings.HasPrefix(evmAddress, "0x") {
		return "", fmt.Errorf("invalid evm address: %s", evmAddress)
	}

	rawBytes, err := hex.DecodeString(evmAddress[2:])
	if err != nil {
		return "", err
	}
	if len(rawBytes) != common.AddressLength {
		return "", fmt.Errorf("invalid address length: %d", len(rawBytes))
	}

	cosmosAddress, err := sdk.Bech32ifyAddressBytes(prefix, rawBytes)
	if err != nil {
		return "", err
	}

	return cosmosAddress, nil

}
