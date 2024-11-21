package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	peerstore "github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"
	"github.com/multiformats/go-multiaddr"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	sourceMultiAddr, err := multiaddr.NewMultiaddr("/ip4/0.0.0.0/tcp/0")
	if err != nil {
		panic(err)
	}

	r := rand.Reader
	preKey, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
	if err != nil {
		panic(err)
	}

	node, err := libp2p.New(
		libp2p.ListenAddrs(sourceMultiAddr),
		//libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"),
		//libp2p.Ping(false),
		libp2p.Identity(preKey),
	)
	if err != nil {
		panic(err)
	}

	pingService := &ping.PingService{
		Host: node,
	}

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
		node.SetStreamHandler(ping.ID, pingService.PingHandler)
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
		peerChan := initMDNS(node, "test")
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
