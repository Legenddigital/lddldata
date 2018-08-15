// Copyright (c) 2018, The Decred developers
// Copyright (c) 2018, The Legenddigital developers
// Copyright (c) 2017, Jonathan Chappelow
// See LICENSE for details.

package notification

import (
	"github.com/Legenddigital/lddld/chaincfg/chainhash"
	"github.com/Legenddigital/lddld/lddlutil"

	"github.com/Legenddigital/lddldata/api/insight"
	"github.com/Legenddigital/lddldata/blockdata"
	"github.com/Legenddigital/lddldata/db/lddlpg"
	"github.com/Legenddigital/lddldata/db/lddlsqlite"
	"github.com/Legenddigital/lddldata/explorer"
	"github.com/Legenddigital/lddldata/mempool"
	"github.com/Legenddigital/lddldata/stakedb"
	"github.com/Legenddigital/lddldata/txhelpers"
)

const (
	// blockConnChanBuffer is the size of the block connected channel buffers.
	blockConnChanBuffer = 64

	// newTxChanBuffer is the size of the new transaction channel buffer, for
	// ANY transactions are added into mempool.
	newTxChanBuffer = 48

	// expNewTxChanBuffer is the size of the new transaction buffer for explorer
	expNewTxChanBuffer = 70

	// relevantMempoolTxChanBuffer is the size of the new transaction channel
	// buffer, for relevant transactions that are added into mempool.
	//relevantMempoolTxChanBuffer = 2048
)

// NtfnChans collects the chain server notification channels
var NtfnChans struct {
	ConnectChan                       chan *chainhash.Hash
	ReorgChanBlockData                chan *blockdata.ReorgData
	ConnectChanWiredDB                chan *chainhash.Hash
	ReorgChanWiredDB                  chan *lddlsqlite.ReorgData
	ConnectChanStakeDB                chan *chainhash.Hash
	ReorgChanStakeDB                  chan *stakedb.ReorgData
	ConnectChanLddlpgDB               chan *chainhash.Hash
	ReorgChanLddlpgDB                 chan *lddlpg.ReorgData
	UpdateStatusNodeHeight            chan uint32
	UpdateStatusDBHeight              chan uint32
	SpendTxBlockChan, RecvTxBlockChan chan *txhelpers.BlockWatchedTx
	RelevantTxMempoolChan             chan *lddlutil.Tx
	NewTxChan                         chan *mempool.NewTx
	ExpNewTxChan                      chan *explorer.NewMempoolTx
	InsightNewTxChan                  chan *insight.NewTx
}

// MakeNtfnChans create notification channels based on config
func MakeNtfnChans(monitorMempool, postgresEnabled bool) {
	// If we're monitoring for blocks OR collecting block data, these channels
	// are necessary to handle new block notifications. Otherwise, leave them
	// as nil so that both a send (below) blocks and a receive (in
	// blockConnectedHandler) block. default case makes non-blocking below.
	// quit channel case manages blockConnectedHandlers.
	NtfnChans.ConnectChan = make(chan *chainhash.Hash, blockConnChanBuffer)

	// WiredDB channel for connecting new blocks
	NtfnChans.ConnectChanWiredDB = make(chan *chainhash.Hash, blockConnChanBuffer)

	// Stake DB channel for connecting new blocks - BLOCKING!
	NtfnChans.ConnectChanStakeDB = make(chan *chainhash.Hash)

	NtfnChans.ConnectChanLddlpgDB = make(chan *chainhash.Hash, blockConnChanBuffer)

	// Reorg data channels
	NtfnChans.ReorgChanBlockData = make(chan *blockdata.ReorgData)
	NtfnChans.ReorgChanWiredDB = make(chan *lddlsqlite.ReorgData)
	NtfnChans.ReorgChanStakeDB = make(chan *stakedb.ReorgData)
	NtfnChans.ReorgChanLddlpgDB = make(chan *lddlpg.ReorgData)

	// To update app status
	NtfnChans.UpdateStatusNodeHeight = make(chan uint32, blockConnChanBuffer)
	NtfnChans.UpdateStatusDBHeight = make(chan uint32, blockConnChanBuffer)

	// watchaddress
	// if len(cfg.WatchAddresses) > 0 {
	// // recv/SpendTxBlockChan come with connected blocks
	// 	NtfnChans.RecvTxBlockChan = make(chan *txhelpers.BlockWatchedTx, blockConnChanBuffer)
	// 	NtfnChans.SpendTxBlockChan = make(chan *txhelpers.BlockWatchedTx, blockConnChanBuffer)
	// 	NtfnChans.RelevantTxMempoolChan = make(chan *lddlutil.Tx, relevantMempoolTxChanBuffer)
	// }

	if monitorMempool {
		NtfnChans.NewTxChan = make(chan *mempool.NewTx, newTxChanBuffer)
	}

	// New mempool tx chan for explorer
	NtfnChans.ExpNewTxChan = make(chan *explorer.NewMempoolTx, expNewTxChanBuffer)

	if postgresEnabled {
		NtfnChans.InsightNewTxChan = make(chan *insight.NewTx, expNewTxChanBuffer)
	}
}

// CloseNtfnChans close all notification channels
func CloseNtfnChans() {
	if NtfnChans.ConnectChan != nil {
		close(NtfnChans.ConnectChan)
	}
	if NtfnChans.ConnectChanWiredDB != nil {
		close(NtfnChans.ConnectChanWiredDB)
	}
	if NtfnChans.ConnectChanStakeDB != nil {
		close(NtfnChans.ConnectChanStakeDB)
	}
	if NtfnChans.ConnectChanLddlpgDB != nil {
		close(NtfnChans.ConnectChanLddlpgDB)
	}

	if NtfnChans.ReorgChanBlockData != nil {
		close(NtfnChans.ReorgChanBlockData)
	}
	if NtfnChans.ReorgChanWiredDB != nil {
		close(NtfnChans.ReorgChanWiredDB)
	}
	if NtfnChans.ReorgChanStakeDB != nil {
		close(NtfnChans.ReorgChanStakeDB)
	}
	if NtfnChans.ReorgChanLddlpgDB != nil {
		close(NtfnChans.ReorgChanLddlpgDB)
	}

	if NtfnChans.UpdateStatusNodeHeight != nil {
		close(NtfnChans.UpdateStatusNodeHeight)
	}
	if NtfnChans.UpdateStatusDBHeight != nil {
		close(NtfnChans.UpdateStatusDBHeight)
	}

	if NtfnChans.NewTxChan != nil {
		close(NtfnChans.NewTxChan)
	}
	if NtfnChans.RelevantTxMempoolChan != nil {
		close(NtfnChans.RelevantTxMempoolChan)
	}

	if NtfnChans.SpendTxBlockChan != nil {
		close(NtfnChans.SpendTxBlockChan)
	}
	if NtfnChans.RecvTxBlockChan != nil {
		close(NtfnChans.RecvTxBlockChan)
	}

	if NtfnChans.ExpNewTxChan != nil {
		close(NtfnChans.ExpNewTxChan)
	}

	if NtfnChans.InsightNewTxChan != nil {
		close(NtfnChans.InsightNewTxChan)
	}
}
