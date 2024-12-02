package global

import (
	"fmt"
	"github.com/BurntSushi/toml"
	"log"
	"log/slog"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	SubAddresses       []string `desc:"sub addresses" example:"node1.xxx.cn:80,node2.xxx.cn:443"`
	ListenAddr         string   `desc:"net listen addr" def:"0.0.0.0:80"`
	RegisterUrl        string   `desc:"register url" def:"https://admin.unchain.people.from.censorship"`
	RegisterToken      string   `desc:"register token" def:"unchain people from censorship and surveillance"`
	AllowUsers         string   `desc:"allow users" def:"" example:"903bcd04-79e7-429c-bf0c-0456c7de9cdc,903bcd04-79e7-429c-bf0c-0456c7de9cd1"`
	LogFile            string   `desc:"log file path" def:""`
	DebugLevel         string   `desc:"debug level" def:"DEBUG"`
	PushIntervalSecond int      `desc:"push interval" def:"360"` //seconds
	GitHash            string   `desc:"git hash" def:""`
	BuildTime          string   `desc:"build time" def:""`
}

func (c Config) ListenPort() int {
	parts := strings.Split(c.ListenAddr, ":")
	if len(parts) < 2 {
		return 80
	}
	iv, err := strconv.ParseInt(parts[1], 10, 32)
	if err != nil {
		log.Println("failed to parse port:", err)
		return 80
	}
	return int(iv)
}

var (
	gitHash   string
	buildTime string
)

var cfg *Config

func Cfg() *Config {
	if cfg != nil {
		return cfg
	}
	cfgIns, err := loadFromToml("config.toml")
	if err != nil {
		panic(err)
	} else {
		cfg = cfgIns
	}
	cfg.GitHash = gitHash
	cfg.BuildTime = buildTime
	return cfg
}

func loadFromToml(file string) (*Config, error) {
	opt := Config{}
	_, err := toml.DecodeFile(file, &opt)
	if err != nil {
		return nil, fmt.Errorf("failed to load config file:%s %w", file, err)
	}
	return &opt, nil
}

func (c Config) LogLevel() slog.Level {
	l := slog.LevelDebug
	switch strings.ToUpper(c.DebugLevel) {
	case "DEBUG":
		l = slog.LevelDebug
	case "INFO":
		l = slog.LevelInfo
	case "WARN":
		l = slog.LevelWarn
	case "ERROR":
		l = slog.LevelError
	default:
		l = slog.LevelError
	}
	return l
}
func (c Config) UserIDS() []string {
	parts := strings.Split(c.AllowUsers, ",")
	ids := make([]string, 0)
	for _, uid := range parts {
		uid = strings.TrimSpace(uid)
		if uid != "" {
			ids = append(ids, uid)
		}
	}
	return ids
}

func (c Config) PushInterval() time.Duration {
	if c.PushIntervalSecond <= 0 {
		return time.Minute * 60
	}
	return time.Second * time.Duration(c.PushIntervalSecond)
}
