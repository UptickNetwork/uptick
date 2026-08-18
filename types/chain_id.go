package types

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

// ChainIDFromGenesisFile reads the Cosmos chain-id from $home/config/genesis.json.
func ChainIDFromGenesisFile(homePath string) string {
	if homePath == "" {
		return ""
	}
	bz, err := os.ReadFile(filepath.Join(homePath, "config", "genesis.json"))
	if err != nil {
		return ""
	}
	var genesis struct {
		ChainID string `json:"chain_id"`
	}
	if err := json.Unmarshal(bz, &genesis); err != nil {
		return ""
	}
	return genesis.ChainID
}

// ResolveEVMChainID derives the EIP-155 chain id from the Cosmos chain-id
// ({name}_{eip155}-{revision}). app.toml / --evm.evm-chain-id is only used when
// no parseable Cosmos chain-id is available. The cosmos/evm default 262144 is
// treated as unset so NewRootCmd's temporary app does not lock start.
func ResolveEVMChainID(appOpts servertypes.AppOptions, cosmosChainID string) uint64 {
	if cosmosChainID == "" {
		cosmosChainID = cast.ToString(appOpts.Get(flags.FlagChainID))
	}
	if cosmosChainID != "" {
		if parsed, err := ParseEIP155ChainID(cosmosChainID); err == nil {
			return parsed
		}
	}
	configured := cast.ToUint64(appOpts.Get(srvflags.EVMChainID))
	if configured != 0 && configured != evmtypes.DefaultEVMChainID {
		return configured
	}
	return evmtypes.DefaultEVMChainID
}
