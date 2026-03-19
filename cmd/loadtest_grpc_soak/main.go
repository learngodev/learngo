package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	"learn-go/api/grpcpb"
)

type counters struct {
	dialOK         atomic.Int64
	dialFail       atomic.Int64
	streamOK       atomic.Int64
	streamFail     atomic.Int64
	joinOK         atomic.Int64
	joinFail       atomic.Int64
	joinTimeout    atomic.Int64
	snapshotOK     atomic.Int64
	snapshotFail   atomic.Int64
	endedOK        atomic.Int64
	endedWithError atomic.Int64

	firstErrorOnce sync.Once
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

		addr         = flag.String("addr", envOrDefault("GRPC_ADDR", ""), "(optional) override gRPC address, e.g. 127.0.0.1:9090")
		token        = flag.String("token", "", "(optional) JWT access token (without Bearer prefix). If omitted, tool will login via HTTP when identifier/password are provided")
		verbose      = flag.Bool("verbose", false, "print the first connection/stream error for debugging")
		conversation = flag.String("conversation", os.Getenv("CONVERSATION_ID"), "conversation id to join. Can also set CONVERSATION_ID env var")
		conns        = flag.Int("conns", envOrDefaultInt("CONNS", 100), "number of concurrent client connections (each opens 1 bidi stream)")
		duration     = flag.Duration("duration", envOrDefaultDuration("DURATION", 2*time.Minute), "how long to keep connections open")
		joinTimeout  = flag.Duration("join-timeout", 10*time.Second, "timeout waiting for initial snapshot after join")
		reportEvery  = flag.Duration("report", 5*time.Second, "progress report interval")
	)
	flag.Parse()

	grpcAddr := stringsTrimSpace(*addr)
	if grpcAddr == "" {
		grpcAddr = fmt.Sprintf("%s:%d", stringsTrimSpace(*host), *grpcPort)
	}

	// Token resolution priority:
	// 1) explicit -token
	// 2) if identifier/password provided => login via HTTP (preferred)
	// 3) fallback to TOKEN env var
	if stringsTrimSpace(*token) == "" {
		if stringsTrimSpace(*identifier) != "" || stringsTrimSpace(*password) != "" {
			if stringsTrimSpace(*identifier) == "" {
				fatal("missing credentials: set -identifier/IDENTIFIER")
			}
			if stringsTrimSpace(*password) == "" {
				fatal("missing credentials: set -password/PASSWORD")
			}

			httpBase := fmt.Sprintf("http://%s:%d", stringsTrimSpace(*host), *httpPort)
			loginURL := httpBase + stringsTrimSpace(*loginPath)
			accessToken, role, accountID, err := loginForToken(loginURL, *schoolID, *identifier, *password, *loginTimeout)
			if err != nil {
				fatal("login failed: " + err.Error())
			}
			*token = accessToken
			fmt.Printf("login OK (role=%s accountId=%s tokenLen=%d)\n", role, accountID, len(accessToken))
		} else {
			*token = os.Getenv("TOKEN")
		}
	}
	if stringsTrimSpace(*token) == "" {
		fatal("missing token: provide -token, or provide -identifier/-password for auto-login, or set TOKEN env var")
	}

	if *conversation == "" {
		fatal("missing conversation id: set -conversation or CONVERSATION_ID env var")
	}
	if *conns <= 0 {
		fatal("-conns must be > 0")
	}

	ctx, cancel := context.WithTimeout(context.Background(), *duration)
	defer cancel()

	ctx = withSignalCancel(ctx, cancel)

	var c counters
	start := time.Now()

	var active atomic.Int64
	active.Store(0)

	var wg sync.WaitGroup
	wg.Add(*conns)

	for i := 0; i < *conns; i++ {
		go func(idx int) {
			defer wg.Done()
			runOne(ctx, grpcAddr, *token, *conversation, *joinTimeout, *verbose, &c, &active)
		}(i)
	}

	ticker := time.NewTicker(*reportEvery)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			wg.Wait()
			printSummary(time.Since(start), &c, active.Load())
			return
		case <-ticker.C:
			printSummary(time.Since(start), &c, active.Load())
		}
	}
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
		SchoolID:   stringsTrimSpace(schoolID),
		Identifier: stringsTrimSpace(identifier),
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
		// Avoid printing tokens; include http status to help debugging.
		return "", "", "", fmt.Errorf("login unsuccessful (http=%d)", resp.StatusCode)
	}
	if stringsTrimSpace(out.Data.AccessToken) == "" {
		return "", "", "", fmt.Errorf("login response missing access_token")
	}
	return out.Data.AccessToken, out.Data.Account.Role, out.Data.Account.ID, nil
}

func runOne(ctx context.Context, addr, token, conversationID string, joinTimeout time.Duration, verbose bool, c *counters, active *atomic.Int64) {
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

	// Join conversation.
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

	// Wait for initial snapshot (or any first response) to ensure server accepted join.
	firstRespCh := make(chan error, 1)
	go func() {
		resp, err := stream.Recv()
		if err != nil {
			c.firstErrorOnce.Do(func() {
				if verbose {
					fmt.Printf("first recv error: %v\n", err)
				}
			})
			firstRespCh <- err
			return
		}
		if resp.GetSnapshot() != nil {
			c.snapshotOK.Add(1)
		} else {
			// Some servers may send other first payloads; treat as ok but record.
			c.snapshotFail.Add(1)
		}
		firstRespCh <- nil
	}()

	select {
	case <-time.After(joinTimeout):
		c.joinTimeout.Add(1)
		_ = stream.CloseSend()
		return
	case err := <-firstRespCh:
		if err != nil {
			c.snapshotFail.Add(1)
			_ = stream.CloseSend()
			return
		}
	}

	// Soak: keep receiving until context is done or server closes.
	for {
		_, err := stream.Recv()
		if err != nil {
			if ctx.Err() != nil {
				c.endedOK.Add(1)
			} else {
				c.endedWithError.Add(1)
			}
			return
		}
	}
}

func printSummary(elapsed time.Duration, c *counters, active int64) {
	fmt.Printf("elapsed=%s conns_active=%d dial_ok=%d dial_fail=%d stream_ok=%d stream_fail=%d join_ok=%d join_fail=%d join_timeout=%d snapshot_ok=%d ended_ok=%d ended_err=%d\n",
		elapsed.Truncate(time.Second),
		active,
		c.dialOK.Load(),
		c.dialFail.Load(),
		c.streamOK.Load(),
		c.streamFail.Load(),
		c.joinOK.Load(),
		c.joinFail.Load(),
		c.joinTimeout.Load(),
		c.snapshotOK.Load(),
		c.endedOK.Load(),
		c.endedWithError.Load(),
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

func stringsTrimSpace(s string) string {
	return strings.TrimSpace(s)
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
