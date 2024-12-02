package node

import (
	"encoding/json"
	"net/http"
	"strings"
)

func (app *App) Stat(w http.ResponseWriter, _ *http.Request) {

	all, err := json.Marshal(app.stat())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	//json response hello world
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(all)
}

func (app *App) Ping(w http.ResponseWriter, _ *http.Request) {
	//json response hello world
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	lines := []string{
		"BUILT HASH:  https://github.com/unchainese/unchain/tree/" + app.cfg.GitHash,
		"BUILT TIME:  " + app.cfg.BuildTime,
	}
	w.Write([]byte(strings.Join(lines, "\n\n")))
}
