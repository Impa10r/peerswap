package swap

import (
	"fmt"

	"github.com/btcsuite/btcd/btcec"
	"github.com/sputn1ck/glightning/glightning"
	"github.com/sputn1ck/peerswap/lightning"
)

type Messenger interface {
	SendMessage(peerId string, message []byte, messageType int) error
	AddMessageHandler(func(peerId string, msgType string, payload string) error)
}

type PeerMessage interface {
	MessageType() MessageType
}

type Policy interface {
	IsPeerAllowed(peer string) bool
	GetReserveOnchainMsat() uint64
}
type LightningClient interface {
	DecodePayreq(payreq string) (*lightning.Invoice, error)
	PayInvoice(payreq string) (preImage string, err error)
	CheckChannel(channelId string, amount uint64) error
	GetPayreq(msatAmount uint64, preimage string, label string) (string, error)
	AddPaymentCallback(f func(*glightning.Payment))
	RebalancePayment(payreq string, channel string) (preimage string, err error)
}

type Onchain interface {
	CreateOpeningTransaction(swapParams *OpeningParams) (unpreparedTxHex string, txId string, fee uint64, cltv int64, vout uint32, err error)
	BroadcastOpeningTx(unpreparedTxHex string) (txId, txHex string, error error)
	CreatePreimageSpendingTransaction(swapParams *OpeningParams, claimParams *ClaimParams, openingTxId string) (txId, txHex string, error error)
	CreateCltvSpendingTransaction(swapParams *OpeningParams, claimParams *ClaimParams, openingTxHex string, vout uint32) (txId, txHex string, error error)
	AddWaitForConfirmationTx(swapId, txId string) (err error)
	AddWaitForCltvTx(swapId, txId string, blockheight uint64) (err error)
	AddConfirmationCallback(func(swapId string) error)
	AddCltvCallback(func(swapId string) error)
	ValidateTx(swapParams *OpeningParams, cltv int64, openingTxId string) (bool, error)
	TakerCreateCoopSigHash(swapParams *OpeningParams, claimParams *ClaimParams, openingTxId, refundAddress string) (sigHash string, error error)
	CreateCooperativeSpendingTransaction(swapParams *OpeningParams, claimParams *ClaimParams, refundAddress, openingTxHex string, vout uint32, takerSignatureHex string) (txId, txHex string, error error)
	CreateRefundAddress() (string, error)
}

type OpeningParams struct {
	TakerPubkeyHash  string
	MakerPubkeyHash  string
	ClaimPaymentHash string
	Amount           uint64
}

type ClaimParams struct {
	Cltv     int64
	Preimage string
	Signer   Signer
}

type Signer interface {
	Sign(hash []byte) (*btcec.Signature, error)
}

type SwapServices struct {
	swapStore      Store
	lightning      LightningClient
	messenger      Messenger
	policy         Policy
	bitcoinOnchain Onchain
	bitcoinEnabled bool
	liquidOnchain  Onchain
	liquidEnabled  bool
}

func NewSwapServices(swapStore Store, lightning LightningClient, messenger Messenger, policy Policy, bitcoinEnabled bool, bitcoinOnchain Onchain, liquidEnabled bool, liquidOnchain Onchain) *SwapServices {
	return &SwapServices{swapStore: swapStore, lightning: lightning, messenger: messenger, policy: policy, bitcoinOnchain: bitcoinOnchain, bitcoinEnabled: bitcoinEnabled, liquidEnabled: liquidEnabled, liquidOnchain: liquidOnchain}
}

func (s *SwapServices) getOnchainAsset(asset string) (Onchain, error) {
	if asset == "" {
		return nil, fmt.Errorf("missing asset")
	}
	if asset == "btc" {
		return s.bitcoinOnchain, nil
	}
	if asset == "l-btc" {
		return s.liquidOnchain, nil
	}
	return nil, WrongAssetError(asset)
}
