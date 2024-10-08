package builder

import (
	"errors"

	"github.com/gagliardetto/solana-go"
	ata "github.com/gagliardetto/solana-go/programs/associated-token-account"
	compute_budget "github.com/gagliardetto/solana-go/programs/compute-budget"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/gagliardetto/solana-go/programs/token"
	"github.com/openweb3-io/crosschain/blockchain/solana/tx"
	"github.com/openweb3-io/crosschain/blockchain/solana/tx_input"
	solana_types "github.com/openweb3-io/crosschain/blockchain/solana/types"
	xcbuilder "github.com/openweb3-io/crosschain/builder"
	"github.com/openweb3-io/crosschain/types"
	xc_types "github.com/openweb3-io/crosschain/types"
)

// Max number of token transfers we can fit in a solana transaction,
// when there's also a create ATA included.
const MaxTokenTransfers = 20
const MaxAccountUnstakes = 20
const MaxAccountWithdraws = 20

type TxBuilder struct {
	Chain *xc_types.ChainConfig
}

func NewTxBuilder(chain *xc_types.ChainConfig) (*TxBuilder, error) {
	return &TxBuilder{
		Chain: chain,
	}, nil
}

func (b *TxBuilder) NewTransfer(args *xcbuilder.TransferArgs, input types.TxInput) (types.Tx, error) {
	asset, _ := args.GetAsset()
	if asset == nil {
		asset = b.Chain
	}

	switch asset.(type) {
	case *types.TokenAssetConfig:
		return b.NewTokenTransfer(args, input)
	default:
		return b.NewNativeTransfer(args, input)
	}
}

func (b *TxBuilder) NewTokenTransfer(args *xcbuilder.TransferArgs, input types.TxInput) (types.Tx, error) {
	txInput := input.(*tx_input.TxInput)

	asset, _ := args.GetAsset()

	if asset == nil {
		return nil, errors.New("asset missing")
	}

	contract := asset.GetContract()
	if contract == "" {
		return nil, errors.New("contract missing")
	}
	decimals := asset.GetDecimals()

	accountFrom, err := solana.PublicKeyFromBase58(string(args.GetFrom()))
	if err != nil {
		return nil, err
	}

	accountContract, err := solana.PublicKeyFromBase58(string(contract))
	if err != nil {
		return nil, err
	}

	accountTo, err := solana.PublicKeyFromBase58(string(args.GetTo()))
	if err != nil {
		return nil, err
	}

	ataFromStr, err := solana_types.FindAssociatedTokenAddress(string(args.GetTo()), string(contract), solana.PublicKey(txInput.TokenProgram))
	if err != nil {
		return nil, err
	}
	ataFrom := solana.MustPublicKeyFromBase58(ataFromStr)
	if len(txInput.SourceTokenAccounts) > 0 {
		ataFrom = txInput.SourceTokenAccounts[0].Account
	}

	ataTo := accountTo
	if !txInput.ToIsATA {
		ataToStr, err := solana_types.FindAssociatedTokenAddress(string(args.GetTo()), string(contract), solana.PublicKey(txInput.TokenProgram))
		if err != nil {
			return nil, err
		}
		ataTo = solana.MustPublicKeyFromBase58(ataToStr)
	}

	// Temporarily adjust the backend library to use a different program ID.
	// This is to support token2022 and potential other future variants.
	originalTokenId := token.ProgramID
	defer token.SetProgramID(originalTokenId)
	if !txInput.TokenProgram.IsZero() && !txInput.TokenProgram.Equals(originalTokenId) {
		token.SetProgramID(txInput.TokenProgram)
	}

	instructions := []solana.Instruction{}
	if txInput.ShouldCreateATA {
		createAta := ata.NewCreateInstruction(
			accountFrom,
			accountTo,
			accountContract,
		).Build()
		// Adjust the ata-create-account arguments:
		// index 1 - associated token account
		// index 5 - token program
		createAta.Impl.(ata.Create).AccountMetaSlice[1].PublicKey = ataTo
		createAta.Impl.(ata.Create).AccountMetaSlice[5].PublicKey = txInput.TokenProgram
		instructions = append(instructions,
			createAta,
		)
	}
	if len(txInput.SourceTokenAccounts) <= 1 {
		// just send 1 instruction using the single ATA
		instructions = append(instructions,
			token.NewTransferCheckedInstruction(
				args.GetAmount().Uint64(),
				uint8(decimals),
				ataFrom,
				accountContract,
				ataTo,
				accountFrom,
				[]solana.PublicKey{},
			).Build(),
		)
	} else {
		// Sometimes tokens can get put into any number of auxiliary accounts.
		// So we need to spend them like UTXO. Here we'll just send a solana
		// instruction for each one until we've reached the target balance.
		zero := types.NewBigIntFromUint64(0)
		remainingBalanceToSend := args.GetAmount()
		for _, tokenAcc := range txInput.SourceTokenAccounts {
			amountToSend := remainingBalanceToSend
			if tokenAcc.Balance.Cmp(&remainingBalanceToSend) < 0 {
				// Send everything in the token account
				amountToSend = tokenAcc.Balance
			}
			amountToSendUint := amountToSend.Uint64()
			instructions = append(instructions,
				token.NewTransferCheckedInstruction(
					amountToSendUint,
					uint8(decimals),
					tokenAcc.Account,
					accountContract,
					ataTo,
					accountFrom,
					[]solana.PublicKey{},
				).Build(),
			)
			remainingBalanceToSend = remainingBalanceToSend.Sub(&amountToSend)
			if remainingBalanceToSend.Cmp(&zero) <= 0 {
				// we've spent enough from source accounts to meet target balance
				break
			}
			if len(instructions) > MaxTokenTransfers {
				return nil, errors.New("cannot send total amount in single tx, try sending smaller amount")
			}
		}
		if remainingBalanceToSend.Cmp(&zero) > 0 {
			return nil, errors.New("cannot send requested amount in single tx, try sending smaller amount")
		}
	}

	// add priority fee last
	priorityFee := txInput.GetLimitedPrioritizationFee(b.Chain)
	if priorityFee > 0 {
		instructions = append(instructions,
			compute_budget.NewSetComputeUnitPriceInstruction(priorityFee).Build(),
		)
	}

	var feePayer solana.PublicKey
	if fee, has := args.GetFeePayer(); has {
		feePayer, err = solana.PublicKeyFromBase58(string(fee))
		if err != nil {
			return nil, err
		}
	} else {
		feePayer = accountFrom
	}

	return b.buildSolanaTx(instructions, &feePayer, txInput)
}

func (txBuilder TxBuilder) buildSolanaTx(instructions []solana.Instruction, feePayer *solana.PublicKey, txInput *tx_input.TxInput) (*tx.Tx, error) {
	var opts []solana.TransactionOption
	if feePayer != nil {
		opts = append(opts, solana.TransactionPayer(*feePayer))
	}

	tx1, err := solana.NewTransaction(
		instructions,
		txInput.RecentBlockHash,
		opts...,
	)
	if err != nil {
		return nil, err
	}
	return &tx.Tx{
		SolTx: tx1,
	}, nil
}

func (b *TxBuilder) NewNativeTransfer(args *xcbuilder.TransferArgs, input types.TxInput) (types.Tx, error) {
	txInput := input.(*tx_input.TxInput)

	accountFrom, err := solana.PublicKeyFromBase58(string(args.GetFrom()))
	if err != nil {
		return nil, err
	}

	accountTo, err := solana.PublicKeyFromBase58(string(args.GetTo()))
	if err != nil {
		return nil, err
	}

	instructions := []solana.Instruction{
		system.NewTransferInstruction(
			args.GetAmount().Int().Uint64(),
			accountFrom,
			accountTo,
		).Build(),
	}

	prioprityFee := txInput.GetLimitedPrioritizationFee(b.Chain)
	if prioprityFee > 0 {
		instructions = append(instructions, compute_budget.NewSetComputeUnitPriceInstruction(prioprityFee).Build())
	}

	solTx, err := solana.NewTransaction(
		instructions,
		txInput.RecentBlockHash,
	)
	if err != nil {
		return nil, err
	}

	return &tx.Tx{
		SolTx: solTx,
	}, nil
}
