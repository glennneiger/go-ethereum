// Copyright 2018 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p/simulations"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	"github.com/ethereum/go-ethereum/swarm/network"
	"github.com/ethereum/go-ethereum/swarm/network/simulation"
	cli "gopkg.in/urfave/cli.v1"
)

var serviceFuncs = adapters.Services{
	"discovery": newService,
}

const testMinProxBinSize = 2
const NoConnectionTimeout = 1

var discoveryEnabled = true

func init() {
	adapters.RegisterServices(serviceFuncs)
}

func create(ctx *cli.Context) error {
	log.PrintOrigins(true)
	log.Root().SetHandler(log.LvlFilterHandler(log.Lvl(verbosity), log.StreamHandler(os.Stdout, log.TerminalFormat(true))))
	if len(ctx.Args()) < 1 {
		return errors.New("argument should be the filename to verify or write-to")
	}
	filename = ctx.Args()[0]
	err := ResolvePath()
	if err != nil {
		return err
	}
	err = discoverySnapshot(10, adapters.NewSimAdapter(serviceFuncs))
	if err != nil {
		utils.Fatalf("Simulation failed: %s", err)
	}

	return err
}

func discoverySnapshot(nodes int, adapter adapters.NodeAdapter) error {
	// create network
	net := simulations.NewNetwork(adapter, &simulations.NetworkConfig{
		ID:             "0",
		DefaultService: "discovery",
	})
	defer net.Shutdown()
	ids, err := net.AddNodes(nodes)

	if err != nil {
		return err
	}

	events := make(chan *simulations.Event)
	sub := net.Events().Subscribe(events)
	select {
	case ev := <-events:
		//only catch node up events
		if ev.Type == simulations.EventTypeConn {
			utils.Fatalf("this shouldn't happen as connections weren't initiated yet")
		}
	case <-time.After(NoConnectionTimeout * time.Second):
	}

	sub.Unsubscribe()

	if len(net.Conns) > 0 {
		utils.Fatalf("no connections should exist after just adding nodes")
	}

	switch topology {
	case "star":
		net.SetPivotNode(ids[pivot])
		err := net.ConnectNodesStarPivot(nil)
		if err != nil {
			utils.Fatalf("had an error connecting the nodes in a star: %v", err)
		}
	case "ring":
		err := net.ConnectNodesRing(nil)
		if err != nil {
			utils.Fatalf("had an error connecting the nodes in a ring: %v", err)
		}
	case "chain":
		err := net.ConnectNodesChain(nil)
		if err != nil {
			utils.Fatalf("had an error connecting the nodes in a chain: %v", err)
		}
	case "full":
		err := net.ConnectNodesFull(nil)
		if err != nil {
			utils.Fatalf("had an error connecting full: %v", err)
		}
	}

	err = simulation.WaitNetworkHealth(net)
	if err != nil {
		return err
	}

	var snap *simulations.Snapshot
	if len(services) > 0 {
		var addServices []string
		var removeServices []string
		for _, osvc := range strings.Split(services, ",") {
			if strings.Index(osvc, "+") == 0 {
				addServices = append(addServices, osvc[1:])
			} else if strings.Index(osvc, "-") == 0 {
				removeServices = append(removeServices, osvc[1:])
			} else {
				panic("stick to the rules, you know what they are")
			}
		}
		snap, err = net.SnapshotWithServices(addServices, removeServices)
	} else {
		snap, err = net.Snapshot()
	}

	if err != nil {
		return errors.New("no shapshot dude")
	}
	jsonsnapshot, err := json.Marshal(snap)
	if err != nil {
		return fmt.Errorf("corrupt json snapshot: %v", err)
	}
	err = ioutil.WriteFile(filename, jsonsnapshot, 0755)
	if err != nil {
		return err
	}

	return nil
}

func newService(ctx *adapters.ServiceContext) (node.Service, error) {
	addr := network.NewAddr(ctx.Config.Node())

	kp := network.NewKadParams()
	kp.MinProxBinSize = testMinProxBinSize

	kad := network.NewKademlia(addr.Over(), kp)
	hp := network.NewHiveParams()
	hp.KeepAliveInterval = time.Duration(200) * time.Millisecond
	hp.Discovery = discoveryEnabled

	log.Info(fmt.Sprintf("discovery for nodeID %s is %t", ctx.Config.ID.String(), hp.Discovery))

	config := &network.BzzConfig{
		OverlayAddr:  addr.Over(),
		UnderlayAddr: addr.Under(),
		HiveParams:   hp,
	}

	return network.NewBzz(config, kad, nil, nil, nil), nil
}
