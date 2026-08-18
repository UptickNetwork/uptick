package types

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cast"

	"github.com/cosmos/cosmos-sdk/client/flags"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	srvflags "github.com/cosmos/evm/server/flags"
	evmtypes "github.com/cosmos/evm/x/vm/types"
)

const (
	// MainnetEVMChainID is the EIP-155 chain ID for uptick_117-1.
	MainnetEVMChainID uint64 = 117
	// TestnetEVMChainID is the EIP-155 chain ID for uptick_1170-3.
	TestnetEVMChainID uint64 = 1170
)

// ParseEIP155ChainID extracts the EIP-155 id from a Cosmos chain-id of the
// form {name}_{eip155}-{revision}, e.g. uptick_117-1 → 117.
func ParseEIP155ChainID(chainID string) (uint64, error) {
	underscore := strings.LastIndex(chainID, "_")
	dash := strings.LastIndex(chainID, "-")
	if underscore < 0 || dash <= underscore+1 {
		return 0, fmt.Errorf("invalid eip-155 chain-id %q", chainID)
	}
	id, err := strconv.ParseUint(chainID[underscore+1:dash], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid eip-155 chain-id %q: %w", chainID, err)
	}
	return id, nil
}

// ResolveEVMChainID picks the EIP-155 chain id from app.toml when it is a
// real configured value, otherwise parses it from the Cosmos chain-id.
func ResolveEVMChainID(appOpts servertypes.AppOptions, cosmosChainID string) uint64 {
	configured := cast.ToUint64(appOpts.Get(srvflags.EVMChainID))
	if configured != 0 && configured != evmtypes.DefaultEVMChainID {
		return configured
	}
	if cosmosChainID == "" {
		cosmosChainID = cast.ToString(appOpts.Get(flags.FlagChainID))
	}
	if cosmosChainID != "" {
		if parsed, err := ParseEIP155ChainID(cosmosChainID); err == nil {
			return parsed
		}
	}
	if configured == evmtypes.DefaultEVMChainID {
		return MainnetEVMChainID
	}
	return MainnetEVMChainID
}
