package main

import (
	"context"
	"github.com/unchainese/unchain/internal/global"
	"github.com/unchainese/unchain/internal/node"
	"os"
	"os/signal"
	"time"
)

func main() {
	c := global.Cfg()
	fd := global.SetupLogger(c)
	defer fd.Close()

	// Channel to listen for OS signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	app := node.NewApp(c, stop)

	app.PrintVLESSConnectionURLS()
	go app.Run()
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	app.Shutdown(ctx)
}
