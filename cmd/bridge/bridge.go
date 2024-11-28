package bridge

import (
	"context"
	"fmt"
	"math/big"
	"strconv"

	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	"github.com/0xPolygonHermez/zkevm-bridge-service/log"
	clientUtils "github.com/0xPolygonHermez/zkevm-bridge-service/test/client"
	"github.com/0xPolygonHermez/zkevm-bridge-service/utils"
	"github.com/ethereum/go-ethereum/common"

	"github.com/spf13/cobra"
)

type bridgeArgs struct {
	OriginBridgeAddr       *string
	OriginAccHexAddress    *string
	OriginAccHexPrivateKey *string
	OriginNetworkURL       *string
	FundsToSend            *string
	DestNetwork            *string

	DestBridgeAddr       *string
	DestAccHexAddress    *string
	DestAccHexPrivateKey *string
	DestNetworkURL       *string
	BridgeURL            *string
}

var bridgeInputArgs bridgeArgs

var BridgeCmd = &cobra.Command{
	Use:   "bridge",
	Short: "Utilities for interacting with the bridge service",
	Long:  "These are low level tools for using the bridge service directly.",
	Args:  cobra.NoArgs,
}

//go:embed depositUsage.md
var depositUsage string
var depositCmd = &cobra.Command{
	Use:     "deposit",
	Short:   "make a deposit bridge transaction",
	Long:    depositUsage,
	Args:    cobra.NoArgs,
	PreRunE: checkDepositArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		deposit()
		return nil
	},
}

//go:embed depositUsage.md
var claimUsage string
var claimCmd = &cobra.Command{
	Use:     "claim",
	Short:   "make a claim bridge transaction",
	Long:    claimUsage,
	Args:    cobra.NoArgs,
	PreRunE: checkClaimArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		claim()
		return nil
	},
}

var tokenAddr = common.Address{}

func deposit() {
	// // Check if sufficient arguments are provided
	// if len(os.Args) < 7 {
	// 	fmt.Println("Usage: <program_name> <originBridgeAddr> <originAccHexAddress> <originAccHexPrivateKey> <originNetworkURL> <fundsToSend> <destNetwork>")
	// 	return
	// }
	// // Get the name and age from the command-line arguments
	// originBridgeAddr := os.Args[1]
	// originAccHexAddress := os.Args[2]
	// originAccHexPrivateKey := os.Args[3]
	// originNetworkURL := os.Args[4]
	// fundsToSend := os.Args[5]
	// destNetwork := os.Args[6]

	// Convert the string input to uint32
	destNetworkUint, err := strconv.ParseUint(destNetwork, 10, 32)
	if err != nil {
		fmt.Println("Error parsing input:", err)
		return
	}

	// Since ParseUint returns an uint64, cast it to uint32
	destNetwork32 := uint32(destNetworkUint)

	// Create a new big.Int
	amount := new(big.Int)

	// Convert the hexadecimal string to big.Int
	_, success := amount.SetString(fundsToSend, 10)
	if !success {
		fmt.Println("Error: Invalid string format for big.Int")
		return
	}

	ctx := context.Background()
	client, err := utils.NewClient(ctx, originNetworkURL, common.HexToAddress(originBridgeAddr))
	if err != nil {
		log.Fatal("Error: ", err)
	}
	auth, err := client.GetSigner(ctx, originAccHexPrivateKey)
	if err != nil {
		log.Fatal("Error: ", err)
	}
	emptyAddr := common.Address{}
	if tokenAddr == emptyAddr {
		auth.Value = amount
	}
	destAddr := common.HexToAddress(originAccHexAddress)
	log.Info("Sending bridge tx...")
	err = client.SendBridgeAsset(ctx, tokenAddr, amount, destNetwork32, &destAddr, []byte{}, auth)
	if err != nil {
		log.Fatal("Error: ", err)
	}
	log.Info("Success!")
}

const (
	mtHeight = 32
)

func claim() {
	// // Check if sufficient arguments are provided
	// if len(os.Args) < 6 {
	// 	fmt.Println("Usage: <program_name> <destBridgeAddr> <destAccHexAddress> <destAccHexPrivateKey> <destNetworkURL> <bridgeURL>")
	// 	return
	// }
	// // Get the name and age from the command-line arguments
	// destBridgeAddr := os.Args[1]
	// destAccHexAddress := os.Args[2]
	// destAccHexPrivateKey := os.Args[3]
	// destNetworkURL := os.Args[4]
	// bridgeURL := os.Args[5]

	ctx := context.Background()
	c, err := utils.NewClient(ctx, destNetworkURL, common.HexToAddress(destBridgeAddr))
	if err != nil {
		log.Fatal("Error: ", err)
	}
	auth, err := c.GetSigner(ctx, destAccHexPrivateKey)
	if err != nil {
		log.Fatal("Error: ", err)
	}

	// Get Claim data
	cfg := clientUtils.Config{
		L1NodeURL:    destNetworkURL,
		L2NodeURL:    destNetworkURL,
		BridgeURL:    bridgeURL,
		L2BridgeAddr: common.HexToAddress(destBridgeAddr),
	}
	client, err := clientUtils.NewClient(ctx, cfg)
	if err != nil {
		log.Fatal("Error: ", err)
	}
	deposits, _, err := client.GetBridges(destAccHexAddress, 0, 10) //nolint
	if err != nil {
		log.Fatal("Error: ", err)
	}
	bridgeData := deposits[0]
	proof, err := client.GetMerkleProof(deposits[0].NetworkId, deposits[0].DepositCnt)
	if err != nil {
		log.Fatal("error: ", err)
	}
	log.Debug("bridge: ", bridgeData)
	log.Debug("mainnetExitRoot: ", proof.MainExitRoot)
	log.Debug("rollupExitRoot: ", proof.RollupExitRoot)

	var smtProof, smtRollupProof [mtHeight][32]byte
	for i := 0; i < len(proof.MerkleProof); i++ {
		log.Debug("smtProof: ", proof.MerkleProof[i])
		smtProof[i] = common.HexToHash(proof.MerkleProof[i])
		log.Debug("smtRollupProof: ", proof.RollupMerkleProof[i])
		smtRollupProof[i] = common.HexToHash(proof.RollupMerkleProof[i])
	}
	globalExitRoot := &etherman.GlobalExitRoot{
		ExitRoots: []common.Hash{common.HexToHash(proof.MainExitRoot), common.HexToHash(proof.RollupExitRoot)},
	}
	log.Info("Sending claim tx...")
	err = c.SendClaim(ctx, bridgeData, smtProof, smtRollupProof, globalExitRoot, auth)
	if err != nil {
		log.Fatal("error: ", err)
	}
	log.Info("Success!")
	balance, err := c.Client.BalanceAt(ctx, common.HexToAddress(destAccHexAddress), nil)
	if err != nil {
		log.Fatal("error getting balance: ", err)
	}
	log.Info("L2 balance: ", balance)
}

func checkDepositArgs(cmd *cobra.Command, args []string) error {
	// if *ulxlyInputArgs.DepositBridgeAddress == "" {
	// 	return fmt.Errorf("please provide the bridge address")
	// }
	// if *ulxlyInputArgs.DepositGasLimit < 130000 && *ulxlyInputArgs.DepositGasLimit != 0 {
	// 	return fmt.Errorf("the gas limit may be too low for the transaction to pass")
	// }
	return nil
}

func checkClaimArgs(cmd *cobra.Command, args []string) error {
	// if *ulxlyInputArgs.ClaimGasLimit < 150000 && *ulxlyInputArgs.ClaimGasLimit != 0 {
	// 	return fmt.Errorf("the gas limit may be too low for the transaction to pass")
	// }
	// if *ulxlyInputArgs.ClaimMessage && *ulxlyInputArgs.ClaimWETH {
	// 	return fmt.Errorf("choose a single claim mode (asset, message, or WETH)")
	// }
	return nil
}

func init() {
	BridgeCmd.AddCommand(depositCmd)
	BridgeCmd.AddCommand(claimCmd)

	bridgeInputArgs.OriginBridgeAddr = depositCmd.PersistentFlags().String("origin-bridge-address", "0xd8886e9D827218a02B8C04323b5550f2F36BC8d5", "The bridge address of the origin network.")
	bridgeInputArgs.OriginAccHexAddress = depositCmd.PersistentFlags().String("origin-account-address", "0xE34aaF64b29273B7D567FCFc40544c014EEe9970", "")
	bridgeInputArgs.OriginAccHexPrivateKey = depositCmd.PersistentFlags().String("origin-account-private-key", "0x12d7de8621a77640c9241b2595ba78ce443d05e94090365ab3bb5e19df82c625", "")
	bridgeInputArgs.OriginNetworkURL = depositCmd.PersistentFlags().String("origin-network-url", "http://127.0.0.1:8545", "")
	bridgeInputArgs.FundsToSend = depositCmd.PersistentFlags().String("funds-to-send", "0", "")
	bridgeInputArgs.DestNetwork = depositCmd.PersistentFlags().String("destination-network", "0", "")

	bridgeInputArgs.DestBridgeAddr = claimCmd.PersistentFlags().String("destination-bridge-address", "0xd8886e9D827218a02B8C04323b5550f2F36BC8d5", "")
	bridgeInputArgs.DestAccHexAddress = claimCmd.PersistentFlags().String("destination-account-address", "0xE34aaF64b29273B7D567FCFc40544c014EEe9970", "")
	bridgeInputArgs.DestAccHexPrivateKey = claimCmd.PersistentFlags().String("destination-account-private-key", "0x12d7de8621a77640c9241b2595ba78ce443d05e94090365ab3bb5e19df82c625", "")
	bridgeInputArgs.DestNetworkURL = claimCmd.PersistentFlags().String("destination-network-url", "0", "")
	bridgeInputArgs.BridgeURL = claimCmd.PersistentFlags().String("bridge-url", "http://127.0.0.1:8080", "")

}
