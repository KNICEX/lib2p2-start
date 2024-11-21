package main

import (
	"bufio"
	"context"
	"fmt"
	"github.com/KNICEX/libp2p-start/chat"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/routing"
	"github.com/libp2p/go-libp2p/p2p/discovery/util"
	"time"
)

func main() {
	ctx := context.Background()
	node, err := libp2p.New(
		libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"),
	)
	if err != nil {
		panic(err)
	}
	node.SetStreamHandler("/chat", chat.HandleStream)
	addrs := node.Addrs()
	fmt.Println("libp2p node address: ", addrs[0])

	bootstrapPeers := make([]peer.AddrInfo, len(dht.DefaultBootstrapPeers))
	for i, p := range dht.DefaultBootstrapPeers {
		pi, err := peer.AddrInfoFromP2pAddr(p)
		if err != nil {
			panic(err)
		}
		bootstrapPeers[i] = *pi
	}
	kademliaDHT, err := dht.New(ctx, node, dht.BootstrapPeers(bootstrapPeers...))
	if err != nil {
		panic(err)
	}

	if err = kademliaDHT.Bootstrap(ctx); err != nil {
		panic(err)
	}
	time.Sleep(time.Second)
	fmt.Println("Announcing ourselves...")
	routingDiscovery := routing.NewRoutingDiscovery(kademliaDHT)
	util.Advertise(ctx, routingDiscovery, "test")
	fmt.Println("Successfully announced!")

	fmt.Println("Searching for other peers...")
	peerChan, err := routingDiscovery.FindPeers(ctx, "test")
	if err != nil {
		panic(err)
	}

	for peerNode := range peerChan {
		if peerNode.ID == node.ID() {
			continue
		}
		fmt.Println("Found peer:", peerNode)

		fmt.Println("Connecting to:", peerNode)
		stream, err := node.NewStream(ctx, peerNode.ID, "/chat")
		if err != nil {
			fmt.Println("Connection failed:", err)
			continue
		} else {
			fmt.Println("Connected to:", peerNode)
			rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))
			go chat.Write(rw)
			go chat.Read(rw)
		}
	}
	fmt.Println("Done")
	select {}
}
