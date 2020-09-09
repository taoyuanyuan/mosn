package network

import (
	"net"
	"strconv"
	"sync"

	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/buffer"
)

var EnvoyEnable bool
var EnvoyMap sync.Map

var EnvoyConnHandler types.ConnectionHandler

func SetConn(rawl net.Conn, id uint64) error {
	listener := findListen(rawl.LocalAddr(), EnvoyConnHandler)
	if listener == nil {
		return nil
	}
	ch := make(chan api.Connection, 1)
	listener.GetListenerCallbacks().OnAccept(rawl, listener.UseOriginalDst(), nil, ch, nil)
	rch := <-ch
	conn, _ := rch.(*connection)
	EnvoyMap.Store(id, conn)

	if conn.readBuffer == nil {
		conn.readBuffer = buffer.GetIoBuffer(DefaultBufferReadCapacity)
	}

	return nil
}

func EnvoyOnData(id uint64, buf []byte) error {
	c, _ := EnvoyMap.Load(id)
	conn := c.(*connection)

	conn.readBuffer.Write(buf)

	conn.onRead(int64(len(buf)))

	return nil
}

func findListen(addr net.Addr, handler types.ConnectionHandler) types.Listener {
	address := addr.(*net.TCPAddr)
	port := strconv.FormatInt(int64(address.Port), 10)
	ipv4, _ := net.ResolveTCPAddr("tcp", "0.0.0.0:"+port)
	ipv6, _ := net.ResolveTCPAddr("tcp", "[::]:"+port)

	var listener types.Listener
	if listener = handler.FindListenerByAddress(address); listener != nil {
		return listener
	} else if listener = handler.FindListenerByAddress(ipv4); listener != nil {
		return listener
	} else if listener = handler.FindListenerByAddress(ipv6); listener != nil {
		return listener
	}

	log.DefaultLogger.Errorf("[network] [transfer] Find Listener failed %v", address)
	return nil
}

