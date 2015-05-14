package ably_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/ably/ably-go/ably"
	"github.com/ably/ably-go/ably/proto"
	"github.com/ably/ably-go/ably/testutil"
)

func await(action func(chan<- error) error) error {
	wait := make(chan error, 1)
	err := action(wait)
	if err != nil {
		return err
	}
	select {
	case err := <-wait:
		if err != nil {
			return errors.New("nack: " + err.Error())
		}
	case <-time.After(timeout):
		return fmt.Errorf("waiting for Publish confirmation timed out after %v", timeout)
	}
	return nil
}

func publish(fn func(string, string, chan<- error) error, name, data string) func(chan<- error) error {
	return func(listen chan<- error) error {
		return fn(name, data, listen)
	}
}

func testMsg(ch <-chan *proto.Message, name, data string, t time.Duration, present bool) error {
	select {
	case msg := <-ch:
		if !present {
			return fmt.Errorf("received unexpected message name=%q data=%q", msg.Name, msg.Data)
		}
		if msg.Name != name {
			return fmt.Errorf("want msg.Name=%q; got %q", name, msg.Name)
		}
		if msg.Data != data {
			return fmt.Errorf("want msg.Data=%q; got %q", name, msg.Name)
		}
		return nil
	case <-time.After(t):
		if present {
			return fmt.Errorf("waiting for message name=%q data=%q timed out after %v", name, data, t)
		}
		return nil
	}
}

func TestRealtimeChannel_Publish(t *testing.T) {
	client, err := ably.NewRealtimeClient(testutil.Options(t, nil))
	if err != nil {
		t.Fatalf("NewRealtimeClient()=%v", err)
	}
	defer client.Close()

	channel := client.Channels.Get("test")
	if err := await(publish(channel.Publish, "hello", "world")); err != nil {
		t.Fatalf("Publish()=%v", err)
	}
}

func TestRealtimeChannel_Subscribe(t *testing.T) {
	opts := testutil.Options(t, nil)
	client1, err := ably.NewRealtimeClient(opts)
	if err != nil {
		t.Fatalf("client1: NewRealtimeClient()=%v", err)
	}
	defer client1.Close()
	opts.NoEcho = true
	client2, err := ably.NewRealtimeClient(opts)
	if err != nil {
		t.Fatalf("client2: NewRealtimeClient()=%v", err)
	}
	defer client2.Close()

	channel1 := client1.Channels.Get("test")
	ch1, err := channel1.Subscribe(make(chan *proto.Message, 2))
	if err != nil {
		t.Fatalf("client1: Subscribe()=%v", err)
	}

	channel2 := client2.Channels.Get("test")
	ch2, err := channel2.Subscribe(make(chan *proto.Message, 2))
	if err != nil {
		t.Fatalf("client2: Subscribe()=%v", err)
	}

	if err := await(publish(channel1.Publish, "hello", "client1")); err != nil {
		t.Fatalf("client1: Publish()=%v", err)
	}
	if err := await(publish(channel2.Publish, "hello", "client2")); err != nil {
		t.Fatalf("client2: Publish()=%v", err)
	}

	timeout := 15 * time.Second

	if err := testMsg(ch1, "hello", "client1", timeout, true); err != nil {
		t.Fatal(err)
	}
	if err := testMsg(ch1, "hello", "client2", timeout, true); err != nil {
		t.Fatal(err)
	}
	if err := testMsg(ch2, "hello", "client1", timeout, true); err != nil {
		t.Fatal(err)
	}
	if err := testMsg(ch2, "hello", "client2", timeout, false); err != nil {
		t.Fatal(err)
	}
}
