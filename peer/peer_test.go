package peer

import (
	"context"
	"fmt"
	"github.com/libp2p/go-libp2p"
	peerstore "github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"
	"github.com/multiformats/go-multiaddr"
	"os"
	"os/signal"
	"syscall"
	"testing"
)

func TestPeer(t *testing.T) {
	node, err := libp2p.New(
		libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"),
		libp2p.Ping(false),
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
		t.Fatal(err)
	}
	fmt.Println("libp2p node address: ", addrs[0])

	if len(os.Args) > 1 {
		addr, err := multiaddr.NewMultiaddr(os.Args[1])
		if err != nil {
			t.Fatal(err)
		}
		peer, err := peerstore.AddrInfoFromP2pAddr(addr)
		if err != nil {
			t.Fatal(err)
		}
		if err = node.Connect(context.Background(), *peer); err != nil {
			t.Fatal(err)
		}
		fmt.Println("sending 5 ping messages to ", addr)
		ch := pingService.Ping(context.Background(), peer.ID)
		for i := 0; i < 5; i++ {
			res := <-ch
			fmt.Println("pinged ", addr, " in ", res.RTT)
		}
	} else {
		// mdns 节点发现
		notifee := NewDiscoveryNotifee()
		ser := mdns.NewMdnsService(node, "test1", notifee)
		go func() {
			er := ser.Start()
			if er != nil {
				panic(er)
			}
		}()

		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
		<-ch
		fmt.Println("Received signal, shutting down...")
	}
	if err = node.Close(); err != nil {
		panic(err)
	}

}

type discoveryNotifee struct {
	peerChan chan peerstore.AddrInfo
}

func NewDiscoveryNotifee() mdns.Notifee {
	return &discoveryNotifee{
		peerChan: make(chan peerstore.AddrInfo),
	}
}

func (d discoveryNotifee) HandlePeerFound(info peerstore.AddrInfo) {
	d.peerChan <- info
	fmt.Printf("Found peer, id: %s, addrs: %v\n", info.ID, info.Addrs)
}
