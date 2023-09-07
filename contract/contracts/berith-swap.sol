// SPDX-License-Identifier: GPL-3.0-or-later
pragma solidity ^0.8.1;

import "@openzeppelin/contracts/access/Ownable.sol";
import "@openzeppelin/contracts/utils/Context.sol";

contract BerithSwap is Ownable {
    uint64 public depositNonce;

    event Deposit(
        uint64 depositNonce,
        address indexed user,
        address receipient,
        uint256 amount
    );

    function deposit(address receipientAddress) external payable {
        address rec;
        address sender = _msgSender();

        if (receipientAddress == address(0)) {
            rec = _msgSender();
        } else {
            rec = receipientAddress;
        }

        depositNonce++;
        emit Deposit(depositNonce, sender, rec, msg.value);
    }

    function weiToEther(uint256 weiAmount) private pure returns (uint256) {
        return weiAmount / 1 ether;
    }

    function transferFunds() external onlyOwner {
        address owner = owner();
        payable(owner).transfer(address(this).balance);
    }
}
