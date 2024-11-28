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
var tokenAddr = common.Address{}

const (
	mtHeight = 32
)

//go:embed depositUsage.md
var depositUsage string
var depositCmd = &cobra.Command{
	Use:     "deposit",
	Short:   "make a deposit bridge transaction",
	Long:    depositUsage,
	Args:    cobra.NoArgs,
	PreRunE: checkDepositArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Convert the string input to uint32
		destNetworkUint, err := strconv.ParseUint(*bridgeInputArgs.DestNetwork, 10, 32)
		if err != nil {
			fmt.Println("Error parsing input:", err)
			return nil
		}

		// Since ParseUint returns an uint64, cast it to uint32
		destNetwork32 := uint32(destNetworkUint)

		// Create a new big.Int
		amount := new(big.Int)

		// Convert the hexadecimal string to big.Int
		_, success := amount.SetString(*bridgeInputArgs.FundsToSend, 10)
		if !success {
			fmt.Println("Error: Invalid string format for big.Int")
			return nil
		}

		ctx := context.Background()
		client, err := utils.NewClient(ctx, *bridgeInputArgs.OriginNetworkURL, common.HexToAddress(*bridgeInputArgs.OriginBridgeAddr))
		if err != nil {
			log.Fatal("Error: ", err)
		}
		auth, err := client.GetSigner(ctx, *bridgeInputArgs.OriginAccHexPrivateKey)
		if err != nil {
			log.Fatal("Error: ", err)
		}
		emptyAddr := common.Address{}
		if tokenAddr == emptyAddr {
			auth.Value = amount
		}
		destAddr := common.HexToAddress(*bridgeInputArgs.OriginAccHexAddress)
		log.Info("Sending bridge tx...")
		err = client.SendBridgeAsset(ctx, tokenAddr, amount, destNetwork32, &destAddr, []byte{}, auth)
		if err != nil {
			log.Fatal("Error: ", err)
		}
		log.Info("Success!")
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
		ctx := context.Background()
		c, err := utils.NewClient(ctx, *bridgeInputArgs.DestNetworkURL, common.HexToAddress(*bridgeInputArgs.DestBridgeAddr))
		if err != nil {
			log.Fatal("Error: ", err)
		}
		auth, err := c.GetSigner(ctx, *bridgeInputArgs.DestAccHexPrivateKey)
		if err != nil {
			log.Fatal("Error: ", err)
		}

		// Get Claim data
		cfg := clientUtils.Config{
			L1NodeURL:    *bridgeInputArgs.DestNetworkURL,
			L2NodeURL:    *bridgeInputArgs.DestNetworkURL,
			BridgeURL:    *bridgeInputArgs.BridgeURL,
			L2BridgeAddr: common.HexToAddress(*bridgeInputArgs.DestBridgeAddr),
		}
		client, err := clientUtils.NewClient(ctx, cfg)
		if err != nil {
			log.Fatal("Error: ", err)
		}
		deposits, _, err := client.GetBridges(*bridgeInputArgs.DestAccHexAddress, 0, 10) //nolint
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
		balance, err := c.Client.BalanceAt(ctx, common.HexToAddress(*bridgeInputArgs.DestAccHexAddress), nil)
		if err != nil {
			log.Fatal("error getting balance: ", err)
		}
		log.Info("L2 balance: ", balance)
		return nil
	},
}

func checkDepositArgs(cmd *cobra.Command, args []string) error {
	// if *ulxlyInputArgs.DepositBridgeAddress == "" {
	// 	return fmt.Errorf("please provide the bridge address")
	// }
	return nil
}

func checkClaimArgs(cmd *cobra.Command, args []string) error {
	// if *ulxlyInputArgs.ClaimGasLimit < 150000 && *ulxlyInputArgs.ClaimGasLimit != 0 {
	// 	return fmt.Errorf("the gas limit may be too low for the transaction to pass")
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
	bridgeInputArgs.DestNetworkURL = claimCmd.PersistentFlags().String("destination-network-url", "http://127.0.0.1:8545", "")
	bridgeInputArgs.BridgeURL = claimCmd.PersistentFlags().String("bridge-url", "http://127.0.0.1:8080", "")

}
