const hre = require("hardhat");
const { expect } = require("chai");

describe("BerithSwap", function () {
  let berithSwap;
  let owner;
  let abi =
    require("../artifacts/contracts/berith-swap.sol/BerithSwap.json").abi;
  let iface = new ethers.Interface(abi);

  describe("deposit", function () {
    before(async () => {
      [owner, nonOwner] = await ethers.getSigners();

      const BerithSwap = await ethers.getContractFactory("BerithSwap");
      berithSwap = await BerithSwap.deploy();
      // await berithSwap.deployed();
    });

    it("should allow deposit with 0 address", async function () {
      const initialNonce = await berithSwap.depositNonce();
      // Deposit with 0 address
      const result = await berithSwap
        .connect(owner)
        .deposit(ethers.ZeroAddress, {
          value: ethers.parseEther("1"),
        });

      const newNonce = await berithSwap.depositNonce();
      // Check if the nonce increased
      expect(Number(newNonce) - Number(initialNonce)).equal(1);

      // Check if the Deposit event was emitted with the correct parameters
      const receipt = await ethers.provider.getTransactionReceipt(result.hash);
      const depositEvent = receipt.logs.find(
        (log) => log.topics[0] === ethers.id("Deposit(uint64,address)")
      );

      console.log(depositEvent.topics[1]);
      console.log(iface.parseLog({ data: depositEvent.topics[1] }));

      expect(depositEvent.args.receipient).to.equal(ethers.ZeroAddress);
    });
  });
});
