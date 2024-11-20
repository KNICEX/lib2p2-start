package main

import (
	"context"
	"fmt"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	peerstore "github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"
	"github.com/multiformats/go-multiaddr"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	node, err := libp2p.New(
		libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"),
		//libp2p.Ping(false),
	)
	if err != nil {
		panic(err)
	}

	pingService := &ping.PingService{
		Host: node,
	}
	node.SetStreamHandler(ping.ID, pingService.PingHandler)

	peerInfo := peerstore.AddrInfo{
		ID:    node.ID(),
		Addrs: node.Addrs(),
	}
	// print connect addr
	addrs, err := peerstore.AddrInfoToP2pAddrs(&peerInfo)
	if err != nil {
		panic(err)
	}
	fmt.Println("libp2p node address: ", addrs[0])

	if len(os.Args) > 1 {
		addr, err := multiaddr.NewMultiaddr(os.Args[1])
		if err != nil {
			panic(err)
		}
		peer, err := peerstore.AddrInfoFromP2pAddr(addr)
		if err != nil {
			panic(err)
		}
		if err = node.Connect(context.Background(), *peer); err != nil {
			panic(err)
		}
		fmt.Println("sending 10 ping messages to ", addr)
		ch := pingService.Ping(context.Background(), peer.ID)
		for i := 0; i < 10; i++ {
			res := <-ch
			fmt.Println("pinged ", addr, " in ", res.RTT)
		}
	} else {
		// mdns 节点发现
		name := fmt.Sprintf("test-%d", rand.Intn(100))
		peerChan := initMDNS(node, name)
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
		for {
			select {
			case peer := <-peerChan:
				fmt.Println("Found peer:", peer, ", connecting")
			case <-ch:
				fmt.Println("Received signal, shutting down...")
				return
			}
		}
	}
	if err = node.Close(); err != nil {
		panic(err)
	}
}

type discoveryNotifee struct {
	PeerChan chan peerstore.AddrInfo
}

func initMDNS(host host.Host, name string) chan peerstore.AddrInfo {
	n := &discoveryNotifee{
		PeerChan: make(chan peerstore.AddrInfo, 10),
	}
	ser := mdns.NewMdnsService(host, name, n)
	if err := ser.Start(); err != nil {
		panic(err)
	}
	return n.PeerChan
}

func (d *discoveryNotifee) HandlePeerFound(info peerstore.AddrInfo) {
	d.PeerChan <- info
}
