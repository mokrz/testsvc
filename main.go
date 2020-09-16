package main

// this is just a dummy modification so we can test a commit

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/p2p/discovery"
)

type discoveryNotifee struct {
	node host.Host
}

func (n *discoveryNotifee) HandlePeerFound(pi peer.AddrInfo) {
	fmt.Printf("[+] Discovered peer %s\n", pi.ID.Pretty())
	err := n.node.Connect(context.Background(), pi)

	if err != nil {
		fmt.Printf("error connecting to peer %s: %s\n", pi.ID.Pretty(), err)
	}
}

func handleStream(stream network.Stream) {
	return
}

func configureDiscovery(ctx context.Context, node host.Host) error {
	disc, err := discovery.NewMdnsService(ctx, node, time.Hour, "testing")

	if err != nil {
		return err
	}

	n := discoveryNotifee{node: node}
	disc.RegisterNotifee(&n)

	return nil
}

type part struct {
	Hash string `json:"hash"`
	Path string `json:"Path"`
}

type dfile struct {
	subscription *pubsub.Subscription
	topic        *pubsub.Topic
	self         peer.ID
	parts        chan *part
}

func (df *dfile) readLoop() {
	for {
		var (
			msg *pubsub.Message
			p   part
			err error
		)

		if msg, err = df.subscription.Next(context.Background()); err != nil {
			close(df.parts)
			return
		}

		if err = json.NewDecoder(bytes.NewReader(msg.Data)).Decode(&p); err != nil {
			fmt.Printf("failed to decode message data: %s\n", err.Error())
			close(df.parts)
			return
		}

		df.parts <- &p
	}
}

func (df *dfile) writeLoop() {
	rand.Seed(time.Now().UnixNano())

	randString := func(n int) string {
		letterRunes := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
		b := make([]rune, n)

		for i := range b {
			b[i] = letterRunes[rand.Intn(len(letterRunes))]
		}

		return string(b)
	}

	for {
		r := randString(8)
		b, err := json.Marshal(&part{
			Hash: r,
			Path: "/tmp/test",
		})

		if err != nil {
			fmt.Printf("failed to marshal message data: %s\n", err.Error())
			return
		}

		if err = df.topic.Publish(context.Background(), b); err != nil {
			fmt.Printf("failed to publish: %s\n", err.Error())
			return
		}

		fmt.Printf("[+] Published: %s\n", string(b))
		time.Sleep(time.Duration(rand.Intn(7)) * time.Second)
	}
}

func main() {
	var (
		ctx                                      context.Context
		opts                                     []libp2p.Option
		node                                     host.Host
		ps                                       *pubsub.PubSub
		topic                                    *pubsub.Topic
		sub                                      *pubsub.Subscription
		nodeErr, psErr, discErr, joinErr, subErr error
	)
	ctx = context.Context(context.Background())
	opts = []libp2p.Option{
		libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"),
		libp2p.DisableRelay(),
		libp2p.Ping(false),
	}

	flag.Parse()

	if node, nodeErr = libp2p.New(ctx, opts...); nodeErr != nil {
		fmt.Printf("failed to create libp2p node: %s\n", nodeErr.Error())
		return
	}

	if ps, psErr = pubsub.NewGossipSub(ctx, node); psErr != nil {
		fmt.Printf("failed to create gossip sub service: %s\n", psErr.Error())
		return
	}

	if discErr = configureDiscovery(ctx, node); discErr != nil {
		fmt.Printf("failed to configure discovery: %s\n", discErr.Error())
		return
	}

	if topic, joinErr = ps.Join("testing"); joinErr != nil {
		fmt.Printf("failed to join testing topic: %s\n", joinErr.Error())
		return
	}

	if sub, subErr = topic.Subscribe(); subErr != nil {
		fmt.Printf("failed to subscibe to testing topic: %s\n", subErr.Error())
		return
	}

	fmt.Printf("[+] Subscribed to topic\n")

	df := &dfile{
		subscription: sub,
		self:         node.ID(),
		topic:        topic,
		parts:        make(chan *part, 128),
	}

	go df.writeLoop()

	for {
		var (
			msg *pubsub.Message
			p   part
			err error
		)

		if msg, err = df.subscription.Next(context.Background()); err != nil {
			close(df.parts)
			break
		}

		if err = json.NewDecoder(bytes.NewReader(msg.Data)).Decode(&p); err != nil {
			fmt.Printf("failed to decode message data: %s\n", err.Error())
			close(df.parts)
			break
		}

		fmt.Printf("Got: %s from %s\n", string(msg.Data), msg.GetFrom())
	}

	return
}
