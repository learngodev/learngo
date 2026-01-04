package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	"learn-go/internal/api/grpcpb"
)

type counters struct {
	sentOK   atomic.Int64
	sentFail atomic.Int64

	ackOK   atomic.Int64
	ackFail atomic.Int64

	eventOK   atomic.Int64
	eventFail atomic.Int64

	joinOK   atomic.Int64
	joinFail atomic.Int64

	streamOK   atomic.Int64
	streamFail atomic.Int64

	dialOK   atomic.Int64
	dialFail atomic.Int64

	endedOK        atomic.Int64
	endedWithError atomic.Int64

	firstErrorOnce sync.Once
}

type latencyAgg struct {
	count atomic.Int64
	sumNs atomic.Int64
	minNs atomic.Int64
	maxNs atomic.Int64
}

func newLatencyAgg() *latencyAgg {
	a := &latencyAgg{}
	a.minNs.Store(int64(^uint64(0) >> 1))
	a.maxNs.Store(0)
	return a
}

func (a *latencyAgg) add(ns int64) {
	a.count.Add(1)
	a.sumNs.Add(ns)
	for {
		cur := a.minNs.Load()
		if ns >= cur {
			break
		}
		if a.minNs.CompareAndSwap(cur, ns) {
			break
		}
	}
	for {
		cur := a.maxNs.Load()
		if ns <= cur {
			break
		}
		if a.maxNs.CompareAndSwap(cur, ns) {
			break
		}
	}
}

func (a *latencyAgg) snapshot() (count int64, avg time.Duration, min time.Duration, max time.Duration) {
	c := a.count.Load()
	if c == 0 {
		return 0, 0, 0, 0
	}
	sum := a.sumNs.Load()
	return c, time.Duration(sum / c), time.Duration(a.minNs.Load()), time.Duration(a.maxNs.Load())
}

func main() {
	var (
		host         = flag.String("host", envOrDefault("HOST", "127.0.0.1"), "server host (used to build http/grpc addresses)")
		httpPort     = flag.Int("http-port", envOrDefaultInt("HTTP_PORT", 8080), "HTTP port for REST login")
		grpcPort     = flag.Int("grpc-port", envOrDefaultInt("GRPC_PORT", 9090), "gRPC port")
		loginPath    = flag.String("login-path", envOrDefault("LOGIN_PATH", "/api/v1/auth/login"), "REST login path")
		schoolID     = flag.String("school", envOrDefault("SCHOOL_ID", "11111111-1111-1111-1111-111111111111"), "school id for login")
		identifier   = flag.String("identifier", envOrDefault("IDENTIFIER", ""), "login identifier (e.g. stu-2025001). Can also set IDENTIFIER env var")
		password     = flag.String("password", envOrDefault("PASSWORD", ""), "login password. Prefer setting PASSWORD env var")
		loginTimeout = flag.Duration("login-timeout", envOrDefaultDuration("LOGIN_TIMEOUT", 10*time.Second), "REST login request timeout")

		addr    = flag.String("addr", envOrDefault("GRPC_ADDR", ""), "(optional) override gRPC address, e.g. 127.0.0.1:9090")
		token   = flag.String("token", "", "(optional) JWT access token (without Bearer prefix). If omitted, tool will login via HTTP when identifier/password are provided")
		verbose = flag.Bool("verbose", false, "print the first connection/stream error for debugging")

		conversation = flag.String("conversation", os.Getenv("CONVERSATION_ID"), "conversation id to join. Can also set CONVERSATION_ID env var")
		conns        = flag.Int("conns", envOrDefaultInt("CONNS", 10), "number of concurrent client connections (each opens 1 bidi stream)")
		duration     = flag.Duration("duration", envOrDefaultDuration("DURATION", 30*time.Second), "test duration")
		reportEvery  = flag.Duration("report", 2*time.Second, "progress report interval")

		rate = flag.Int("rate", envOrDefaultInt("RATE", 100), "target total send rate (messages/sec across ALL connections)")

		kind     = flag.String("kind", envOrDefault("KIND", "text"), "message kind")
		textSize = flag.Int("text-size", envOrDefaultInt("TEXT_SIZE", 64), "message text size (bytes)")
	)
	flag.Parse()

	grpcAddr := strings.TrimSpace(*addr)
	if grpcAddr == "" {
		grpcAddr = fmt.Sprintf("%s:%d", strings.TrimSpace(*host), *grpcPort)
	}

	*token = resolveTokenOrLogin(*token, *identifier, *password, *host, *httpPort, *loginPath, *schoolID, *loginTimeout)

	if strings.TrimSpace(*conversation) == "" {
		fatal("missing conversation id: set -conversation or CONVERSATION_ID env var")
	}
	if *conns <= 0 {
		fatal("-conns must be > 0")
	}
	if *duration <= 0 {
		fatal("-duration must be > 0")
	}
	if *rate < 0 {
		fatal("-rate must be >= 0")
	}
	if *textSize < 0 {
		fatal("-text-size must be >= 0")
	}

	ctx, cancel := context.WithTimeout(context.Background(), *duration)
	defer cancel()
	ctx = withSignalCancel(ctx, cancel)

	var c counters
	lat := newLatencyAgg()

	start := time.Now()
	var active atomic.Int64

	type connHandle struct {
		sendCh chan sendItem
		pend   *pendingMap
	}
	connsH := make([]connHandle, 0, *conns)

	for i := 0; i < *conns; i++ {
		h := connHandle{sendCh: make(chan sendItem, 1024), pend: newPendingMap()}
		connsH = append(connsH, h)
	}

	var wg sync.WaitGroup
	wg.Add(*conns)
	for i := 0; i < *conns; i++ {
		go func(idx int) {
			defer wg.Done()
			runConn(ctx, grpcAddr, *token, strings.TrimSpace(*conversation), *kind, makeText(*textSize), *verbose, &c, lat, &active, connsH[idx].sendCh, connsH[idx].pend)
		}(i)
	}

	// Global sender: pace total rate across all connections.
	var seq atomic.Uint64
	go func() {
		if *rate == 0 {
			return
		}
		interval := time.Second / time.Duration(*rate)
		if interval <= 0 {
			interval = time.Nanosecond
		}

		t := time.NewTicker(interval)
		defer t.Stop()

		i := 0
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-t.C:
				s := seq.Add(1)
				item := sendItem{seq: s, at: now}
				// Round-robin distribute.
				connsH[i%len(connsH)].sendCh <- item
				i++
			}
		}
	}()

	// Reporter
	last := time.Now()
	var lastSent, lastAck, lastEvent int64
	for {
		select {
		case <-ctx.Done():
			wg.Wait()
			printSummary(time.Since(start), &c, &active, lat)
			return
		case <-time.After(*reportEvery):
			now := time.Now()
			dt := now.Sub(last)
			if dt <= 0 {
				dt = *reportEvery
			}
			sent := c.sentOK.Load()
			ack := c.ackOK.Load()
			evt := c.eventOK.Load()
			fmt.Printf("elapsed=%s active=%d sent_ok=%d ack_ok=%d event_ok=%d sent_rps=%.1f ack_rps=%.1f event_rps=%.1f\n",
				now.Sub(start).Truncate(time.Second),
				active.Load(),
				sent,
				ack,
				evt,
				float64(sent-lastSent)/dt.Seconds(),
				float64(ack-lastAck)/dt.Seconds(),
				float64(evt-lastEvent)/dt.Seconds(),
			)
			last = now
			lastSent = sent
			lastAck = ack
			lastEvent = evt
		}
	}
}

type sendItem struct {
	seq uint64
	at  time.Time
}

type pendingMap struct {
	mu sync.Mutex
	m  map[uint64]time.Time
}

func newPendingMap() *pendingMap {
	return &pendingMap{m: make(map[uint64]time.Time, 1024)}
}

func (p *pendingMap) put(seq uint64, t time.Time) {
	p.mu.Lock()
	p.m[seq] = t
	p.mu.Unlock()
}

func (p *pendingMap) pop(seq uint64) (time.Time, bool) {
	p.mu.Lock()
	t, ok := p.m[seq]
	if ok {
		delete(p.m, seq)
	}
	p.mu.Unlock()
	return t, ok
}

func resolveTokenOrLogin(token, identifier, password, host string, httpPort int, loginPath, schoolID string, loginTimeout time.Duration) string {
	if strings.TrimSpace(token) != "" {
		return strings.TrimSpace(token)
	}
	if strings.TrimSpace(identifier) != "" || strings.TrimSpace(password) != "" {
		if strings.TrimSpace(identifier) == "" {
			fatal("missing credentials: set -identifier/IDENTIFIER")
		}
		if strings.TrimSpace(password) == "" {
			fatal("missing credentials: set -password/PASSWORD")
		}
		httpBase := fmt.Sprintf("http://%s:%d", strings.TrimSpace(host), httpPort)
		loginURL := httpBase + strings.TrimSpace(loginPath)
		accessToken, role, accountID, err := loginForToken(loginURL, schoolID, identifier, password, loginTimeout)
		if err != nil {
			fatal("login failed: " + err.Error())
		}
		fmt.Printf("login OK (role=%s accountId=%s tokenLen=%d)\n", role, accountID, len(accessToken))
		return accessToken
	}
	if v := os.Getenv("TOKEN"); strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	fatal("missing token: provide -token, or provide -identifier/-password for auto-login, or set TOKEN env var")
	return ""
}

type loginRequest struct {
	SchoolID   string `json:"school_id"`
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

type loginResponse struct {
	Success bool `json:"success"`
	Error   any  `json:"error"`
	Data    struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		Account      struct {
			ID   string `json:"id"`
			Role string `json:"role"`
		} `json:"account"`
	} `json:"data"`
}

func loginForToken(loginURL, schoolID, identifier, password string, timeout time.Duration) (token string, role string, accountID string, err error) {
	body, err := json.Marshal(loginRequest{
		SchoolID:   strings.TrimSpace(schoolID),
		Identifier: strings.TrimSpace(identifier),
		Password:   password,
	})
	if err != nil {
		return "", "", "", err
	}

	req, err := http.NewRequest(http.MethodPost, loginURL, bytes.NewReader(body))
	if err != nil {
		return "", "", "", err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", "", err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var out loginResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		return "", "", "", fmt.Errorf("decode login response: %w", err)
	}
	if !out.Success {
		return "", "", "", fmt.Errorf("login unsuccessful (http=%d)", resp.StatusCode)
	}
	if strings.TrimSpace(out.Data.AccessToken) == "" {
		return "", "", "", fmt.Errorf("login response missing access_token")
	}
	return out.Data.AccessToken, out.Data.Account.Role, out.Data.Account.ID, nil
}

func runConn(
	ctx context.Context,
	addr, token, conversationID, kind, text string,
	verbose bool,
	c *counters,
	lat *latencyAgg,
	active *atomic.Int64,
	sendCh <-chan sendItem,
	pend *pendingMap,
) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		c.dialFail.Add(1)
		c.firstErrorOnce.Do(func() {
			if verbose {
				fmt.Printf("first dial error: %v\n", err)
			}
		})
		return
	}
	defer conn.Close()
	c.dialOK.Add(1)

	client := grpcpb.NewConversationServiceClient(conn)
	md := metadata.Pairs("authorization", "Bearer "+token)
	streamCtx := metadata.NewOutgoingContext(ctx, md)

	stream, err := client.Stream(streamCtx)
	if err != nil {
		c.streamFail.Add(1)
		c.firstErrorOnce.Do(func() {
			if verbose {
				fmt.Printf("first stream error: %v\n", err)
			}
		})
		return
	}
	c.streamOK.Add(1)

	active.Add(1)
	defer active.Add(-1)

	// Join
	if err := stream.Send(&grpcpb.ConversationStreamRequest{
		Payload: &grpcpb.ConversationStreamRequest_Join{
			Join: &grpcpb.JoinConversation{ConversationId: conversationID},
		},
	}); err != nil {
		c.joinFail.Add(1)
		c.firstErrorOnce.Do(func() {
			if verbose {
				fmt.Printf("first join send error: %v\n", err)
			}
		})
		_ = stream.CloseSend()
		return
	}
	c.joinOK.Add(1)

	// Receiver
	recvDone := make(chan struct{})
	go func() {
		defer close(recvDone)
		for {
			resp, err := stream.Recv()
			if err != nil {
				if ctx.Err() != nil {
					c.endedOK.Add(1)
					// Normal shutdown due to context timeout/cancel.
					return
				}
				c.endedWithError.Add(1)
				c.firstErrorOnce.Do(func() {
					if verbose {
						fmt.Printf("first recv error: %v\n", err)
					}
				})
				return
			}
			if resp.GetMessageAck() != nil {
				c.ackOK.Add(1)
				seq, ok := parseSeqFromMetadata(resp.GetMessageAck().GetMessage().GetMetadata())
				if ok {
					if sentAt, found := pend.pop(seq); found {
						lat.add(time.Since(sentAt).Nanoseconds())
					}
				}
				continue
			}
			if resp.GetMessageEvent() != nil {
				c.eventOK.Add(1)
				continue
			}
			if resp.GetError() != nil {
				c.ackFail.Add(1)
				continue
			}
		}
	}()

	// Sender
	for {
		select {
		case <-ctx.Done():
			_ = stream.CloseSend()
			<-recvDone
			return
		case <-recvDone:
			return
		case item := <-sendCh:
			meta := formatMetadata(item.seq, item.at)
			pend.put(item.seq, item.at)
			err := stream.Send(&grpcpb.ConversationStreamRequest{
				Payload: &grpcpb.ConversationStreamRequest_Create{
					Create: &grpcpb.MessageCreate{
						ConversationId: conversationID,
						Kind:           kind,
						Text:           text,
						Metadata:       meta,
					},
				},
			})
			if err != nil {
				c.sentFail.Add(1)
				c.firstErrorOnce.Do(func() {
					if verbose {
						fmt.Printf("first send error: %v\n", err)
					}
				})
				continue
			}
			c.sentOK.Add(1)
		}
	}
}

func formatMetadata(seq uint64, at time.Time) string {
	// Intentionally plain format to avoid JSON parse overhead: seq=<n>;ts=<unixnano>
	return "seq=" + strconv.FormatUint(seq, 10) + ";ts=" + strconv.FormatInt(at.UnixNano(), 10)
}

func parseSeqFromMetadata(s string) (uint64, bool) {
	// Expect: seq=<n>;ts=<...>
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	parts := strings.Split(s, ";")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		k, v, ok := strings.Cut(p, "=")
		if !ok {
			continue
		}
		if k == "seq" {
			n, err := strconv.ParseUint(v, 10, 64)
			if err != nil {
				return 0, false
			}
			return n, true
		}
	}
	return 0, false
}

func makeText(n int) string {
	if n <= 0 {
		return ""
	}
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = alphabet[rand.Intn(len(alphabet))]
	}
	return string(b)
}

func printSummary(elapsed time.Duration, c *counters, active *atomic.Int64, lat *latencyAgg) {
	cnt, avg, min, max := lat.snapshot()
	fmt.Printf("SUMMARY elapsed=%s active=%d dial_ok=%d dial_fail=%d stream_ok=%d stream_fail=%d join_ok=%d join_fail=%d sent_ok=%d sent_fail=%d ack_ok=%d event_ok=%d ended_ok=%d ended_err=%d ack_latency_count=%d ack_latency_avg=%s ack_latency_min=%s ack_latency_max=%s\n",
		elapsed.Truncate(time.Second),
		active.Load(),
		c.dialOK.Load(), c.dialFail.Load(),
		c.streamOK.Load(), c.streamFail.Load(),
		c.joinOK.Load(), c.joinFail.Load(),
		c.sentOK.Load(), c.sentFail.Load(),
		c.ackOK.Load(), c.eventOK.Load(),
		c.endedOK.Load(), c.endedWithError.Load(),
		cnt, avg, min, max,
	)
}

func withSignalCancel(ctx context.Context, cancel context.CancelFunc) context.Context {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		select {
		case <-ctx.Done():
			return
		case <-sigCh:
			cancel()
		}
	}()
	return ctx
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envOrDefaultInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		var out int
		if _, err := fmt.Sscanf(v, "%d", &out); err == nil {
			return out
		}
	}
	return def
}

func envOrDefaultDuration(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		d, err := time.ParseDuration(v)
		if err == nil {
			return d
		}
	}
	return def
}

func fatal(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(2)
}
