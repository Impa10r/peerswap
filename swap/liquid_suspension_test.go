package swap

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"

	"github.com/elementsproject/peerswap/messages"
	"go.etcd.io/bbolt"
)

const suspensionBitcoinNetwork = "mainnet"

type suspensionMessenger struct {
	peer    string
	kind    int
	payload []byte
}

func (m *suspensionMessenger) AddMessageHandler(func(string, string, []byte) error) {}
func (m *suspensionMessenger) SendMessage(peer string, payload []byte, kind int) error {
	m.peer, m.payload, m.kind = peer, payload, kind
	return nil
}

func TestNewLiquidSwapsDisabledBeforeSideEffects(t *testing.T) {
	// No dependencies: any wallet, invoice, policy or chain call would panic.
	s := &SwapService{}
	_, err := s.SwapIn("peer", l_btc_chain, "channel", "local", 100000, 0)
	if err := err; !errors.Is(err, ErrNewLiquidSwapsDisabled) {
		t.Fatalf("got %v, want %v", err, ErrNewLiquidSwapsDisabled)
	}
	_, err = s.SwapOut("peer", l_btc_chain, "channel", "local", 100000, 0)
	if err := err; !errors.Is(err, ErrNewLiquidSwapsDisabled) {
		t.Fatalf("got %v, want %v", err, ErrNewLiquidSwapsDisabled)
	}
}

func TestRemoteLiquidRequestsRejected(t *testing.T) {
	for _, network := range []string{"", suspensionBitcoinNetwork} {
		for _, direction := range []string{"in", "out"} {
			t.Run(network+"/"+direction, func(t *testing.T) {
				id := NewSwapId()
				m := &suspensionMessenger{}
				s := &SwapService{
					swapServices: &SwapServices{messenger: m},
					activeSwaps:  map[string]*SwapStateMachine{},
					lastMsgLog:   map[string]string{},
				}
				var msg PeerMessage = &SwapInRequestMessage{SwapId: id, Asset: "liquid-asset", Network: network}
				if direction == "out" {
					msg = &SwapOutRequestMessage{SwapId: id, Asset: "liquid-asset", Network: network}
				}
				payload, kind, err := MarshalPeerswapMessage(msg)
				assertNoError(t, err)
				err = s.OnMessageReceived("peer", messages.MessageTypeToHexString(messages.MessageType(kind)), payload)
				if err := err; !errors.Is(err, ErrNewLiquidSwapsDisabled) {
					t.Fatalf("got %v, want %v", err, ErrNewLiquidSwapsDisabled)
				}
				assertEqual(t, 0, len(s.activeSwaps))
				assertEqual(t, "peer", m.peer)
				assertEqual(t, int(messages.MESSAGETYPE_CANCELED), m.kind)
				var cancel CancelMessage
				assertNoError(t, json.Unmarshal(m.payload, &cancel))
				assertEqual(t, id.String(), cancel.SwapId.String())
				assertEqual(t, ErrNewLiquidSwapsDisabled.Error(), cancel.Message)
			})
		}
	}
}

func TestLiquidRetransmissionDoesNotCancelExistingSwap(t *testing.T) {
	id := NewSwapId()
	m := &suspensionMessenger{}
	existing := &SwapStateMachine{Data: &SwapData{PeerNodeId: "peer"}, Current: State_WaitCsv}
	s := &SwapService{
		swapServices: &SwapServices{messenger: m},
		activeSwaps:  map[string]*SwapStateMachine{id.String(): existing},
	}
	if err := s.OnSwapInRequestReceived(id, "peer", &SwapInRequestMessage{}); !errors.Is(err, AlreadyExistsError) {
		t.Fatalf("got %v, want %v", err, AlreadyExistsError)
	}
	if err := s.OnSwapOutRequestReceived(id, "peer", &SwapOutRequestMessage{}); !errors.Is(err, AlreadyExistsError) {
		t.Fatalf("got %v, want %v", err, AlreadyExistsError)
	}
	if err := s.OnSwapInRequestReceived(id, "other", &SwapInRequestMessage{}); err == nil {
		t.Fatal("expected an error")
	}
	assertEqual(t, 0, len(m.payload))
	assertEqual(t, State_WaitCsv, existing.Current)
}

// recoveryChain records the watch installed when a persisted swap is restored.
type recoveryChain struct {
	dummyChain
	watchedCSV uint32
	watchedID  string
}

func (c *recoveryChain) AddWaitForCsvTx(id, _ string, _, _, csv uint32, _ []byte) {
	c.watchedID, c.watchedCSV = id, csv
}

func TestSuspensionPreservesPersistedLiquidRecovery(t *testing.T) {
	for _, direction := range []SwapType{SWAPTYPE_IN, SWAPTYPE_OUT} {
		for _, claim := range []string{"csv", "coop"} {
			t.Run(direction.String()+"/"+claim, func(t *testing.T) {
				services := getSwapServices(t, nil)
				services.bitcoinEnabled = false
				db, err := bbolt.Open(filepath.Join(t.TempDir(), "swaps.db"), 0o600, nil)
				assertNoError(t, err)
				t.Cleanup(func() { assertNoError(t, db.Close()) })
				store, err := NewBboltStore(db)
				assertNoError(t, err)
				services.swapStore = store
				chain := &recoveryChain{}
				services.liquidTxWatcher = chain
				services.liquidWallet = chain
				services.liquidValidator = chain
				id := NewSwapId()
				data := NewSwapData(id, "local", "peer")
				data.OpeningTxBroadcasted = &OpeningTxBroadcastedMessage{SwapId: id, TxId: getRandom32ByteHexString()}
				data.OpeningTxHex = "opening"
				role := SWAPROLE_SENDER
				if direction == SWAPTYPE_IN {
					data.SwapInRequest = &SwapInRequestMessage{
						SwapId:          id,
						Asset:           l_btc_chain,
						ProtocolVersion: PEERSWAP_PROTOCOL_VERSION,
						Scid:            "channel",
						Amount:          100000,
					}
				} else {
					role = SWAPROLE_RECEIVER
					data.SwapOutRequest = &SwapOutRequestMessage{
						SwapId:          id,
						Asset:           l_btc_chain,
						ProtocolVersion: PEERSWAP_PROTOCOL_VERSION,
						Scid:            "channel",
						Amount:          100000,
					}
				}
				data.SwapInAgreement = &SwapInAgreementMessage{SwapId: id, ProtocolVersion: PEERSWAP_PROTOCOL_VERSION}
				data.SwapOutAgreement = &SwapOutAgreementMessage{SwapId: id, ProtocolVersion: PEERSWAP_PROTOCOL_VERSION}

				data.Role = role
				saved := &SwapStateMachine{SwapId: id, Data: data, Type: direction, Role: role, Current: State_WaitCsv}
				assertNoError(t, store.UpdateData(saved))
				service := NewSwapService(services)
				assertNoError(t, service.Start())
				assertEqual(t, true, service.LiquidEnabled)
				assertEqual(t, false, service.BitcoinEnabled)
				assertNoError(t, service.RecoverSwaps())
				restored, err := service.GetActiveSwap(id.String())
				assertNoError(t, err)
				assertEqual(t, State_WaitCsv, restored.Current)
				assertEqual(t, id.String(), chain.watchedID)
				assertEqual(t, uint32(10080), chain.watchedCSV)
				assertEqual(t, 0, len(restored.Data.ClaimTxId))
				// With no CSV event the restored swap remains waiting. No wall-clock
				// deadline is substituted for Liquid block progress.
				if claim == "csv" {
					assertNoError(t, service.OnCsvPassed(id.String()))
					assertEqual(t, State_ClaimedCsv, restored.Current)
				} else {
					assertNoError(
						t,
						service.OnCoopCloseReceived(
							id,
							&CoopCloseMessage{SwapId: id, Privkey: getRandom32ByteHexString()},
						),
					)
					assertEqual(t, State_ClaimedCoop, restored.Current)
				}
				if len(restored.Data.ClaimTxId) == 0 {
					t.Fatal("expected a claim transaction")
				}
			})
		}
	}
}
