package ante

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"

	anteinterfaces "github.com/cosmos/evm/ante/interfaces"
	"github.com/cosmos/evm/crypto/ethsecp256k1"
	"github.com/cosmos/evm/ethereum/eip712"

	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	"github.com/cosmos/cosmos-sdk/x/auth/migrations/legacytx"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
)

// Eip712SigVerificationDecorator verifies EIP-712 signatures produced by
// wallets (e.g. Keplr) that sign Cosmos txs against the legacy ethermint
// ExtensionOptionsWeb3Tx extension option.
//
// CONTRACT: Pubkeys are set in context for all signers before this decorator runs.
type Eip712SigVerificationDecorator struct {
	ak  anteinterfaces.AccountKeeper
	cdc codec.BinaryCodec
}

// NewEip712SigVerificationDecorator creates a new Eip712SigVerificationDecorator.
// The codec is injected explicitly (the app's fully-populated ProtoCodec) so
// message unpacking for typed-data construction works for every registered
// interface — a package-level init() codec would only know the eip712 types
// and silently break for any message type it does not register.
func NewEip712SigVerificationDecorator(
	ak anteinterfaces.AccountKeeper,
	cdc codec.BinaryCodec,
) Eip712SigVerificationDecorator {
	return Eip712SigVerificationDecorator{
		ak:  ak,
		cdc: cdc,
	}
}

// AnteHandle validates and verifies EIP-712 signed Cosmos transactions. It is
// not executed on ReCheckTx.
func (svd Eip712SigVerificationDecorator) AnteHandle(
	ctx sdk.Context,
	tx sdk.Tx,
	simulate bool,
	next sdk.AnteHandler,
) (newCtx sdk.Context, err error) {
	if ctx.IsReCheckTx() {
		return next(ctx, tx, simulate)
	}

	sigTx, ok := tx.(authsigning.SigVerifiableTx)
	if !ok {
		return ctx, errorsmod.Wrapf(errortypes.ErrInvalidType, "tx %T doesn't implement authsigning.SigVerifiableTx", tx)
	}

	authSignTx, ok := tx.(authsigning.Tx)
	if !ok {
		return ctx, errorsmod.Wrapf(errortypes.ErrInvalidType, "tx %T doesn't implement the authsigning.Tx interface", tx)
	}

	sigs, err := sigTx.GetSignaturesV2()
	if err != nil {
		return ctx, err
	}

	signerAddrs, err := sigTx.GetSigners()
	if err != nil {
		return ctx, err
	}

	// EIP-712 allows just one signature.
	if len(sigs) != 1 {
		return ctx, errorsmod.Wrapf(
			errortypes.ErrTooManySignatures,
			"invalid number of signers (%d); EIP712 signatures allows just one signature",
			len(sigs),
		)
	}

	if len(sigs) != len(signerAddrs) {
		return ctx, errorsmod.Wrapf(errortypes.ErrorInvalidSigner, "invalid number of signers; expected: %d, got %d", len(signerAddrs), len(sigs))
	}

	i := 0
	sig := sigs[i]

	acc, err := authante.GetSignerAcc(ctx, svd.ak, signerAddrs[i])
	if err != nil {
		return ctx, err
	}

	pubKey := acc.GetPubKey()
	if !simulate && pubKey == nil {
		return ctx, errorsmod.Wrap(errortypes.ErrInvalidPubKey, "pubkey on account is not set")
	}

	// Self-contained signer binding. Do not rely solely on the SDK
	// IsSigverifyTx gate: if that gate is disabled, or the account has a balance
	// but no pubkey yet, a tx that swaps in an attacker pubkey must still be
	// rejected here.
	if sig.PubKey != nil {
		sigAddr := sdk.AccAddress(sig.PubKey.Address().Bytes())
		if !sigAddr.Equals(sdk.AccAddress(signerAddrs[i])) {
			return ctx, errorsmod.Wrapf(
				errortypes.ErrorInvalidSigner,
				"tx signer pubkey %s does not match declared signer %s",
				sigAddr, sdk.AccAddress(signerAddrs[i]),
			)
		}
	}

	if pubKey != nil && sig.PubKey != nil && !pubKey.Equals(sig.PubKey) {
		return ctx, errorsmod.Wrapf(
			errortypes.ErrInvalidPubKey,
			"on-chain pubkey %s does not match tx-declared pubkey %s",
			pubKey, sig.PubKey,
		)
	}

	if sig.Sequence != acc.GetSequence() {
		return ctx, errorsmod.Wrapf(
			errortypes.ErrWrongSequence,
			"account sequence mismatch, expected %d, got %d", acc.GetSequence(), sig.Sequence,
		)
	}

	genesis := ctx.BlockHeight() == 0
	chainID := ctx.ChainID()

	var accNum uint64
	if !genesis {
		accNum = acc.GetAccountNumber()
	}

	signerData := authsigning.SignerData{
		ChainID:       chainID,
		AccountNumber: accNum,
		Sequence:      acc.GetSequence(),
	}

	if simulate {
		return next(ctx, tx, simulate)
	}

	if err := verifyEip712Signature(svd.cdc, pubKey, signerData, sig.Data, authSignTx); err != nil {
		errMsg := fmt.Errorf("signature verification failed; please verify account number (%d) and chain-id (%s): %w", accNum, chainID, err)
		return ctx, errorsmod.Wrap(errortypes.ErrUnauthorized, errMsg.Error())
	}

	return next(ctx, tx, simulate)
}

// verifyEip712Signature verifies an EIP-712 signature embedded in a legacy
// ethermint ExtensionOptionsWeb3Tx extension option.
func verifyEip712Signature(
	cdc codectypes.AnyUnpacker,
	pubKey cryptotypes.PubKey,
	signerData authsigning.SignerData,
	sigData signing.SignatureData,
	tx authsigning.Tx,
) error {
	data, ok := sigData.(*signing.SingleSignatureData)
	if !ok {
		return errorsmod.Wrapf(errortypes.ErrNotSupported, "unexpected SignatureData %T", sigData)
	}

	if data.SignMode != signing.SignMode_SIGN_MODE_LEGACY_AMINO_JSON {
		return errorsmod.Wrapf(errortypes.ErrNotSupported, "unexpected SignatureData %T: wrong SignMode", sigData)
	}

	// The EIP-712 signature lives in the extension option; the Cosmos signature
	// body must be empty.
	if len(data.Signature) != 0 {
		return errorsmod.Wrap(errortypes.ErrTooManySignatures, "invalid signature value; EIP712 must have the cosmos transaction signature empty")
	}

	msgs := tx.GetMsgs()
	if len(msgs) == 0 {
		return errorsmod.Wrap(errortypes.ErrNoSignatures, "tx doesn't contain any msgs to verify signature")
	}

	txBytes := legacytx.StdSignBytes(
		signerData.ChainID,
		signerData.AccountNumber,
		signerData.Sequence,
		tx.GetTimeoutHeight(),
		legacytx.StdFee{
			Amount: tx.GetFee(),
			Gas:    tx.GetGas(),
		},
		msgs, tx.GetMemo(),
	)

	signerChainID, err := parseEIP155ChainID(signerData.ChainID)
	if err != nil {
		return errorsmod.Wrapf(err, "failed to parse chain-id: %s", signerData.ChainID)
	}

	txWithExtensions, ok := tx.(authante.HasExtensionOptionsTx)
	if !ok {
		return errorsmod.Wrap(errortypes.ErrUnknownExtensionOptions, "tx doesn't contain any extensions")
	}
	opts := txWithExtensions.GetExtensionOptions()
	if len(opts) != 1 {
		return errorsmod.Wrap(errortypes.ErrUnknownExtensionOptions, "tx doesn't contain expected amount of extension options")
	}

	extOpt, ok := opts[0].GetCachedValue().(*eip712.ExtensionOptionsWeb3Tx)
	if !ok {
		return errorsmod.Wrap(errortypes.ErrUnknownExtensionOptions, "unknown extension option")
	}

	if extOpt.TypedDataChainID != signerChainID {
		return errorsmod.Wrap(errortypes.ErrInvalidChainID, "invalid chain-id")
	}

	if len(extOpt.FeePayer) == 0 {
		return errorsmod.Wrap(errortypes.ErrUnknownExtensionOptions, "no feePayer on ExtensionOptionsWeb3Tx")
	}
	feePayer, err := sdk.AccAddressFromBech32(extOpt.FeePayer)
	if err != nil {
		return errorsmod.Wrap(err, "failed to parse feePayer from ExtensionOptionsWeb3Tx")
	}

	feeDelegation := &eip712.FeeDelegationOptions{
		FeePayer: feePayer,
	}

	typedData, err := eip712.LegacyWrapTxToTypedData(cdc, extOpt.TypedDataChainID, msgs[0], txBytes, feeDelegation)
	if err != nil {
		return errorsmod.Wrap(err, "failed to create EIP-712 typed data from tx")
	}

	sigHash, _, err := apitypes.TypedDataAndHash(typedData)
	if err != nil {
		return err
	}

	feePayerSig := make([]byte, len(extOpt.FeePayerSig))
	copy(feePayerSig, extOpt.FeePayerSig)
	if len(feePayerSig) != ethcrypto.SignatureLength {
		return errorsmod.Wrap(errortypes.ErrorInvalidSigner, "signature length doesn't match typical [R||S||V] signature 65 bytes")
	}

	// Remove the recovery offset if needed (e.g. MetaMask EIP-712 signature).
	// Copy first so CheckTx cannot mutate the tx's backing signature bytes.
	if feePayerSig[ethcrypto.RecoveryIDOffset] == 27 || feePayerSig[ethcrypto.RecoveryIDOffset] == 28 {
		feePayerSig[ethcrypto.RecoveryIDOffset] -= 27
	}

	// Enforce EIP-2 low-s normalization to reject malleable high-s signatures.
	r := new(big.Int).SetBytes(feePayerSig[:ethcrypto.RecoveryIDOffset-32])
	s := new(big.Int).SetBytes(feePayerSig[ethcrypto.RecoveryIDOffset-32 : ethcrypto.RecoveryIDOffset])
	if !ethcrypto.ValidateSignatureValues(feePayerSig[ethcrypto.RecoveryIDOffset], r, s, true) {
		return errorsmod.Wrap(errortypes.ErrorInvalidSigner, "invalid signature values (high-s or out-of-range)")
	}

	feePayerPubkey, err := secp256k1.RecoverPubkey(sigHash, feePayerSig)
	if err != nil {
		return errorsmod.Wrap(err, "failed to recover delegated fee payer from sig")
	}

	ecPubKey, err := ethcrypto.UnmarshalPubkey(feePayerPubkey)
	if err != nil {
		return errorsmod.Wrap(err, "failed to unmarshal recovered fee payer pubkey")
	}

	pk := &ethsecp256k1.PubKey{
		Key: ethcrypto.CompressPubkey(ecPubKey),
	}

	if !pubKey.Equals(pk) {
		return errorsmod.Wrapf(errortypes.ErrInvalidPubKey, "feePayer pubkey %s is different from transaction pubkey %s", pubKey, pk)
	}

	recoveredFeePayerAcc := sdk.AccAddress(pk.Address().Bytes())
	if !recoveredFeePayerAcc.Equals(feePayer) {
		return errorsmod.Wrapf(errortypes.ErrorInvalidSigner, "failed to verify delegated fee payer %s signature", recoveredFeePayerAcc)
	}

	// VerifySignature of ethsecp256k1 accepts 64 byte signature [R||S].
	if !secp256k1.VerifySignature(pubKey.Bytes(), sigHash, feePayerSig[:len(feePayerSig)-1]) {
		return errorsmod.Wrap(errortypes.ErrorInvalidSigner, "unable to verify signer signature of EIP712 typed data")
	}

	return nil
}

// parseEIP155ChainID extracts the numeric EIP-155 chain id from an
// ethermint-style chain id ({name}_{eip155}-{epoch}), e.g. "uptick_1170-1" ->
// 1170. Numeric chain ids are parsed as-is.
func parseEIP155ChainID(chainID string) (uint64, error) {
	if underscore := strings.LastIndex(chainID, "_"); underscore >= 0 {
		rest := chainID[underscore+1:]
		if dash := strings.Index(rest, "-"); dash >= 0 {
			return strconv.ParseUint(rest[:dash], 10, 64)
		}
	}
	return strconv.ParseUint(chainID, 10, 64)
}
