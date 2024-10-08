package btc_test

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"testing"

	. "github.com/openweb3-io/crosschain/blockchain/btc"
	"github.com/openweb3-io/crosschain/blockchain/btc/address"
	"github.com/openweb3-io/crosschain/blockchain/btc/tx"
	"github.com/openweb3-io/crosschain/blockchain/btc/tx_input"
	xcbuilder "github.com/openweb3-io/crosschain/builder"
	xc "github.com/openweb3-io/crosschain/types"
	"github.com/stretchr/testify/suite"
)

var UXTO_ASSETS []xc.NativeAsset = []xc.NativeAsset{
	xc.BTC,
	xc.DOGE,
	xc.LTC,
}

type CrosschainTestSuite struct {
	suite.Suite
	Ctx context.Context
}

func (s *CrosschainTestSuite) SetupTest() {
	s.Ctx = context.Background()
}

func TestBitcoinTestSuite(t *testing.T) {
	suite.Run(t, new(CrosschainTestSuite))
}

// Address

func (s *CrosschainTestSuite) TestNewAddressBuilder() {
	require := s.Require()
	for _, nativeAsset := range UXTO_ASSETS {
		builder, err := address.NewAddressBuilder(&xc.ChainConfig{Chain: nativeAsset})
		require.NotNil(builder)
		require.NoError(err)
	}
}

func (s *CrosschainTestSuite) TestGetAddressFromPublicKey() {
	require := s.Require()
	for _, nativeAsset := range UXTO_ASSETS {
		builder, err := address.NewAddressBuilder(&xc.ChainConfig{
			Network:    "testnet",
			Chain:      nativeAsset,
			Blockchain: nativeAsset.Blockchain(),
		})
		require.NoError(err)
		pubkey, err := base64.RawStdEncoding.DecodeString("AptrsfXbXbvnsWxobWNFoUXHLO5nmgrQb3PDmGGu1CSS")
		require.NoError(err)
		fmt.Println("checking address for ", nativeAsset)
		switch nativeAsset {
		case xc.BTC:
			address, err := builder.GetAddressFromPublicKey(pubkey)
			require.NoError(err)
			// BTC should use newest address type, segwit
			require.Equal(xc.Address("tb1qzca49vcyxkt989qcmhjfp7wyze7n9pq50k2cfd"), address)
		case xc.DOGE:
			address, err := builder.GetAddressFromPublicKey(pubkey)
			require.NoError(err)
			require.Equal(xc.Address("nWDiCL2RxZcMTvhUGRWCnPDWFWHSCfkhoz"), address)
		case xc.LTC:
			address, err := builder.GetAddressFromPublicKey(pubkey)
			require.NoError(err)
			require.Equal(xc.Address("mhYWE7RrYCgbq4RJDaqZp8fvzVmYnPVnFD"), address)
		default:
			panic("need to add address test case for " + nativeAsset)
		}
	}
}
func (s *CrosschainTestSuite) TestGetAddressFromPublicKeyUsesCompressed() {
	require := s.Require()
	builder, err := address.NewAddressBuilder(&xc.ChainConfig{
		Network:    "testnet",
		Chain:      xc.BTC,
		Blockchain: xc.BlockchainBtc,
	})
	require.NoError(err)
	compressedPubkey, _ := hex.DecodeString("0228a9dd8c304464e0d0f011ca3dccb0e373afd2f5c51e89113b8be2a905687fb9")
	uncompressedPubkey, _ := hex.DecodeString("0428a9dd8c304464e0d0f011ca3dccb0e373afd2f5c51e89113b8be2a905687fb967cf9090845d6e8cac68f7bedf4335ed946c678b371c8cad7dbd5f63f1a9e992")

	addressCompressed, _ := builder.GetAddressFromPublicKey(compressedPubkey)
	addressUncompressed, _ := builder.GetAddressFromPublicKey(uncompressedPubkey)

	require.EqualValues("tb1q6y6kkfsrzhlex4u8eel436cyh26qmlmjxgwrel", addressCompressed)
	require.EqualValues("tb1q6y6kkfsrzhlex4u8eel436cyh26qmlmjxgwrel", addressUncompressed)
}

func (s *CrosschainTestSuite) TestGetAllPossibleAddressesFromPublicKey() {
	require := s.Require()
	builder, err := address.NewAddressBuilder(&xc.ChainConfig{
		Network:    "testnet",
		Chain:      "BTC",
		Blockchain: xc.BTC.Blockchain(),
	})
	require.NoError(err)
	pubkey, err := base64.RawStdEncoding.DecodeString("AptrsfXbXbvnsWxobWNFoUXHLO5nmgrQb3PDmGGu1CSS")
	require.NoError(err)
	addresses, err := builder.GetAllPossibleAddressesFromPublicKey(pubkey)
	require.NoError(err)

	validated_p2pkh := false
	validated_p2wkh := false

	fmt.Println(addresses)
	for _, addr := range addresses {
		if addr.Address == "mhYWE7RrYCgbq4RJDaqZp8fvzVmYnPVnFD" {
			require.Equal(xc.AddressTypeP2PKH, addr.Type)
			validated_p2pkh = true
		} else if addr.Address == "tb1qzca49vcyxkt989qcmhjfp7wyze7n9pq50k2cfd" {
			require.Equal(xc.AddressTypeP2WPKH, addr.Type)
			validated_p2wkh = true
		} else {
			// panic("unexpected address generated: " + addr.Address)
		}
	}
	require.True(validated_p2pkh)
	require.True(validated_p2wkh)
}

// TxBuilder

func (s *CrosschainTestSuite) TestNewTxBuilder() {
	require := s.Require()
	for _, nativeAsset := range UXTO_ASSETS {
		builder, err := NewTxBuilder(&xc.ChainConfig{Chain: nativeAsset})
		require.NotNil(builder)
		require.NoError(err)
	}
}

func (s *CrosschainTestSuite) TestNewNativeTransfer() {
	require := s.Require()
	for _, fromAddr := range []string{
		// legacy
		"mpjwFvP88ZwAt3wEHY6irKkGhxcsv22BP6",
		// segwit
		"tb1qhymp5maj7x2rqxsj02exqn26v5jcqm0q3x3pz4",
		// taproot (not supported)
		// "tb1p5gkytm46mtksmssryta62fejfxvh82vnqs96hnd96gwmn0ztz4esam80dt",
	} {
		for _, toAddr := range []string{
			// legacy
			"mxVFsFW5N4mu1HPkxPttorvocvzeZ7KZyk",
			// segwit
			"tb1qtguj96eqjtzt2fywyqdgmuw6wtpdsuahheqja6",
			// taproot
			"tb1p5gkytm46mtksmssryta62fejfxvh82vnqs96hnd96gwmn0ztz4esam80dt",
		} {
			for _, native_asset := range []xc.NativeAsset{
				xc.BTC,
			} {
				asset := &xc.ChainConfig{Chain: native_asset, Network: "testnet"}
				builder, _ := NewTxBuilder(asset)
				from := xc.Address(fromAddr)
				to := xc.Address(toAddr)
				amount := xc.NewBigIntFromUint64(1)
				input := &tx_input.TxInput{
					UnspentOutputs: []tx_input.Output{{
						Value: xc.NewBigIntFromUint64(1000),
					}},
					GasPricePerByte: xc.NewBigIntFromUint64(1),
				}

				args, err := xcbuilder.NewTransferArgs(from, to, amount)
				require.NoError(err)

				tf, err := builder.NewNativeTransfer(args, input)
				require.NoError(err)
				require.NotNil(tf)
				hash := tf.Hash()
				require.Len(hash, 64)

				// Having not enough balance for fees will be an error
				input_small := &tx_input.TxInput{
					UnspentOutputs: []tx_input.Output{{
						Value: xc.NewBigIntFromUint64(5),
					}},
					GasPricePerByte: xc.NewBigIntFromUint64(1),
				}
				_, err = builder.NewNativeTransfer(args, input_small)
				require.Error(err)

				// add signature
				sig := []byte{}
				for i := 0; i < 65; i++ {
					sig = append(sig, byte(i))
				}
				err = tf.AddSignatures(xc.TxSignature(sig))
				require.NoError(err)

				ser, err := tf.Serialize()
				require.NoError(err)
				require.True(len(ser) > 64)
			}
		}
	}
}

func (s *CrosschainTestSuite) TestNewTokenTransfer() {
	require := s.Require()
	asset := &xc.ChainConfig{Chain: xc.BTC, Network: "testnet"}
	builder, _ := NewTxBuilder(asset)
	from := xc.Address("mpjwFvP88ZwAt3wEHY6irKkGhxcsv22BP6")
	to := xc.Address("tb1qtpqqpgadjr2q3f4wrgd6ndclqtfg7cz5evtvs0")
	amount := xc.NewBigIntFromUint64(1)

	args, err := xcbuilder.NewTransferArgs(from, to, amount)
	require.NoError(err)

	input := &tx_input.TxInput{
		UnspentOutputs: []tx_input.Output{{
			Value: xc.NewBigIntFromUint64(1000),
		}},
		GasPricePerByte: xc.NewBigIntFromUint64(1),
	}
	tf, err := builder.NewTokenTransfer(args, input)
	require.Nil(tf)
	require.EqualError(err, "not implemented")
}

func (s *CrosschainTestSuite) TestNewTransfer() {
	require := s.Require()
	asset := &xc.ChainConfig{Chain: xc.BTC, Network: "testnet"}
	builder, _ := NewTxBuilder(asset)
	from := xc.Address("mpjwFvP88ZwAt3wEHY6irKkGhxcsv22BP6")
	to := xc.Address("tb1qtpqqpgadjr2q3f4wrgd6ndclqtfg7cz5evtvs0")
	amount := xc.NewBigIntFromUint64(1)
	args, err := xcbuilder.NewTransferArgs(from, to, amount)
	require.NoError(err)

	input := &tx_input.TxInput{
		UnspentOutputs: []tx_input.Output{{
			Value: xc.NewBigIntFromUint64(1000),
		}},
		GasPricePerByte: xc.NewBigIntFromUint64(1),
	}
	tf, err := builder.NewTransfer(args, input)
	require.NotNil(tf)
	require.NoError(err)
}

func (s *CrosschainTestSuite) TestNewTransfer_Token() {
	require := s.Require()
	builder, _ := NewTxBuilder(&xc.ChainConfig{Network: "testnet"})
	from := xc.Address("mpjwFvP88ZwAt3wEHY6irKkGhxcsv22BP6")
	to := xc.Address("tb1qtpqqpgadjr2q3f4wrgd6ndclqtfg7cz5evtvs0")
	amount := xc.BigInt{}

	args, err := xcbuilder.NewTransferArgs(
		from,
		to,
		amount,
		xcbuilder.WithAsset(
			&xc.TokenAssetConfig{Asset: string(xc.BTC)},
		),
	)
	require.NoError(err)

	input := &tx_input.TxInput{}
	tf, err := builder.NewTransfer(args, input)
	require.Nil(tf)
	require.ErrorContains(err, "not implemented")
}

// Tx

func (s *CrosschainTestSuite) TestTxHash() {
	require := s.Require()

	asset := &xc.ChainConfig{Chain: xc.BTC, Network: "testnet"}
	builder, _ := NewTxBuilder(asset)
	from := xc.Address("mpjwFvP88ZwAt3wEHY6irKkGhxcsv22BP6")
	to := xc.Address("tb1qtpqqpgadjr2q3f4wrgd6ndclqtfg7cz5evtvs0")
	amount := xc.NewBigIntFromUint64(1)
	args, err := xcbuilder.NewTransferArgs(from, to, amount)
	require.NoError(err)

	input := &tx_input.TxInput{
		UnspentOutputs: []tx_input.Output{{
			Value: xc.NewBigIntFromUint64(1000),
		}},
		GasPricePerByte: xc.NewBigIntFromUint64(1),
	}
	tf, err := builder.NewNativeTransfer(args, input)
	require.NoError(err)

	tx := tf.(*tx.Tx)
	require.Equal(xc.TxHash("0ebdd0e519cf4bf67ac4d924c07e3312483b09844c9f16f46c04f5fe1500c788"), tx.Hash())
}

func (s *CrosschainTestSuite) TestTxSighashes() {
	require := s.Require()
	tx := tx.Tx{
		Input: &tx_input.TxInput{},
	}
	sighashes, err := tx.Sighashes()
	require.NotNil(sighashes)
	require.NoError(err)
}

func (s *CrosschainTestSuite) TestTxAddSignature() {
	require := s.Require()
	asset := &xc.ChainConfig{Chain: xc.BTC, Network: "testnet"}
	builder, _ := NewTxBuilder(asset)
	from := xc.Address("mpjwFvP88ZwAt3wEHY6irKkGhxcsv22BP6")
	to := xc.Address("tb1qtpqqpgadjr2q3f4wrgd6ndclqtfg7cz5evtvs0")
	amount := xc.NewBigIntFromUint64(10)
	args, err := xcbuilder.NewTransferArgs(from, to, amount)
	require.NoError(err)

	input := &tx_input.TxInput{
		UnspentOutputs: []tx_input.Output{{
			Value: xc.NewBigIntFromUint64(1000),
		}},
	}
	err = input.SetPublicKey([]byte{})
	require.NoError(err)
	tf, err := builder.NewNativeTransfer(args, input)
	require.NoError(err)

	txObject := tf.(*tx.Tx)
	err = txObject.AddSignatures([]xc.TxSignature{
		[]byte{1, 2, 3, 4},
	}...)
	require.ErrorContains(err, "signature must be 64 or 65 length")
	sig := []byte{}
	for i := 0; i < 65; i++ {
		sig = append(sig, byte(i))
	}
	err = txObject.AddSignatures([]xc.TxSignature{
		sig,
	}...)
	require.NoError(err)

	// can't sign multiple times in a row
	err = txObject.AddSignatures([]xc.TxSignature{
		sig,
	}...)
	require.ErrorContains(err, "already signed")

	// must have a signature for each input needed
	tf, _ = builder.NewNativeTransfer(args, input)
	err = tf.(*tx.Tx).AddSignatures([]xc.TxSignature{
		sig, sig,
	}...)
	require.ErrorContains(err, "expected 1 signatures, got 2 signatures")

	// 2 inputs = 2 sigs
	args.SetAmount(xc.NewBigIntFromUint64(15000))
	input = &tx_input.TxInput{
		UnspentOutputs: []tx_input.Output{{
			Value: xc.NewBigIntFromUint64(10000),
		},
			{
				Value: xc.NewBigIntFromUint64(10000),
			},
		},
	}
	tf, _ = builder.NewNativeTransfer(args, input)
	require.Len(tf.(*tx.Tx).Input.UnspentOutputs, 2)
	err = tf.(*tx.Tx).AddSignatures([]xc.TxSignature{
		sig, sig,
	}...)
	require.NoError(err)
}
