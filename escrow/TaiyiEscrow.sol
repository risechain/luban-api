// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

interface TaiyiEscrow {
    event Deposited(address indexed user, uint256 amount);
    event Withdrawn(address indexed user, uint256 amount);
    event PaymentMade(address indexed from, uint256 amount, bool isAfterExec);

    function balanceOf(address user) external view returns (uint256);

    function deposit() external payable;

    function withdraw(uint256 amount) external;
}
