package node

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/unchainese/unchain/internal/schema"
	"io"
	"log"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

const buffSize = 8 << 10

var upGrader = websocket.Upgrader{
	ReadBufferSize:  buffSize,
	WriteBufferSize: buffSize,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all connections by default
		return true
	},
}

func startDstConnection(vd *schema.ProtoVLESS, timeout time.Duration) (net.Conn, []byte, error) {
	conn, err := net.DialTimeout(vd.DstProtocol, vd.HostPort(), timeout)
	if err != nil {
		return nil, nil, fmt.Errorf("connecting to destination: %w", err)
	}
	return conn, []byte{vd.Version, 0x00}, nil
}

func (app *App) WsVLESS(w http.ResponseWriter, r *http.Request) {
	app.reqInc()
	uid := r.PathValue("uid")
	//check can upgrade websocket
	if r.Header.Get("Upgrade") != "websocket" {
		//json response hello world
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		data := map[string]string{"msg": "pong", "uid": uid}
		json.NewEncoder(w).Encode(data)
		return
	}

	ctx := r.Context()
	earlyDataHeader := r.Header.Get("sec-websocket-protocol")
	earlyData, err := base64.RawURLEncoding.DecodeString(earlyDataHeader)
	if err != nil {
		log.Println("Error decoding early data:", err)
	}

	ws, err := upGrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Error upgrading to websocket:", err)
		return
	}
	defer ws.Close()

	if len(earlyData) == 0 {
		mt, p, err := ws.ReadMessage()
		if err != nil {
			log.Println("Error reading message:", err)
			return
		}
		if mt == websocket.BinaryMessage {
			earlyData = p
		}
	}

	vData, err := schema.VlessParse(earlyData)
	if err != nil {
		log.Println("Error parsing vless data:", err)
		return
	}
	if app.IsUserNotAllowed(vData.UUID()) {
		return
	}

	sessionTrafficByteN := int64(len(earlyData))

	if vData.DstProtocol == "udp" {
		sessionTrafficByteN += vlessUDP(ctx, vData, ws)
	} else if vData.DstProtocol == "tcp" {
		sessionTrafficByteN += vlessTCP(ctx, vData, ws)
	}
	app.trafficInc(vData.UUID(), sessionTrafficByteN)
}

func vlessTCP(_ context.Context, sv *schema.ProtoVLESS, ws *websocket.Conn) int64 {
	logger := sv.Logger()
	conn, headerVLESS, err := startDstConnection(sv, time.Millisecond*1000)
	if err != nil {
		logger.Error("Error starting session:", "err", err)
		return 0
	}
	defer conn.Close()
	logger.Info("Session started tcp")

	//write early data
	_, err = conn.Write(sv.DataTcp())
	if err != nil {
		logger.Error("Error writing early data to TCP connection:", "err", err)
		return 0
	}
	var trafficMeter atomic.Int64
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for {
			mt, message, err := ws.ReadMessage()
			trafficMeter.Add(int64(len(message)))
			if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				return
			}
			if err != nil {
				logger.Error("Error reading message:", "err", err)
				return
			}
			if mt != websocket.BinaryMessage {
				continue
			}
			_, err = conn.Write(message)
			if err != nil {
				logger.Error("Error writing to TCP connection:", "err", err)
				return
			}
		}
	}()

	go func() {
		defer wg.Done()
		hasNotSentHeader := true
		buf := make([]byte, buffSize)
		for {
			n, err := conn.Read(buf)
			trafficMeter.Add(int64(n))
			if errors.Is(err, io.EOF) {
				return
			}
			if err != nil {
				logger.Error("Error reading from TCP connection:", "err", err)
				return
			}
			data := buf[:n]
			// send header data only for the first time
			if hasNotSentHeader {
				hasNotSentHeader = false
				data = append(headerVLESS, data...)
			}
			err = ws.WriteMessage(websocket.BinaryMessage, data)
			if err != nil {
				logger.Error("Error writing to websocket:", "err", err)
				return
			}
		}
	}()
	wg.Wait()
	return trafficMeter.Load()
}

func vlessUDP(_ context.Context, sv *schema.ProtoVLESS, ws *websocket.Conn) (trafficMeter int64) {
	logger := sv.Logger()
	conn, headerVLESS, err := startDstConnection(sv, time.Millisecond*1000)
	if err != nil {
		logger.Error("Error starting session:", "err", err)
		return
	}
	defer conn.Close()
	trafficMeter += int64(len(sv.DataUdp()))
	//write early data
	_, err = conn.Write(sv.DataUdp())
	if err != nil {
		logger.Error("Error writing early data to TCP connection:", "err", err)
		return
	}

	buf := make([]byte, buffSize)
	n, err := conn.Read(buf)
	if err != nil {
		logger.Error("Error reading from TCP connection:", "err", err)
		return
	}
	udpDataLen1 := (n >> 8) & 0xff
	udpDataLen2 := n & 0xff
	headerVLESS = append(headerVLESS, byte(udpDataLen1), byte(udpDataLen2))
	headerVLESS = append(headerVLESS, buf[:n]...)

	trafficMeter += int64(len(headerVLESS))
	err = ws.WriteMessage(websocket.BinaryMessage, headerVLESS)
	if err != nil {
		logger.Error("Error writing to websocket:", "err", err)
		return
	}
	return trafficMeter
}
