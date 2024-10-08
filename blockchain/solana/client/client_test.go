package client_test

import (
	"context"
	"crypto/ed25519"
	"fmt"
	"testing"

	solana_sdk "github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/stretchr/testify/suite"

	"github.com/openweb3-io/crosschain/blockchain/solana"
	"github.com/openweb3-io/crosschain/blockchain/solana/builder"
	"github.com/openweb3-io/crosschain/blockchain/solana/client"
	xcbuilder "github.com/openweb3-io/crosschain/builder"
	xc_types "github.com/openweb3-io/crosschain/types"

	"github.com/openweb3-io/crosschain/types"
)

var (
	//DBomk9vPzgLWpDBvvQpJUAB1aFz8EHsPq6xEuA1cGMcV
	senderPrivkeyStr = "3PiA3WZuqGKv1E5aGWYfjsYVXZWLEJiUzGtFHZ8SNXUkLBbX9goGAHouEhTeFGUiBXVvRkfkHRga7XPENyJ7c3nq"
	senderPrivateKey solana_sdk.PrivateKey

	//8FLngQGnatEDQwNBV27yFxuWDhvQfriaCL56fx84TxoN
	recipientPrivk      = "2vLh8LUmwr9LVbFrJXKLcYcgMXAy52X6PHqZ9yhLvVfW1Fz3k1uJjheLcpUvum5oLYv8xZX5AnEXoMAEZMUMLyja"
	recipientPrivateKey solana_sdk.PrivateKey

	feePayerPrivk      = recipientPrivk
	feePayerPrivateKey solana_sdk.PrivateKey
)

type ClientTestSuite struct {
	suite.Suite
	client *client.Client
}

func (suite *ClientTestSuite) SetupTest() {
	//testnet
	client, err := client.NewClient(&types.ChainConfig{
		URL: rpc.TestNet_RPC,
	})
	suite.Require().NoError(err)
	suite.client = client

	senderPrivateKey = solana_sdk.MustPrivateKeyFromBase58(senderPrivkeyStr)
	fmt.Printf("sender address: %s \nprivate: %s\n", senderPrivateKey.PublicKey(), senderPrivateKey)

	// recipientPrivateKey = solana_sdk.NewWallet().PrivateKey
	recipientPrivateKey = solana_sdk.MustPrivateKeyFromBase58(recipientPrivk)
	fmt.Printf("recipient address: %s \nprivate: %s\n", recipientPrivateKey.PublicKey(), recipientPrivateKey)

	feePayerPrivateKey = solana_sdk.MustPrivateKeyFromBase58(feePayerPrivk)
	fmt.Printf("feePayer address: %s \nprivate: %s\n", feePayerPrivateKey.PublicKey(), feePayerPrivateKey)

}

func TestClientTestSuite(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
}

func (suite *ClientTestSuite) TestTranfser() {
	ctx := context.Background()

	builder, err := builder.NewTxBuilder(&xc_types.ChainConfig{})
	suite.Require().NoError(err)

	args, err := xcbuilder.NewTransferArgs(
		types.Address(senderPrivateKey.PublicKey().String()),
		types.Address(recipientPrivateKey.PublicKey().String()), // must exist
		types.NewBigIntFromInt64(35),
	)
	suite.Require().NoError(err)

	input, err := suite.client.FetchTransferInput(ctx, args)
	suite.Require().NoError(err)

	tx, err := builder.NewTransfer(args, input)
	suite.Require().NoError(err)

	fee, err := suite.client.EstimateGas(ctx, tx)
	suite.Require().NoError(err)
	fmt.Printf("estimate SOL gas: %v\n", fee)

	privateKey := ed25519.PrivateKey(solana_sdk.MustPrivateKeyFromBase58(senderPrivkeyStr))
	signer := solana.NewLocalSigner(privateKey)

	sighashes, err := tx.Sighashes()
	suite.Require().NoError(err)
	suite.Require().Equal(len(sighashes), 1)

	signature, err := signer.Sign(sighashes[0])
	suite.Require().NoError(err)

	err = tx.AddSignatures(signature)
	suite.Require().NoError(err)

	err = suite.client.BroadcastTx(ctx, tx)
	suite.Require().NoError(err)

	fmt.Printf("tx hash: %s\n", tx.Hash())
}

func (suite *ClientTestSuite) TestSPLTranfser() {
	ctx := context.Background()

	args, err := xcbuilder.NewTransferArgs(
		types.Address(senderPrivateKey.PublicKey().String()),          //这里填写sol的主地址，转账时程序自动找到合约的关联账户地址
		types.Address("AyqkhCrb8gt3PqiVMCshSy4to8wQcHzXtfCKbJ42qJLp"), //这里写sol的主地址，自动会创建关联地址
		types.NewBigIntFromInt64(35),
		xcbuilder.WithAsset(&types.TokenAssetConfig{
			Contract: "Gh9ZwEmdLJ8DscKNTkTqPbNwLNNBjuSzaG9Vp2KGtKJr",
			Decimals: 6,
		}),
	)
	suite.Require().NoError(err)

	input, err := suite.client.FetchTransferInput(ctx, args)
	suite.Require().NoError(err)

	builder, err := builder.NewTxBuilder(&xc_types.ChainConfig{})
	suite.Require().NoError(err)

	tx, err := builder.NewTokenTransfer(args, input)
	suite.Require().NoError(err)

	fee, err := suite.client.EstimateGas(ctx, tx)
	suite.Require().NoError(err)
	fmt.Printf("estimate SOL gas: %v\n", fee)

	privateKey := ed25519.PrivateKey(solana_sdk.MustPrivateKeyFromBase58(senderPrivkeyStr))
	signer := solana.NewLocalSigner(privateKey)

	sighashes, err := tx.Sighashes()
	suite.Require().NoError(err)
	suite.Require().Equal(len(sighashes), 1)

	signature, err := signer.Sign(sighashes[0])
	suite.Require().NoError(err)

	err = tx.AddSignatures(signature)
	suite.Require().NoError(err)

	err = suite.client.BroadcastTx(ctx, tx)
	suite.Require().NoError(err)

	fmt.Printf("tx hash: %s\n", tx.Hash())
}

func (suite *ClientTestSuite) TestSPLTranfserSetFeePayer() {
	ctx := context.Background()

	feePayer := feePayerPrivateKey.PublicKey().String()

	args, err := xcbuilder.NewTransferArgs(
		types.Address(senderPrivateKey.PublicKey().String()), //这里填写sol的主地址，转账时程序自动找到合约的关联账户地址
		types.Address(recipientPrivateKey.PublicKey().String()),
		//types.Address("AyqkhCrb8gt3PqiVMCshSy4to8wQcHzXtfCKbJ42qJLp"), //这里写sol的主地址，自动会创建关联地址
		types.NewBigIntFromInt64(1),
		xcbuilder.WithAsset(&types.TokenAssetConfig{
			Contract: "Gh9ZwEmdLJ8DscKNTkTqPbNwLNNBjuSzaG9Vp2KGtKJr",
			Decimals: 6,
		}),
		xcbuilder.WithFeePayer(types.Address(feePayer)),
	)
	suite.Require().NoError(err)

	input, err := suite.client.FetchTransferInput(ctx, args)
	suite.Require().NoError(err)

	builder, err := builder.NewTxBuilder(&xc_types.ChainConfig{})
	suite.Require().NoError(err)

	tx, err := builder.NewTokenTransfer(args, input)
	suite.Require().NoError(err)

	fee, err := suite.client.EstimateGas(ctx, tx)
	suite.Require().NoError(err)
	fmt.Printf("estimate SOL gas: %v\n", fee)

	privateKey := ed25519.PrivateKey(solana_sdk.MustPrivateKeyFromBase58(senderPrivkeyStr))
	signer := solana.NewLocalSigner(privateKey)

	feePrivateKey := ed25519.PrivateKey(feePayerPrivateKey)
	feeSigner := solana.NewLocalSigner(feePrivateKey)

	sighashes, err := tx.Sighashes()
	suite.Require().NoError(err)
	suite.Require().Equal(len(sighashes), 1)

	//fee
	signatureFee, err := feeSigner.Sign(sighashes[0])
	suite.Require().NoError(err)

	signature, err := signer.Sign(sighashes[0])
	suite.Require().NoError(err)

	err = tx.AddSignatures(signatureFee, signature)
	suite.Require().NoError(err)

	err = suite.client.BroadcastTx(ctx, tx)
	suite.Require().NoError(err)

	fmt.Printf("tx hash: %s\n", tx.Hash())
}

func (suite *ClientTestSuite) TestFetchBalance() {
	ctx := context.Background()

	contractAddress := "Gh9ZwEmdLJ8DscKNTkTqPbNwLNNBjuSzaG9Vp2KGtKJr"

	out, err := suite.client.FetchBalance(ctx, types.Address(senderPrivateKey.PublicKey().String()))
	suite.Require().NoError(err)
	fmt.Printf("\n %s SOL balance: %v", senderPrivateKey.PublicKey().String(), out)

	out, err = suite.client.FetchBalanceForAsset(ctx, types.Address(senderPrivateKey.PublicKey().String()), types.ContractAddress(contractAddress))
	suite.Require().NoError(err)

	fmt.Printf("\n %s SPL token balance: %v", senderPrivateKey.PublicKey().String(), out)
}
