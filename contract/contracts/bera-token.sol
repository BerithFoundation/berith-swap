// SPDX-License-Identifier: GPL-3.0-or-later
pragma solidity ^0.8.1;

import "@openzeppelin/contracts/token/ERC20/ERC20.sol";

contract BeraToken is ERC20{
    constructor(uint256 _totalSupply) ERC20("Bitmoi", "MOI") {
        _mint(msg.sender, _totalSupply);
    }
}