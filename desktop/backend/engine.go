package backend

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/samosvalishe/free-turn-proxy/desktop/backend/notify"
	"github.com/samosvalishe/free-turn-proxy/desktop/backend/wintunnel"
	"github.com/samosvalishe/free-turn-proxy/mobile"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// Status is a UI-friendly snapshot of the running (or last) session.
type Status struct {
	State      string `json:"state"` // idle | connecting | connected | captcha | error
	Streams    int    `json:"streams"`
	Total      int    `json:"total"`
	ErrMsg     string `json:"errMsg"`
	TxRate     int64  `json:"txRate"`
	RxRate     int64  `json:"rxRate"`
	TxTotal    int64  `json:"txTotal"`
	RxTotal    int64  `json:"rxTotal"`
	ServerID   string `json:"serverId"`
	CaptchaURL string `json:"captchaUrl"`
}

// LogLine is pushed to the frontend over the "core:log" event as the session runs.
type LogLine struct {
	Level string `json:"level"`
	Msg   string `json:"msg"`
	AtMs  int64  `json:"atMs"`
}

const (
	eventLog     = "core:log"
	eventState   = "core:state"
	eventStats   = "core:stats"
	eventCaptcha = "core:captcha"
)

// Engine drives the in-process free-turn-proxy core (github.com/.../mobile) and
// bridges its callbacks to Wails frontend events. Only one session can run at a
// time - that mirrors the core's own single "current session" model.
type Engine struct {
	ctx   context.Context
	store *Store

	mu         sync.Mutex
	serverID   string
	planCancel context.CancelFunc
	captchaURL string
	traffic    trafficMeter
}

func NewEngine(store *Store) *Engine {
	e := &Engine{store: store}
	mobile.SetStateDir(store.StateDir())
	mobile.SetEventSink(e)
	return e
}

func (e *Engine) SetContext(ctx context.Context) {
	e.ctx = ctx
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				wailsruntime.EventsEmit(ctx, eventStats, e.Status())
			}
		}
	}()
}

func (e *Engine) Connect(srv Server) error {
	e.mu.Lock()
	if e.planCancel != nil {
		e.planCancel()
	}
	ctx := e.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithCancel(ctx)
	e.planCancel = cancel
	e.mu.Unlock()
	defer cancel()
	if srv.Peer == "" {
		return errors.New("peer is required")
	}
	vpn := srv.ConnMode == ConnModeVPN
	if vpn && srv.WgClientConf == "" {
		return errors.New("VPN mode needs a WireGuard config - none saved on this server")
	}
	cfg, err := srv.ToCoreConfig().JSON()
	if err != nil {
		return err
	}
	if verr := mobile.ValidateConfig(cfg); verr != "" {
		return errors.New(verr)
	}
	var exclusions []string
	if vpn {
		exclusions, err = resolveExclusions(ctx, srv, net.DefaultResolver)
		if err != nil {
			return fmt.Errorf("VPN endpoints: %w", err)
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	// Unconditionally clear whatever might already be running, including a
	// half-failed previous attempt (e.g. the adapter came up but the tunnel
	// handshake didn't) that would otherwise block every further Connect with
	// "already running"/"already connected" and leave no way back except
	// force-stop. Stop/Disconnect are safe no-ops when nothing is up.
	e.forceStop()

	e.mu.Lock()
	e.serverID = srv.ID
	e.traffic = trafficMeter{}
	e.mu.Unlock()

	if vpn {
		if err := ctx.Err(); err != nil {
			return err
		}
		return wintunnel.Connect(cfg, srv.WgClientConf, DefaultTunnelMTU, exclusions...)
	}
	return mobile.Start(cfg)
}

func (e *Engine) Disconnect() {
	e.mu.Lock()
	if e.planCancel != nil {
		e.planCancel()
		e.planCancel = nil
	}
	e.mu.Unlock()
	e.forceStop()
}

func (e *Engine) forceStop() {
	e.mu.Lock()
	e.captchaURL = ""
	e.mu.Unlock()
	mobile.Stop()
	wintunnel.Disconnect()
}

func (e *Engine) Status() Status {
	e.mu.Lock()
	defer e.mu.Unlock()
	snap := mobile.GetState()
	txRate, rxRate := e.traffic.sample(snap.TxTotal, snap.RxTotal, time.Now())
	if snap.State != "connected" {
		txRate, rxRate = 0, 0
	}
	return Status{
		State: snap.State, Streams: snap.Streams, Total: snap.Total, ErrMsg: snap.ErrMsg,
		TxRate: txRate, RxRate: rxRate, TxTotal: snap.TxTotal, RxTotal: snap.RxTotal,
		ServerID: e.serverID, CaptchaURL: e.captchaURL,
	}
}

// ConfigCommand returns the equivalent `client ...` CLI invocation for a profile,
// for users who want to sanity-check or copy it (mirrors the Android "command" screen).
func (e *Engine) ConfigCommand(srv Server) (string, error) {
	cfg, err := srv.ToCoreConfig().JSON()
	if err != nil {
		return "", err
	}
	args, err := mobile.ConfigToArgs(cfg)
	if err != nil {
		return "", err
	}
	return "client " + args, nil
}

// --- mobile.EventSink ---

func (e *Engine) OnState(state string, streams, total int, errMsg string) {
	if e.ctx == nil {
		return
	}
	e.mu.Lock()
	serverID := e.serverID
	if state == "idle" || state == "error" {
		e.captchaURL = ""
	}
	captchaURL := e.captchaURL
	e.mu.Unlock()
	wailsruntime.EventsEmit(e.ctx, eventState, Status{State: state, Streams: streams, Total: total, ErrMsg: errMsg, ServerID: serverID, CaptchaURL: captchaURL})
}

func (e *Engine) OnLog(level, msg string, unixMillis int64) {
	if e.ctx == nil {
		return
	}
	wailsruntime.EventsEmit(e.ctx, eventLog, LogLine{Level: level, Msg: msg, AtMs: unixMillis})
}

func (e *Engine) OnCaptcha(url string) {
	e.mu.Lock()
	e.captchaURL = url
	e.mu.Unlock()
	if e.ctx == nil {
		return
	}
	wailsruntime.EventsEmit(e.ctx, eventCaptcha, url)
	wailsruntime.EventsEmit(e.ctx, eventStats, e.Status())
	if url != "" {
		go func() {
			_ = notify.ShowCaptchaNotification(url, nil)
		}()
	}
}
