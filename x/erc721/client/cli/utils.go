package cli

import (
	"io/ioutil"
	"path/filepath"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/x/nft"
)

// ParseRegisterNFTProposal reads and parses a ParseRegisterNFTProposal from a file.
func ParseClass(cdc codec.JSONCodec, classFile string) (nft.Class, error) {
	class := nft.Class{}

	contents, err := ioutil.ReadFile(filepath.Clean(classFile))
	if err != nil {
		return class, err
	}

	if err = cdc.UnmarshalJSON(contents, &class); err != nil {
		return class, err
	}

	return class, nil
}
