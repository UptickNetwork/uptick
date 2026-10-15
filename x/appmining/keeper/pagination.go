package keeper

import (
	sdkerrors "cosmossdk.io/errors"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/query"
)

var (
	paginationDefaultLimit uint64 = 100
	paginationMaxLimit     uint64 = 100
)

// shapePageRequest matches x/collection and x/cw721: key-based only, no offset/count_total.
func shapePageRequest(req *query.PageRequest) (*query.PageRequest, error) {
	res := &query.PageRequest{
		Limit: paginationDefaultLimit,
	}
	if req == nil {
		return res, nil
	}
	if req.Offset > 0 {
		return nil, sdkerrors.Wrap(errortypes.ErrInvalidRequest, "page request offset is not supported, use key-based pagination")
	}
	if req.CountTotal {
		return nil, sdkerrors.Wrap(errortypes.ErrInvalidRequest, "page request count_total is not supported")
	}
	res.Key = req.Key
	res.Reverse = req.Reverse
	if req.Limit > 0 && req.Limit <= paginationMaxLimit {
		res.Limit = req.Limit
	}
	return res, nil
}
