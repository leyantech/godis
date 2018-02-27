package godis

import (
	"encoding/json"
	"github.com/alicebob/miniredis"
	"github.com/go-redis/redis"
	"github.com/leyantech/godis/internal"
	"github.com/samuel/go-zookeeper/zk"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

const ZkAddr = "localhost:2181"
const ZkProxyDir = "/proxy"

func Test(t *testing.T) {
	zkConn, _, err := zk.Connect([]string{ZkAddr}, time.Second)
	if err != nil {
		t.Fatalf("failed to connect to zk, %v", err)
	}
	options := redis.Options{
		DB: 0,
	}

	addNode := func(name, addr, state string) {
		conn, _, err := zk.Connect([]string{ZkAddr}, time.Second)
		if err != nil {
			t.Fatalf("failed to connect to zk, %v", err)
		}
		defer conn.Close()

		exists, _, err := conn.Exists(ZkProxyDir)
		if err != nil {
			t.Fatalf("failed to connect to zk, %v", err)
		}
		if !exists {
			conn.Create(ZkProxyDir, []byte{}, 0, zk.WorldACL(zk.PermAll))
		}
		proxyInfo := internal.ProxyInfo{
			Addr:  addr,
			State: state,
		}
		bytes, err := json.Marshal(proxyInfo)
		if err != nil {
			t.Fatalf("failed to marshal proxy info, %v", err)
		}
		_, err = conn.Create(ZkProxyDir+"/"+name, bytes, 0, zk.WorldACL(zk.PermAll))
		if err != nil {
			t.Fatalf("failed to create zk node, %v", err)
		}
	}

	removeNode := func(name string) {
		conn, _, err := zk.Connect([]string{ZkAddr}, time.Second*1)
		defer conn.Close()
		if err != nil {
			t.Fatalf("failed to connect to zk, %v", err)
		}
		conn.Delete(ZkProxyDir+"/"+name, -1)
	}

	server1, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start redis server, %v", err)
	}
	defer server1.Close()

	server2, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start redis server, %v", err)
	}
	defer server2.Close()

	addNode("node1", server1.Addr(), "online")

	pool, err := NewRoundRobinPool(zkConn, ZkProxyDir, options)
	if err != nil {
		t.Fatalf("failed to create pool, %v", err)
	}
	defer pool.Close()

	pool.GetClient().Set("k1", "v1", 0)
	v1, _ := server1.Get("k1")
	assert.Equal(t, "v1", v1)

	addNode("node2", "127.0.0.1:12345", "offline")

	time.Sleep(time.Second)

	pool.GetClient().Set("k2", "v2", 0)
	v2, _ := server1.Get("k2")
	assert.Equal(t, "v2", v2)

	addNode("node3", server2.Addr(), "online")

	time.Sleep(time.Second)

	pool.GetClient().Set("k3", "v3", 0)
	v3, _ := server2.Get("k3")
	assert.Equal(t, "v3", v3)

	removeNode("node1")

	time.Sleep(time.Second)

	pool.GetClient().Set("k4", "v4", 0)
	v4, _ := server2.Get("k4")
	assert.Equal(t, "v4", v4)
}
