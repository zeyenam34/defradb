// Copyright 2023 Democratized Data Foundation
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package cli

import (
	"context"
	"encoding/json"
	"os"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/spf13/cobra"

	"github.com/sourcenetwork/defradb/client"
	"github.com/sourcenetwork/defradb/config"
	"github.com/sourcenetwork/defradb/datastore"
	"github.com/sourcenetwork/defradb/http"
)

type contextKey string

var (
	// txContextKey is the context key for the datastore.Txn
	//
	// This will only be set if a transaction id is specified.
	txContextKey = contextKey("tx")
	// dbContextKey is the context key for the client.DB
	dbContextKey = contextKey("db")
	// storeContextKey is the context key for the client.Store
	//
	// If a transaction exists, all operations will be executed
	// in the current transaction context.
	storeContextKey = contextKey("store")
	// colContextKey is the context key for the client.Collection
	//
	// If a transaction exists, all operations will be executed
	// in the current transaction context.
	colContextKey = contextKey("col")
)

// mustGetStoreContext returns the store for the current command context.
//
// If a store is not set in the current context this function panics.
func mustGetStoreContext(cmd *cobra.Command) client.Store {
	return cmd.Context().Value(storeContextKey).(client.Store)
}

// mustGetP2PContext returns the p2p implementation for the current command context.
//
// If a p2p implementation is not set in the current context this function panics.
func mustGetP2PContext(cmd *cobra.Command) client.P2P {
	return cmd.Context().Value(dbContextKey).(client.P2P)
}

// tryGetCollectionContext returns the collection for the current command context
// and a boolean indicating if the collection was set.
func tryGetCollectionContext(cmd *cobra.Command) (client.Collection, bool) {
	col, ok := cmd.Context().Value(colContextKey).(client.Collection)
	return col, ok
}

// setTransactionContext sets the transaction for the current command context.
func setTransactionContext(cmd *cobra.Command, cfg *config.Config, txId uint64) error {
	if txId == 0 {
		return nil
	}
	tx, err := http.NewTransaction(cfg.API.Address, txId)
	if err != nil {
		return err
	}
	ctx := context.WithValue(cmd.Context(), txContextKey, tx)
	cmd.SetContext(ctx)
	return nil
}

// setStoreContext sets the store for the current command context.
func setStoreContext(cmd *cobra.Command, cfg *config.Config) error {
	db, err := http.NewClient(cfg.API.Address)
	if err != nil {
		return err
	}
	ctx := context.WithValue(cmd.Context(), dbContextKey, db)
	if tx, ok := ctx.Value(txContextKey).(datastore.Txn); ok {
		ctx = context.WithValue(ctx, storeContextKey, db.WithTxn(tx))
	} else {
		ctx = context.WithValue(ctx, storeContextKey, db)
	}
	cmd.SetContext(ctx)
	return nil
}

// loadConfig loads the rootDir containing the configuration file,
// otherwise warn about it and load a default configuration.
func loadConfig(cfg *config.Config) error {
	if err := cfg.LoadRootDirFromFlagOrDefault(); err != nil {
		return err
	}
	return cfg.LoadWithRootdir(cfg.ConfigFileExists())
}

// createConfig creates the config directories and writes
// the current config to a file.
func createConfig(cfg *config.Config) error {
	if config.FolderExists(cfg.Rootdir) {
		return cfg.WriteConfigFile()
	}
	return cfg.CreateRootDirAndConfigFile()
}

// loadOrGeneratePrivateKey loads the private key from the given path
// or generates a new key and writes it to a file at the given path.
func loadOrGeneratePrivateKey(path string) (crypto.PrivKey, error) {
	key, err := loadPrivateKey(path)
	if err == nil {
		return key, nil
	}
	if os.IsNotExist(err) {
		return generatePrivateKey(path)
	}
	return nil, err
}

// generatePrivateKey generates a new private key and writes it
// to a file at the given path.
func generatePrivateKey(path string) (crypto.PrivKey, error) {
	key, _, err := crypto.GenerateKeyPair(crypto.Ed25519, 0)
	if err != nil {
		return nil, err
	}
	data, err := crypto.MarshalPrivateKey(key)
	if err != nil {
		return nil, err
	}
	return key, os.WriteFile(path, data, 0644)
}

// loadPrivateKey reads the private key from the file at the given path.
func loadPrivateKey(path string) (crypto.PrivKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return crypto.UnmarshalPrivateKey(data)
}

func writeJSON(cmd *cobra.Command, out any) error {
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
