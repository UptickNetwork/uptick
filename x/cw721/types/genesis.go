package types

import "fmt"

// NewGenesisState creates a new genesis state.
func NewGenesisState(params Params, pairs []TokenPair) GenesisState {
	return GenesisState{
		Params:     params,
		TokenPairs: pairs,
	}
}

// DefaultGenesisState sets default evm genesis state with empty accounts and
// default params and chain config values.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params: DefaultParams(),
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	seencw721 := make(map[string]bool)
	seenClass := make(map[string]bool)

	for _, b := range gs.TokenPairs {
		if seencw721[b.Cw721Address] {
			return fmt.Errorf("token CW721 contract duplicated on genesis '%s'", b.Cw721Address)
		}
		if seenClass[b.ClassId] {
			return fmt.Errorf("nft class duplicated on genesis: '%s'", b.ClassId)
		}
		if err := b.Validate(); err != nil {
			return err
		}
		seencw721[b.Cw721Address] = true
		seenClass[b.ClassId] = true
	}

	return gs.Params.Validate()
}
