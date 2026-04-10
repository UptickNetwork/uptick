package types

import "testing"

func TestTokenPairGettersNilReceiver(t *testing.T) {
	var pair *TokenPair
	if pair.GetErc721Address() != "" {
		t.Fatalf("expected empty erc721 address for nil receiver")
	}
	if pair.GetClassId() != "" {
		t.Fatalf("expected empty class id for nil receiver")
	}
}

func TestQueryEvmAddressRequestGettersNilReceiver(t *testing.T) {
	var req *QueryEvmAddressRequest
	if req.GetPort() != "" {
		t.Fatalf("expected empty port for nil receiver")
	}
	if req.GetChannel() != "" {
		t.Fatalf("expected empty channel for nil receiver")
	}
	if req.GetClassId() != "" {
		t.Fatalf("expected empty class id for nil receiver")
	}
}
