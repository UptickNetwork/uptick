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

// shapePageRequest shapes the PageRequest params to avoid querying all items.
// PageRequest.offset is forbidden and PageRequest.count_total must be zero:
// the underlying store iterators are key-based, so an offset would silently
// scan-and-discard rows (potentially the entire collection). Report these as
// InvalidArgument so the caller cannot accidentally trigger an unbounded scan.
// PageRequest.limit mustn't exceed paginationMaxLimit and is set to
// paginationDefaultLimit when unset.
func shapePageRequest(req *query.PageRequest) (*query.PageRequest, error) {
	res := newDefaultPageRequest()

	if req == nil {
		return res, nil
	}

	// Offset and CountTotal are rejected: an offset forces a linear scan from
	// the start of the store to the offset position, and CountTotal forces a
	// full scan. Both are unbounded on a large collection and were silently
	// dropped before, which masked client bugs.
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

// newDefaultPageRequest returns a default PageRequest.
func newDefaultPageRequest() *query.PageRequest {
	return &query.PageRequest{
		Key:        nil,
		Offset:     0,
		Limit:      paginationDefaultLimit,
		CountTotal: false,
		Reverse:    false,
	}
}
