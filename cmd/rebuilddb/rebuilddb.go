package main

import (
	"fmt"
	"os"
	"os/signal"
	"runtime/pprof"
	"sync"

	"github.com/Legenddigital/lddld/rpcclient"
	"github.com/Legenddigital/lddldata/db/lddlsqlite"
	"github.com/Legenddigital/lddldata/rpcutils"
	"github.com/Legenddigital/lddldata/stakedb"
	"github.com/Legenddigital/slog"
)

var (
	backendLog      *slog.Backend
	rpcclientLogger slog.Logger
	sqliteLogger    slog.Logger
	stakedbLogger   slog.Logger
)

func init() {
	err := InitLogger()
	if err != nil {
		fmt.Printf("Unable to start logger: %v", err)
		os.Exit(1)
	}
	backendLog = slog.NewBackend(log.Writer())
	rpcclientLogger = backendLog.Logger("RPC")
	rpcclient.UseLogger(rpcclientLogger)
	sqliteLogger = backendLog.Logger("DSQL")
	lddlsqlite.UseLogger(rpcclientLogger)
	stakedbLogger = backendLog.Logger("SKDB")
	stakedb.UseLogger(stakedbLogger)
}

func mainCore() int {
	// Parse the configuration file, and setup logger.
	cfg, err := loadConfig()
	if err != nil {
		fmt.Printf("Failed to load lddldata config: %s\n", err.Error())
		return 1
	}

	if cfg.CPUProfile != "" {
		f, err := os.Create(cfg.CPUProfile)
		if err != nil {
			log.Fatal(err)
			return -1
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	// Connect to node RPC server
	client, _, err := rpcutils.ConnectNodeRPC(cfg.LddldServ, cfg.LddldUser,
		cfg.LddldPass, cfg.LddldCert, cfg.DisableDaemonTLS)
	if err != nil {
		log.Fatalf("Unable to connect to RPC server: %v", err)
		return 1
	}

	infoResult, err := client.GetInfo()
	if err != nil {
		log.Errorf("GetInfo failed: %v", err)
		return 1
	}
	log.Info("Node connection count: ", infoResult.Connections)

	_, _, err = client.GetBestBlock()
	if err != nil {
		log.Error("GetBestBlock failed: ", err)
		return 2
	}

	// Sqlite output
	dbInfo := lddlsqlite.DBInfo{FileName: cfg.DBFileName}
	//sqliteDB, err := lddlsqlite.InitDB(&dbInfo)
	sqliteDB, cleanupDB, err := lddlsqlite.InitWiredDB(&dbInfo, nil, client,
		activeChain, "rebuild_data")
	defer cleanupDB()
	if err != nil {
		log.Errorf("Unable to initialize SQLite database: %v", err)
	}
	log.Infof("SQLite DB successfully opened: %s", cfg.DBFileName)
	defer sqliteDB.Close()

	// Ctrl-C to shut down.
	// Nothing should be sent the quit channel.  It should only be closed.
	quit := make(chan struct{})
	// Only accept a single CTRL+C
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	// Start waiting for the interrupt signal
	go func() {
		<-c
		signal.Stop(c)
		// Close the channel so multiple goroutines can get the message
		log.Infof("CTRL+C hit.  Closing goroutines. Please wait.")
		close(quit)
	}()

	// Resync db
	var waitSync sync.WaitGroup
	waitSync.Add(1)
	//go sqliteDB.SyncDB(&waitSync, quit)
	var height int64
	height, err = sqliteDB.SyncDB(&waitSync, quit, nil, 0)
	if err != nil {
		log.Error(err)
	}

	waitSync.Wait()

	log.Printf("Done at height %d!", height)

	return 0
}

func main() {
	os.Exit(mainCore())
}
