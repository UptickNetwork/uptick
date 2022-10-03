package cli

import (
	flag "github.com/spf13/pflag"
)

const (
	FlagClassName        = "name"
	FlagClassSymbol      = "symbol"
	FlagClassDescription = "description"
	FlagURI              = "uri"
	FlagURIHash          = "uri-hash"
	FlagReceiver         = "receiver"
)

// common flagsets to add to various functions
var (
	fsIssueClass = flag.NewFlagSet("", flag.ContinueOnError)
	fsMintNFT    = flag.NewFlagSet("", flag.ContinueOnError)
)

func init() {
	fsIssueClass.String(FlagClassName, "", "Class Name")
	fsIssueClass.String(FlagClassSymbol, "", "Class Symbol")
	fsIssueClass.String(FlagClassDescription, "", "Class description")
	fsIssueClass.String(FlagURI, "", "Class uri")
	fsIssueClass.String(FlagURIHash, "", "Class uri hash")

	fsMintNFT.String(FlagURI, "", "nft uri")
	fsMintNFT.String(FlagURIHash, "", "nft uri hash")
	fsMintNFT.String(FlagReceiver, "", "nft receiver")
}
