package main

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/client"

	"github.com/UptickNetwork/uptick/app/params"
)

// Resolving a key name needs a keyring. When --keyring-backend is left empty
// AND the command context carries no keyring there is nothing to look the name
// up in, and the old code assigned that nil keyring and immediately called Key
// on it: a nil-interface method call, i.e. a panic, instead of a message that
// names the missing flag.
//
// The address argument is deliberately not valid bech32, which is what routes
// execution into the keyring branch.
func TestAddGenesisAccountCmdDoesNotPanicWithoutKeyring(t *testing.T) {
	home := t.TempDir()
	cmd := AddGenesisAccountCmd(home)

	enc := params.MakeEncodingConfig()
	clientCtx := client.Context{}.
		WithCodec(enc.Codec).
		WithHomeDir(home).
		WithInput(strings.NewReader("")).
		WithKeyring(nil)
	// cobra's Context() returns c.ctx verbatim, and client.SetCmdClientContext
	// dereferences it, so a bare command needs one planted before that call.
	cmd.SetContext(context.Background())
	require.NoError(t, client.SetCmdClientContext(cmd, clientCtx))

	// --keyring-backend defaults to "os", so reaching the nil-keyring path takes
	// an explicit empty value.
	cmd.SetArgs([]string{"not-a-key-name", "1000auptick", "--keyring-backend="})

	require.NotPanics(t, func() {
		err := cmd.Execute()
		require.Error(t, err, "an unresolvable name must fail")
		require.Contains(t, err.Error(), "cannot resolve",
			"the error must name the actual cause instead of a nil dereference")
	})
}
