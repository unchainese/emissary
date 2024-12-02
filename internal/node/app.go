package node

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/unchainese/unchain/internal/global"
	"log"
	"log/slog"
	"net/http"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

type App struct {
	cfg           *global.Config
	mu            sync.Mutex
	allowedUsers  map[string]int64
	trafficUserKB sync.Map
	reqCount      atomic.Int64
	svr           *http.Server
	exitSignal    chan os.Signal
}

func (app *App) httpSvr() {
	mux := http.NewServeMux()
	mux.HandleFunc("/wsv/{uid}", app.WsVLESS)
	mux.HandleFunc("/sub/{uid}", app.Sub)
	mux.HandleFunc("/ws-vless", app.WsVLESS)
	mux.HandleFunc("/", app.Ping)
	server := &http.Server{
		Addr:    app.cfg.ListenAddr,
		Handler: mux,
	}
	app.svr = server

}

func NewApp(c *global.Config, sig chan os.Signal) *App {
	app := &App{
		cfg:           c,
		mu:            sync.Mutex{},
		allowedUsers:  make(map[string]int64),
		trafficUserKB: sync.Map{},
		reqCount:      atomic.Int64{},
		exitSignal:    sig,
		svr:           nil,
	}
	for _, userID := range c.UserIDS() {
		app.allowedUsers[userID] = 1
	}
	app.httpSvr()
	go app.LoopPush()
	return app
}

func (app *App) Run() {
	log.Println("server starting on http://", app.cfg.ListenAddr)
	if err := app.svr.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("Could not listen on %s: %v\n", app.cfg.ListenAddr, err)
	}
}

func (app *App) PrintVLESSConnectionURLS() {
	listenPort := app.cfg.ListenPort()

	fmt.Printf("\n\n\nvist to get VLESS connection info: http://127.0.0.1:%d/sub/<YOUR_CONFIGED_UUID> \n", listenPort)
	fmt.Printf("vist to get VLESS connection info: http://<HOST>:%d/sub/<YOUR_UUID>\n", listenPort)

	for userID, _ := range app.allowedUsers {
		fmt.Println("\n------------- USER UUID:  ", userID, " -------------")
		urls := app.vlessUrls(userID)
		for _, url := range urls {
			fmt.Println(url)
		}
	}
	fmt.Println("\n\n\n")
}

func (app *App) Shutdown(ctx context.Context) {
	log.Println("Shutting down the server...")
	if err := app.svr.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	log.Println("Server exiting")
}

func (app *App) LoopPush() {
	url := app.cfg.RegisterUrl
	if url == "" {
		log.Println("Register url is empty, skip register, runs in standalone mode")
		return
	}
	app.pushNode() //initial register
	tk := time.NewTicker(app.cfg.PushInterval())
	defer tk.Stop()
	for {
		select {
		case sig := <-app.exitSignal:
			app.exitSignal <- sig
			app.pushNode() //last push
			return
		case <-tk.C:
			app.pushNode()
		}
	}
}

func (app *App) reqInc() {
	app.reqCount.Add(1)
}

func (app *App) trafficInc(uid string, byteN int64) {
	kb := byteN/1024 + 1 //floor
	value, ok := app.trafficUserKB.Load(uid)
	if !ok {
		app.trafficUserKB.Store(uid, kb)
		return
	}
	app.trafficUserKB.Store(uid, value.(int64)+kb)
}

func (app *App) stat() *AppStat {
	data := make(map[string]int64)
	app.trafficUserKB.Range(func(key, value interface{}) bool {
		data[key.(string)] = value.(int64)
		return true
	})
	app.trafficUserKB.Clear()

	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
		slog.Error(err.Error())
	}
	res := &AppStat{
		Traffic:     data,
		Hostname:    hostname,
		ReqCount:    app.reqCount.Load(),
		Goroutine:   int64(runtime.NumGoroutine()),
		VersionInfo: app.cfg.GitHash + " -> " + app.cfg.BuildTime,
	}
	res.SubAddresses = app.cfg.SubAddresses
	app.reqCount.Store(0)
	return res
}

type AppStat struct {
	Traffic      map[string]int64 `json:"traffic"`
	Hostname     string           `json:"hostname"`
	SubAddresses []string         `json:"sub_addresses"`
	ReqCount     int64            `json:"req_count"`
	Goroutine    int64            `json:"goroutine"`
	VersionInfo  string           `json:"version_info"`
}

func (app *App) pushNode() {
	url := app.cfg.RegisterUrl
	if url == "" {
		return
	}
	args := app.stat()
	body := bytes.NewBuffer(nil)
	err := json.NewEncoder(body).Encode(args)
	if err != nil {
		log.Println("Error encoding request:", err)
		return
	}

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		log.Println("Error registering:", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", app.cfg.RegisterToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println("Error registering:", err)
		return
	}
	defer resp.Body.Close()
	users := make(map[string]int64)
	err = json.NewDecoder(resp.Body).Decode(&users)
	if err != nil {
		log.Println("Error decoding response:", err)
		return
	}
	app.mu.Lock()
	app.allowedUsers = users
	app.mu.Unlock()
}

func (app *App) IsUserNotAllowed(uuid string) (isNotAllowed bool) {
	app.mu.Lock()
	defer app.mu.Unlock()
	_, ok := app.allowedUsers[uuid]
	if !ok {
		log.Println("Unauthorized user:", uuid)
		return true
	}
	return false
}
