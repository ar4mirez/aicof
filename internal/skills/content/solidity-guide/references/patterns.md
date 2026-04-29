# Solidity Patterns Reference

## Contents

- [ERC20 Token Implementation](#erc20-token-implementation)
- [Proxy Upgrade Pattern](#proxy-upgrade-pattern)
- [Gas Optimization Examples](#gas-optimization-examples)

## ERC20 Token Implementation

Production ERC20 with OpenZeppelin: access control, minting cap, pausability, and EIP-2612 permit.

```solidity
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import {ERC20, ERC20Burnable} from "@openzeppelin/contracts/token/ERC20/extensions/ERC20Burnable.sol";
import {ERC20Pausable} from "@openzeppelin/contracts/token/ERC20/extensions/ERC20Pausable.sol";
import {ERC20Permit} from "@openzeppelin/contracts/token/ERC20/extensions/ERC20Permit.sol";
import {AccessControl} from "@openzeppelin/contracts/access/AccessControl.sol";

contract ProjectToken is ERC20, ERC20Burnable, ERC20Pausable, ERC20Permit, AccessControl {
    bytes32 public constant MINTER_ROLE = keccak256("MINTER_ROLE");
    bytes32 public constant PAUSER_ROLE = keccak256("PAUSER_ROLE");
    uint256 public constant MAX_SUPPLY = 1_000_000_000 * 1e18;

    error ExceedsMaxSupply(uint256 requested, uint256 available);

    constructor(address admin) ERC20("ProjectToken", "PTK") ERC20Permit("ProjectToken") {
        _grantRole(DEFAULT_ADMIN_ROLE, admin); _grantRole(MINTER_ROLE, admin); _grantRole(PAUSER_ROLE, admin);
    }

    function mint(address to, uint256 amount) external onlyRole(MINTER_ROLE) {
        uint256 avail = MAX_SUPPLY - totalSupply();
        if (amount > avail) revert ExceedsMaxSupply(amount, avail);
        _mint(to, amount);
    }
    function pause() external onlyRole(PAUSER_ROLE) { _pause(); }
    function unpause() external onlyRole(PAUSER_ROLE) { _unpause(); }
    function _update(address from, address to, uint256 value)
        internal override(ERC20, ERC20Pausable) { super._update(from, to, value); }
}
```

## Proxy Upgrade Pattern

UUPS V1 with storage gap, then V2 adding a withdrawal fee. Shows safe storage extension.

```solidity
// V1: initial UUPS implementation
contract VaultV1 is Initializable, UUPSUpgradeable, OwnableUpgradeable {
    mapping(address => uint256) public balances;
    uint256 public totalDeposits;
    error InsufficientBalance(uint256 requested, uint256 available);
    error TransferFailed();

    constructor() { _disableInitializers(); } /// @custom:oz-upgrades-unsafe-allow constructor

    function initialize(address owner) external initializer {
        __Ownable_init(owner);
        __UUPSUpgradeable_init();
    }

    function deposit() external payable { balances[msg.sender] += msg.value; totalDeposits += msg.value; }

    function withdraw(uint256 amount) external virtual {
        if (balances[msg.sender] < amount) revert InsufficientBalance(amount, balances[msg.sender]);
        balances[msg.sender] -= amount;
        totalDeposits -= amount;
        (bool ok, ) = msg.sender.call{value: amount}("");
        if (!ok) revert TransferFailed();
    }

    function _authorizeUpgrade(address) internal override onlyOwner {}
    uint256[48] private __gap; // reserve for future upgrades
}

// V2: append new state after existing vars, use reinitializer, reduce gap
contract VaultV2 is VaultV1 {
    uint256 public withdrawalFeeBps; // consumes 1 gap slot
    address public feeRecipient;     // consumes 1 gap slot

    function initializeV2(address r, uint256 bps) external reinitializer(2) {
        feeRecipient = r; withdrawalFeeBps = bps;
    }

    function withdraw(uint256 amount) external override {
        if (balances[msg.sender] < amount) revert InsufficientBalance(amount, balances[msg.sender]);
        uint256 fee = (amount * withdrawalFeeBps) / 10_000;
        balances[msg.sender] -= amount; totalDeposits -= amount;
        (bool ok, ) = msg.sender.call{value: amount - fee}(""); if (!ok) revert TransferFailed();
        if (fee > 0) { (bool f, ) = feeRecipient.call{value: fee}(""); if (!f) revert TransferFailed(); }
    }
    uint256[46] private __gap; // 48 - 2 = 46
}
```

## Gas Optimization Examples

### Storage Packing

```solidity
// BAD: bool between uint256s wastes a full slot
// GOOD: group small types together — address (20B) + bool (1B) share one slot
contract Packed {
    uint256 amount;     // slot 0
    uint256 timestamp;  // slot 1
    address owner;      // slot 2 (20 bytes)
    bool isActive;      // slot 2 (1 byte, packed with owner)
}
```

### Calldata, Caching, and Unchecked

```solidity
// calldata avoids copy, cached SLOAD, unchecked loop counter
function batchTransfer(address[] calldata recipients, uint256 amount) external {
    uint256 bal = balances[msg.sender]; // single SLOAD
    uint256 total = amount * recipients.length;
    if (bal < total) revert InsufficientBalance(total, bal);
    unchecked { balances[msg.sender] = bal - total; }
    for (uint256 i = 0; i < recipients.length; ) {
        balances[recipients[i]] += amount;
        unchecked { ++i; }
    }
}
```
