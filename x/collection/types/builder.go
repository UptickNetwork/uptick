package types

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"cosmossdk.io/x/nft"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	proto "github.com/cosmos/gogoproto/proto"
)

const (
	Namespace          = "irismod:"
	KeyMediaFieldValue = "value"
)

var (
	ClassKeyName             = fmt.Sprintf("%s%s", Namespace, "name")
	ClassKeySymbol           = fmt.Sprintf("%s%s", Namespace, "symbol")
	ClassKeyDescription      = fmt.Sprintf("%s%s", Namespace, "description")
	ClassKeyURIhash          = fmt.Sprintf("%s%s", Namespace, "uri_hash")
	ClassKeyMintRestricted   = fmt.Sprintf("%s%s", Namespace, "mint_restricted")
	ClassKeyUpdateRestricted = fmt.Sprintf("%s%s", Namespace, "update_restricted")
	ClassKeyCreator          = fmt.Sprintf("%s%s", Namespace, "creator")
	ClassKeySchema           = fmt.Sprintf("%s%s", Namespace, "schema")
	TokenKeyName             = fmt.Sprintf("%s%s", Namespace, "name")
	TokenKeyURIhash          = fmt.Sprintf("%s%s", Namespace, "uri_hash")

	Base64 = base64.StdEncoding
)

type (
	ClassBuilder struct {
		cdc              codec.Codec
		getModuleAddress func(string) sdk.AccAddress
	}
	TokenBuilder struct{ cdc codec.Codec }
	MediaField   struct {
		Value interface{} `json:"value"`
		Mime  string      `json:"mime,omitempty"`
	}
)

func NewClassBuilder(cdc codec.Codec,
	getModuleAddress func(string) sdk.AccAddress,
) ClassBuilder {
	return ClassBuilder{
		cdc:              cdc,
		getModuleAddress: getModuleAddress,
	}
}

// BuildMetadata encode class into the metadata format defined by ics721
func (cb ClassBuilder) BuildMetadata(class nft.Class) (string, error) {
	// A class created before the metadata wrapper existed — a legacy record
	// surviving a migration, or one written straight through the base nft
	// keeper — carries nil Data. UnpackAny(nil, ...) reports success and leaves
	// the message nil, so the type assertion used to reject exactly those
	// classes: InterNftKeeper.GetClass then answered "not found" and every
	// ICS-721 transfer of the class aborted. Degrade to zero-value metadata
	// instead, which is what GetDenomInfo (x/collection/keeper/denom.go) already
	// does for the same input — so the query API and the ICS-721 export now
	// describe an unrecorded class identically.
	//
	// A class whose Data is present but not a DenomMetadata is still a hard
	// error: that is a corrupt record, not a missing one.
	metadata := &DenomMetadata{}
	if class.Data != nil {
		var message proto.Message
		if err := cb.cdc.UnpackAny(class.Data, &message); err != nil {
			return "", err
		}
		unpacked, ok := message.(*DenomMetadata)
		if !ok {
			return "", errors.New("unsupported class metadata: expected DenomMetadata")
		}
		metadata = unpacked
	}

	kvals := make(map[string]interface{})
	if len(metadata.Data) > 0 {
		err := json.Unmarshal([]byte(metadata.Data), &kvals)
		if err != nil && IsIBCDenom(class.Id) {
			// when classData is not a legal json, there is no need to parse the data
			return Base64.EncodeToString([]byte(metadata.Data)), nil
		}
		//note: if metadata.Data is null, it may cause map to be redefined as nil
		if kvals == nil {
			kvals = make(map[string]interface{})
		}
	}
	// The creator is the one field the encoder cannot leave blank: an empty
	// string fails AccAddressFromBech32 and would take the whole export down
	// with it (including the degraded case above, one line later). Fall back to
	// the module address — the same default Build applies to an inbound packet
	// that carries no creator — so a class with no recorded creator still
	// travels, and encode/decode stay symmetric for that case.
	creatorBech32 := metadata.Creator
	if creatorBech32 == "" {
		creatorBech32 = cb.getModuleAddress(ModuleName).String()
	}
	creator, err := sdk.AccAddressFromBech32(creatorBech32)
	if err != nil {
		return "", err
	}

	hexCreator := hex.EncodeToString(creator)
	kvals[ClassKeyName] = MediaField{Value: class.Name}
	kvals[ClassKeySymbol] = MediaField{Value: class.Symbol}
	kvals[ClassKeyDescription] = MediaField{Value: class.Description}
	kvals[ClassKeyURIhash] = MediaField{Value: class.UriHash}
	kvals[ClassKeyMintRestricted] = MediaField{Value: metadata.MintRestricted}
	kvals[ClassKeyUpdateRestricted] = MediaField{Value: metadata.UpdateRestricted}
	kvals[ClassKeyCreator] = MediaField{Value: hexCreator}
	kvals[ClassKeySchema] = MediaField{Value: metadata.Schema}
	data, err := json.Marshal(kvals)
	if err != nil {
		return "", err
	}
	return Base64.EncodeToString(data), nil
}

// Build create a class from ics721 packetData
func (cb ClassBuilder) Build(classID, classURI, classData string) (nft.Class, error) {

	classDataBz, err := Base64.DecodeString(classData)

	if err != nil {
		return nft.Class{}, err
	}

	var (
		name             = ""
		symbol           = ""
		description      = ""
		uriHash          = ""
		mintRestricted   = true
		updateRestricted = true
		schema           = ""
		creator          = cb.getModuleAddress(ModuleName).String()
	)

	dataMap := make(map[string]interface{})
	if err := json.Unmarshal(classDataBz, &dataMap); err != nil {
		anyVal, err := codectypes.NewAnyWithValue(&DenomMetadata{
			Creator:          creator,
			Schema:           schema,
			MintRestricted:   mintRestricted,
			UpdateRestricted: updateRestricted,
			Data:             string(classDataBz),
		})
		if err != nil {
			return nft.Class{}, err
		}
		return nft.Class{
			Id:          classID,
			Uri:         classURI,
			Name:        name,
			Symbol:      symbol,
			Description: description,
			UriHash:     uriHash,
			Data:        anyVal,
		}, nil
	}

	if v, ok := dataMap[ClassKeyName]; ok {

		if vMap, ok := v.(map[string]interface{}); ok {

			if vStr, ok := vMap[KeyMediaFieldValue].(string); ok {
				name = vStr
				delete(dataMap, ClassKeyName)
			}
		}
	}

	if v, ok := dataMap[ClassKeySymbol]; ok {

		if vMap, ok := v.(map[string]interface{}); ok {

			if vStr, ok := vMap[KeyMediaFieldValue].(string); ok {

				symbol = vStr
				delete(dataMap, ClassKeySymbol)
			}
		}
	}

	if v, ok := dataMap[ClassKeyDescription]; ok {
		if vMap, ok := v.(map[string]interface{}); ok {
			if vStr, ok := vMap[KeyMediaFieldValue].(string); ok {
				description = vStr
				delete(dataMap, ClassKeyDescription)
			}
		}
	}

	if v, ok := dataMap[ClassKeyURIhash]; ok {
		if vMap, ok := v.(map[string]interface{}); ok {
			if vStr, ok := vMap[KeyMediaFieldValue].(string); ok {
				uriHash = vStr
				delete(dataMap, ClassKeyURIhash)
			}
		}
	}

	if v, ok := dataMap[ClassKeyMintRestricted]; ok {
		if vMap, ok := v.(map[string]interface{}); ok {
			if vBool, ok := vMap[KeyMediaFieldValue].(bool); ok {
				mintRestricted = vBool
				delete(dataMap, ClassKeyMintRestricted)
			}
		}
	}

	if v, ok := dataMap[ClassKeyUpdateRestricted]; ok {
		if vMap, ok := v.(map[string]interface{}); ok {
			if vBool, ok := vMap[KeyMediaFieldValue].(bool); ok {
				updateRestricted = vBool
				delete(dataMap, ClassKeyUpdateRestricted)
			}
		}
	}

	if v, ok := dataMap[ClassKeyCreator]; ok {
		if vMap, ok := v.(map[string]interface{}); ok {
			if vStr, ok := vMap[KeyMediaFieldValue].(string); ok {
				creatorAcc, err := sdk.AccAddressFromHexUnsafe(vStr)
				if err != nil {
					return nft.Class{}, err
				}
				creator = creatorAcc.String()
				delete(dataMap, ClassKeyCreator)
			}
		}
	}

	if v, ok := dataMap[ClassKeySchema]; ok {
		if vMap, ok := v.(map[string]interface{}); ok {
			if vStr, ok := vMap[KeyMediaFieldValue].(string); ok {
				schema = vStr
				delete(dataMap, ClassKeySchema)
			}
		}
	}

	var data = ""
	if len(dataMap) > 0 {
		dataBz, err := json.Marshal(dataMap)
		if err != nil {
			return nft.Class{}, err
		}
		data = string(dataBz)
	}

	anyVal, err := codectypes.NewAnyWithValue(&DenomMetadata{
		Creator:          creator,
		Schema:           schema,
		MintRestricted:   mintRestricted,
		UpdateRestricted: updateRestricted,
		Data:             data,
	})
	if err != nil {
		return nft.Class{}, err
	}

	return nft.Class{
		Id:          classID,
		Uri:         classURI,
		Name:        name,
		Symbol:      symbol,
		Description: description,
		UriHash:     uriHash,
		Data:        anyVal,
	}, nil
}

func NewTokenBuilder(cdc codec.Codec) TokenBuilder {
	return TokenBuilder{
		cdc: cdc,
	}
}

// BuildMetadata encode nft into the metadata format defined by ics721
func (tb TokenBuilder) BuildMetadata(token nft.NFT) (string, error) {
	// Mirror the tolerance ClassBuilder.BuildMetadata applies to a nil Data
	// blob: a token created before the metadata wrapper existed — a legacy
	// record surviving a migration, or one written straight through the base
	// nft keeper — carries nil Data. UnpackAny(nil, ...) reports success and
	// leaves the message nil, so the type assertion used to reject exactly
	// those tokens: InterNftKeeper.GetNFT then answered "not found", so every
	// ICS-721 outbound send and ack -- and every genesis export -- of a token
	// that plainly exists aborted. (Only the outbound path runs through here;
	// an inbound packet is minted via the fork's Mint, not BuildMetadata.)
	// Degrade to zero-value metadata instead, which is what the class
	// side already does for the same input, so the two read paths describe an
	// unrecorded record identically.
	//
	// A token whose Data is present but not an NFTMetadata is still a hard
	// error: that is a corrupt record, not a missing one — the same judgement
	// the class side makes.
	nftMetadata := &NFTMetadata{}
	if token.Data != nil {
		var message proto.Message
		if err := tb.cdc.UnpackAny(token.Data, &message); err != nil {
			return "", err
		}
		unpacked, ok := message.(*NFTMetadata)
		if !ok {
			return "", errors.New("unsupported nft metadata: expected NFTMetadata")
		}
		nftMetadata = unpacked
	}

	kvals := make(map[string]interface{})
	if len(nftMetadata.Data) > 0 {
		err := json.Unmarshal([]byte(nftMetadata.Data), &kvals)
		if err != nil && IsIBCDenom(token.ClassId) {
			// when nftMetadata is not a legal json, there is no need to parse the data
			return Base64.EncodeToString([]byte(nftMetadata.Data)), nil
		}
		//note: if nftMetadata.Data is null, it may cause map to be redefined as nil
		if kvals == nil {
			kvals = make(map[string]interface{})
		}
	}
	kvals[TokenKeyName] = MediaField{Value: nftMetadata.Name}
	kvals[TokenKeyURIhash] = MediaField{Value: token.UriHash}
	data, err := json.Marshal(kvals)
	if err != nil {
		return "", err
	}
	return Base64.EncodeToString(data), nil
}

// Build create a nft from ics721 packet data
func (tb TokenBuilder) Build(classId, tokenId, tokenURI, tokenData string) (nft.NFT, error) {
	tokenDataBz, err := Base64.DecodeString(tokenData)
	if err != nil {
		return nft.NFT{}, err
	}

	dataMap := make(map[string]interface{})
	if err := json.Unmarshal(tokenDataBz, &dataMap); err != nil {
		metadata, err := codectypes.NewAnyWithValue(&NFTMetadata{
			Data: string(tokenDataBz),
		})
		if err != nil {
			return nft.NFT{}, err
		}

		return nft.NFT{
			ClassId: classId,
			Id:      tokenId,
			Uri:     tokenURI,
			Data:    metadata,
		}, nil
	}

	var (
		name    string
		uriHash string
	)
	if v, ok := dataMap[TokenKeyName]; ok {
		if vMap, ok := v.(map[string]interface{}); ok {
			if vStr, ok := vMap[KeyMediaFieldValue].(string); ok {
				name = vStr
				delete(dataMap, TokenKeyName)
			}
		}
	}

	if v, ok := dataMap[TokenKeyURIhash]; ok {
		if vMap, ok := v.(map[string]interface{}); ok {
			if vStr, ok := vMap[KeyMediaFieldValue].(string); ok {
				uriHash = vStr
				delete(dataMap, TokenKeyURIhash)
			}
		}
	}

	var data = ""
	if len(dataMap) > 0 {
		dataBz, err := json.Marshal(dataMap)
		if err != nil {
			return nft.NFT{}, err
		}
		data = string(dataBz)
	}

	metadata, err := codectypes.NewAnyWithValue(&NFTMetadata{
		Name: name,
		Data: data,
	})
	if err != nil {
		return nft.NFT{}, err
	}

	return nft.NFT{
		ClassId: classId,
		Id:      tokenId,
		Uri:     tokenURI,
		UriHash: uriHash,
		Data:    metadata,
	}, nil
}
