package evmibc

import (
	"context"
	"fmt"
	"testing"

	storetypes "cosmossdk.io/store/types"
	cw721types "github.com/UptickNetwork/uptick/x/cw721/types"
	erc721types "github.com/UptickNetwork/uptick/x/erc721/types"
	evmibckeeper "github.com/UptickNetwork/uptick/x/evmibc/keeper"
	nfttransfertypes "github.com/bianjieai/nft-transfer/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v10/modules/core/05-port/types"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
	"github.com/stretchr/testify/require"
)

const (
	testHexReceiver   = "0x1111111111111111111111111111111111111111"
	testClassID       = "class-1"
	testTokenID       = "1"
	testSourcePort    = "nonfungibletokentransfer"
	testSourceChannel = "channel-0"
	testDestPort      = "nonfungibletokentransfer"
	testDestChannel   = "channel-1"
)

var (
	_ porttypes.IBCModule          = (*recordingIBCModule)(nil)
	_ evmibckeeper.ERC721Converter = (*fakeERC721)(nil)
	_ evmibckeeper.CW721Converter  = (*fakeCW721)(nil)
	_ evmibckeeper.ICS721Keeper    = (*fakeICS721)(nil)
)

type recordingIBCModule struct {
	storeKey storetypes.StoreKey
	recvOK   bool
	recvs    []channeltypes.Packet
	acks     int
	timeouts int
}

func (m *recordingIBCModule) OnChanOpenInit(sdk.Context, channeltypes.Order, []string, string, string, channeltypes.Counterparty, string) (string, error) {
	return "", nil
}
func (m *recordingIBCModule) OnChanOpenTry(sdk.Context, channeltypes.Order, []string, string, string, channeltypes.Counterparty, string) (string, error) {
	return "", nil
}
func (m *recordingIBCModule) OnChanOpenAck(sdk.Context, string, string, string, string) error {
	return nil
}
func (m *recordingIBCModule) OnChanOpenConfirm(sdk.Context, string, string) error {
	return nil
}
func (m *recordingIBCModule) OnChanCloseInit(sdk.Context, string, string) error {
	return nil
}
func (m *recordingIBCModule) OnChanCloseConfirm(sdk.Context, string, string) error {
	return nil
}

func (m *recordingIBCModule) OnRecvPacket(ctx sdk.Context, _ string, packet channeltypes.Packet, _ sdk.AccAddress) exported.Acknowledgement {
	m.recvs = append(m.recvs, packet)
	if m.storeKey != nil {
		ctx.KVStore(m.storeKey).Set([]byte("recv"), []byte("1"))
	}
	if !m.recvOK {
		return channeltypes.NewErrorAcknowledgement(fmt.Errorf("nft-transfer recv failed"))
	}
	return channeltypes.NewResultAcknowledgement([]byte{byte(1)})
}

func (m *recordingIBCModule) OnAcknowledgementPacket(sdk.Context, string, channeltypes.Packet, []byte, sdk.AccAddress) error {
	m.acks++
	return nil
}

func (m *recordingIBCModule) OnTimeoutPacket(sdk.Context, string, channeltypes.Packet, sdk.AccAddress) error {
	m.timeouts++
	return nil
}

type fakeERC721 struct {
	convertErr error
	refundErr  error
	converts   []erc721types.MsgConvertNFT
	refunds    []nfttransfertypes.NonFungibleTokenPacketData
}

func (f *fakeERC721) ConvertNFT(_ context.Context, msg *erc721types.MsgConvertNFT) (*erc721types.MsgConvertNFTResponse, error) {
	f.converts = append(f.converts, *msg)
	if f.convertErr != nil {
		return nil, f.convertErr
	}
	return &erc721types.MsgConvertNFTResponse{}, nil
}

func (f *fakeERC721) RefundPacketToken(_ sdk.Context, data nfttransfertypes.NonFungibleTokenPacketData) error {
	f.refunds = append(f.refunds, data)
	return f.refundErr
}

type fakeCW721 struct {
	convertErr error
	refundErr  error
	converts   []cw721types.MsgConvertNFT
	refunds    []nfttransfertypes.NonFungibleTokenPacketData
}

func (f *fakeCW721) ConvertNFT(_ context.Context, msg *cw721types.MsgConvertNFT) (*cw721types.MsgConvertNFTResponse, error) {
	f.converts = append(f.converts, *msg)
	if f.convertErr != nil {
		return nil, f.convertErr
	}
	return &cw721types.MsgConvertNFTResponse{}, nil
}

func (f *fakeCW721) RefundPacketToken(_ sdk.Context, data nfttransfertypes.NonFungibleTokenPacketData) error {
	f.refunds = append(f.refunds, data)
	return f.refundErr
}

type fakeICS721 struct {
	ackErr     error
	timeoutErr error
	acks       []nfttransfertypes.NonFungibleTokenPacketData
	timeouts   []nfttransfertypes.NonFungibleTokenPacketData
}

func (f *fakeICS721) OnAcknowledgementPacket(_ sdk.Context, _ channeltypes.Packet, data nfttransfertypes.NonFungibleTokenPacketData, _ channeltypes.Acknowledgement) error {
	f.acks = append(f.acks, data)
	return f.ackErr
}

func (f *fakeICS721) OnTimeoutPacket(_ sdk.Context, _ channeltypes.Packet, data nfttransfertypes.NonFungibleTokenPacketData) error {
	f.timeouts = append(f.timeouts, data)
	return f.timeoutErr
}

func newConvertHarness(t *testing.T, recvOK bool) (sdk.Context, storetypes.StoreKey, *recordingIBCModule, *fakeERC721, *fakeCW721, *fakeICS721, IBCMiddleware) {
	t.Helper()
	key := storetypes.NewKVStoreKey("convert-e2e")
	tkey := storetypes.NewTransientStoreKey("convert-e2e-transient")
	ctx := testutil.DefaultContext(key, tkey)

	app := &recordingIBCModule{storeKey: key, recvOK: recvOK}
	erc := &fakeERC721{}
	cw := &fakeCW721{}
	ics := &fakeICS721{}

	k := evmibckeeper.NewKeeper(ics)
	k.SetErc721Keeper(erc)
	k.SetCw721Keeper(cw)
	return ctx, key, app, erc, cw, ics, NewIBCMiddleware(k, app, nil)
}

func inboundPacket(receiver, memo string) channeltypes.Packet {
	data := nfttransfertypes.NonFungibleTokenPacketData{
		ClassId:  testClassID,
		TokenIds: []string{testTokenID},
		Sender:   "cosmos1sender",
		Receiver: receiver,
		Memo:     memo,
	}
	return channeltypes.Packet{
		Sequence:           1,
		SourcePort:         testSourcePort,
		SourceChannel:      testSourceChannel,
		DestinationPort:    testDestPort,
		DestinationChannel: testDestChannel,
		Data:               nfttransfertypes.ModuleCdc.MustMarshalJSON(&data),
	}
}

func outboundConvertPacket(sender, memo, classID string) channeltypes.Packet {
	data := nfttransfertypes.NonFungibleTokenPacketData{
		ClassId:  classID,
		TokenIds: []string{testTokenID},
		Sender:   sender,
		Receiver: "cosmos1dest",
		Memo:     memo,
	}
	return channeltypes.Packet{
		Sequence:      2,
		SourcePort:    testSourcePort,
		SourceChannel: testSourceChannel,
		Data:          nfttransfertypes.ModuleCdc.MustMarshalJSON(&data),
	}
}

func decodePacketData(t *testing.T, packet channeltypes.Packet) nfttransfertypes.NonFungibleTokenPacketData {
	t.Helper()
	var data nfttransfertypes.NonFungibleTokenPacketData
	require.NoError(t, nfttransfertypes.ModuleCdc.UnmarshalJSON(packet.GetData(), &data))
	return data
}

func TestKeeperOnRecvPacket_UnknownConvertType(t *testing.T) {
	key := storetypes.NewKVStoreKey("convert-e2e")
	tkey := storetypes.NewTransientStoreKey("convert-e2e-transient")
	ctx := testutil.DefaultContext(key, tkey)
	k := evmibckeeper.NewKeeper(&fakeICS721{})

	ack := k.OnRecvPacket(ctx, inboundPacket(testHexReceiver, ""), testHexReceiver, 99)
	require.False(t, ack.Success(), "unknown convertType must fail closed without committing")
}

func TestConvertRecv_ERC721Success(t *testing.T) {
	ctx, key, app, erc, _, _, im := newConvertHarness(t, true)

	ack := im.OnRecvPacket(ctx, "", inboundPacket(testHexReceiver, `{"convert_to":"erc721"}`), nil)
	require.True(t, ack.Success())
	require.True(t, ctx.KVStore(key).Has([]byte("recv")), "nft-transfer mint must commit on convert success")
	require.Len(t, app.recvs, 1)
	require.Equal(t, erc721types.AccModuleAddress.String(), decodePacketData(t, app.recvs[0]).Receiver)
	require.Len(t, erc.converts, 1)
	require.Equal(t, testHexReceiver, erc.converts[0].EvmReceiver)
	require.Equal(t, erc721types.AccModuleAddress.String(), erc.converts[0].CosmosSender)
	require.Equal(t, []string{testTokenID}, erc.converts[0].CosmosTokenIds)
}

func TestConvertRecv_CW721Success(t *testing.T) {
	receiver := sdk.AccAddress([]byte("cw721-receiver-addr01")).String()
	ctx, key, app, _, cw, _, im := newConvertHarness(t, true)

	ack := im.OnRecvPacket(ctx, "", inboundPacket(receiver, `{"convert_to":"cw721"}`), nil)
	require.True(t, ack.Success())
	require.True(t, ctx.KVStore(key).Has([]byte("recv")))
	require.Len(t, app.recvs, 1)
	require.Equal(t, cw721types.AccModuleAddress.String(), decodePacketData(t, app.recvs[0]).Receiver)
	require.Len(t, cw.converts, 1)
	require.Equal(t, receiver, cw.converts[0].Receiver)
	require.Equal(t, cw721types.AccModuleAddress.String(), cw.converts[0].Sender)
}

func TestConvertRecv_FailClosedOnConvertError(t *testing.T) {
	ctx, key, app, erc, _, _, im := newConvertHarness(t, true)
	erc.convertErr = fmt.Errorf("pair missing")

	ack := im.OnRecvPacket(ctx, "", inboundPacket(testHexReceiver, `{"convert_to":"erc721"}`), nil)
	require.False(t, ack.Success())
	require.False(t, ctx.KVStore(key).Has([]byte("recv")), "convert failure must roll back nft-transfer mint")
	require.Len(t, erc.converts, 1)
	require.Len(t, app.recvs, 1)
}

func TestConvertRecv_FailClosedOnUnderlyingRecvError(t *testing.T) {
	ctx, key, app, erc, _, _, im := newConvertHarness(t, false)

	ack := im.OnRecvPacket(ctx, "", inboundPacket(testHexReceiver, `{"convert_to":"erc721"}`), nil)
	require.False(t, ack.Success())
	require.False(t, ctx.KVStore(key).Has([]byte("recv")))
	require.Empty(t, erc.converts, "convert must not run when nft-transfer recv fails")
	require.Len(t, app.recvs, 1)
}

func TestConvertAck_ErrorRefundsAndSkipsNFTTransferModule(t *testing.T) {
	ctx, _, app, erc, _, ics, im := newConvertHarness(t, true)
	packet := outboundConvertPacket(erc721types.AccModuleAddress.String(), "user"+erc721types.TransferERC721Memo, testClassID)
	ackBz := channeltypes.NewErrorAcknowledgement(fmt.Errorf("remote convert failed")).Acknowledgement()

	require.NoError(t, im.OnAcknowledgementPacket(ctx, "", packet, ackBz, nil))
	require.Equal(t, 0, app.acks, "nft-transfer module must not also refund convert packets")
	require.Len(t, erc.refunds, 1)
	require.Equal(t, testClassID, erc.refunds[0].ClassId)
	require.Len(t, ics.acks, 1)
	require.Equal(t, erc721types.AccModuleAddress.String(), ics.acks[0].Sender)
}

func TestConvertAck_SuccessFallsThroughToNFTTransfer(t *testing.T) {
	ctx, _, app, erc, _, ics, im := newConvertHarness(t, true)
	packet := outboundConvertPacket(erc721types.AccModuleAddress.String(), "user"+erc721types.TransferERC721Memo, testClassID)
	ackBz := channeltypes.NewResultAcknowledgement([]byte{1}).Acknowledgement()

	require.NoError(t, im.OnAcknowledgementPacket(ctx, "", packet, ackBz, nil))
	require.Equal(t, 1, app.acks)
	require.Empty(t, erc.refunds)
	require.Empty(t, ics.acks)
}

func TestConvertTimeout_RefundsAndSkipsNFTTransferModule(t *testing.T) {
	ctx, _, app, _, cw, ics, im := newConvertHarness(t, true)
	packet := outboundConvertPacket(cw721types.AccModuleAddress.String(), "user"+cw721types.TransferCW721Memo, testClassID)

	require.NoError(t, im.OnTimeoutPacket(ctx, "", packet, nil))
	require.Equal(t, 0, app.timeouts, "nft-transfer module must not also refund convert packets")
	require.Len(t, cw.refunds, 1)
	require.Len(t, ics.timeouts, 1)
	require.Equal(t, cw721types.AccModuleAddress.String(), ics.timeouts[0].Sender)
}

func TestConvertTimeout_NonConvertUsesNFTTransfer(t *testing.T) {
	ctx, _, app, erc, cw, ics, im := newConvertHarness(t, true)
	packet := outboundConvertPacket("cosmos1user", "plain memo", testClassID)

	require.NoError(t, im.OnTimeoutPacket(ctx, "", packet, nil))
	require.Equal(t, 1, app.timeouts)
	require.Empty(t, erc.refunds)
	require.Empty(t, cw.refunds)
	require.Empty(t, ics.timeouts)
}
