package simulation

import (
	"encoding/json"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"

	"github.com/UptickNetwork/uptick/x/collection/types"
)

const (
	kitties = "kitties"
	doggos  = "doggos"
)

// RandomizedGenState generates a random GenesisState for nft
func RandomizedGenState(simState *module.SimulationState) {
	collections := types.NewCollections(
		types.NewCollection(
			types.Denom{
				Id:      doggos,
				Name:    doggos,
				Schema:  "",
				Creator: "",
				Symbol:  "dog",
			},
			types.NFTs{},
		),
		types.NewCollection(
			types.Denom{
				Id:      kitties,
				Name:    kitties,
				Schema:  "",
				Creator: "",
				Symbol:  "kit",
			},
			types.NFTs{}),
	)
	for _, acc := range simState.Accounts {
		// 10% of accounts own an NFT
		if simState.Rand.Intn(100) < 10 {
			baseNFT := types.NewBaseNFT(
				genNFTID(simState.Rand, 3, 128), // id
				simtypes.RandStringOfLength(simState.Rand, 10),
				acc.Address,
				simtypes.RandStringOfLength(simState.Rand, 45), // tokenURI
				simtypes.RandStringOfLength(simState.Rand, 32), // tokenURIHash
				simtypes.RandStringOfLength(simState.Rand, 10),
			)

			// 50% doggos and 50% kitties
			if simState.Rand.Intn(100) < 50 {
				collections[0].Denom.Creator = baseNFT.Owner
				collections[0] = collections[0].AddNFT(baseNFT)
			} else {
				collections[1].Denom.Creator = baseNFT.Owner
				collections[1] = collections[1].AddNFT(baseNFT)
			}
		}
	}

	nftGenesis := types.NewGenesisState(collections)

	_, err := json.MarshalIndent(nftGenesis, "", " ")
	if err != nil {
		panic(err)
	}

	simState.GenState[types.ModuleName] = simState.Cdc.MustMarshalJSON(nftGenesis)
}
