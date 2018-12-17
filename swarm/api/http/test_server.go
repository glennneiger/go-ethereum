// Copyright 2017 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ethereum/go-ethereum/swarm/testutil"

	"github.com/ethereum/go-ethereum/swarm/api"
	"github.com/ethereum/go-ethereum/swarm/storage"
	"github.com/ethereum/go-ethereum/swarm/storage/feed"
)

type TestServer interface {
	ServeHTTP(http.ResponseWriter, *http.Request)
}

func NewTestSwarmServer(tx *testing.T, serverFunc func(*api.API) TestServer, resolver api.Resolver) *TestSwarmServer {
	// hacked this function for the testing framework demo. Disregard changes here.
	t := testutil.BeginTest(tx, false)
	dir := t.Services.NewTempDir()

	storeparams := storage.NewDefaultLocalStoreParams()
	storeparams.DbCapacity = 5000000
	storeparams.CacheCapacity = 5000
	storeparams.Init(dir)
	localStore, err := storage.NewLocalStore(storeparams, nil)
	t.Ok(err)

	fileStore := storage.NewFileStore(localStore, storage.NewFileStoreParams())

	rh := feed.NewTestHandler(t, "")

	a := api.NewAPI(fileStore, resolver, rh.Handler, nil)
	srv := httptest.NewServer(serverFunc(a))
	tss := &TestSwarmServer{
		Server:    srv,
		FileStore: fileStore,
		dir:       dir,
		Hasher:    storage.MakeHashFunc(storage.DefaultHash)(),
		cleanup: func() {
			srv.Close()
			rh.Close()
			t.FinishTest()
		},
		CurrentTime: 42,
	}
	feed.TimestampProvider = tss
	return tss
}

type TestSwarmServer struct {
	*httptest.Server
	Hasher      storage.SwarmHash
	FileStore   *storage.FileStore
	dir         string
	cleanup     func()
	CurrentTime uint64
}

func (t *TestSwarmServer) Close() {
	t.cleanup()
}

func (t *TestSwarmServer) Now() feed.Timestamp {
	return feed.Timestamp{Time: t.CurrentTime}
}
