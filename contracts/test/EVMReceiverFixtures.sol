// SPDX-License-Identifier: MIT
// Test-only ERC721 receiver fixtures.
//
// These are NOT part of the shipped protocol: they exist so the Go integration
// suite (app/erc721_contract_evm_test.go) can deploy real contracts into the
// real in-memory EVM and observe how ERC721Uptick behaves when the receiver
// hook is hostile, absent, or state-inspecting.
//
// They are compiled by contracts/test/gen_receivers.py, which emits the
// creation bytecode as Go constants. Nothing here is embedded by the node.
pragma solidity ^0.8.20;

interface IERC721Receiver {
    function onERC721Received(address operator, address from, uint256 tokenId, bytes calldata data)
        external
        returns (bytes4);
}

interface IERC721UptickView {
    function ownerOf(uint256 tokenId) external view returns (address);
    function balanceOf(address owner) external view returns (uint256);
    function totalSupply() external view returns (uint256);
    function tokenURI(uint256 tokenId) external view returns (string memory);
    function getNFTEnhanceInfo(uint256 id)
        external
        view
        returns (string memory name, string memory uri, string memory data, string memory uriHash);
    function burn(uint256 tokenId) external;
}

/// Deploys fine but implements no receiver interface at all: the classic
/// "contract address that cannot take an ERC721" case the audit calls out. A
/// plain mint to it would lock the token forever; a safe mint must revert.
contract PlainContract {
    uint256 public value;

    function setValue(uint256 v) external {
        value = v;
    }
}

/// Accepts every token and reports the canonical selector, recording what the
/// hook observed so the test can prove the callback really executed.
contract AcceptingReceiver is IERC721Receiver {
    uint256 public received;
    address public lastOperator;
    address public lastFrom;
    uint256 public lastTokenId;
    uint256 public lastDataLength;

    function onERC721Received(address operator, address from, uint256 tokenId, bytes calldata data)
        external
        returns (bytes4)
    {
        received += 1;
        lastOperator = operator;
        lastFrom = from;
        lastTokenId = tokenId;
        lastDataLength = data.length;
        return IERC721Receiver.onERC721Received.selector;
    }
}

/// Implements the interface but always reverts: a hook that rejects must roll
/// the whole mint back, not leave a half-created token behind.
contract RevertingReceiver is IERC721Receiver {
    function onERC721Received(address, address, uint256, bytes calldata) external pure returns (bytes4) {
        revert("RevertingReceiver: rejected");
    }
}

/// Implements the interface but burns the entire forwarded gas budget, so the
/// outer mint has to fail with out-of-gas rather than partially succeed.
contract GasBurningReceiver is IERC721Receiver {
    function onERC721Received(address, address, uint256, bytes calldata) external pure returns (bytes4) {
        // Written in assembly so the compiler cannot prove the loop is a no-op
        // and remove it.
        assembly {
            for { } 1 { } {
                mstore(0, add(mload(0), 1))
            }
        }
        return IERC721Receiver.onERC721Received.selector;
    }
}

/// Reads the ERC721 contract from inside the hook. The safe-mint ordering
/// (ownership + metadata written before the callback) means the token must
/// already be owned by this contract and already carry its metadata here.
contract ObservingReceiver is IERC721Receiver {
    address public immutable target;
    address public observedOwner;
    uint256 public observedBalance;
    uint256 public observedTotalSupply;
    string public observedUri;
    string public observedEnhanceName;
    string public observedEnhanceData;
    string public observedEnhanceUriHash;

    constructor(address target_) {
        target = target_;
    }

    function onERC721Received(address, address, uint256 tokenId, bytes calldata) external returns (bytes4) {
        IERC721UptickView t = IERC721UptickView(target);
        observedOwner = t.ownerOf(tokenId);
        observedBalance = t.balanceOf(address(this));
        observedTotalSupply = t.totalSupply();
        observedUri = t.tokenURI(tokenId);
        (string memory name, , string memory data, string memory uriHash) = t.getNFTEnhanceInfo(tokenId);
        observedEnhanceName = name;
        observedEnhanceData = data;
        observedEnhanceUriHash = uriHash;
        return IERC721Receiver.onERC721Received.selector;
    }
}

/// Re-enters a state-changing entry point during the hook: it burns the token
/// it is being handed. The mint must finish without corrupting state, and the
/// burn must leave no metadata residue behind (audit G-08 through the
/// reentrant path).
contract BurnOnReceiveReceiver is IERC721Receiver {
    address public immutable target;
    bool public reentered;

    constructor(address target_) {
        target = target_;
    }

    function onERC721Received(address, address, uint256 tokenId, bytes calldata) external returns (bytes4) {
        if (!reentered) {
            reentered = true;
            IERC721UptickView(target).burn(tokenId);
        }
        return IERC721Receiver.onERC721Received.selector;
    }
}
