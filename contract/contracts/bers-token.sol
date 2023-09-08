// SPDX-License-Identifier: GPL-3.0-or-later
pragma solidity ^0.8.1;

import "@openzeppelin/contracts/token/ERC20/ERC20.sol";
import "@openzeppelin/contracts/access/Ownable.sol";

contract BersToken is Ownable, ERC20 {
    constructor(uint256 _totalSupply) ERC20("BersSwap", "BSP") {
        _mint(msg.sender, _totalSupply);
    }

    function decimals() public view virtual override returns (uint8) {
        return 0;
    }
}
