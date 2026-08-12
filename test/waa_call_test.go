//go:build live

package test

import (
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/pablo-botella/go2waa"
)

// waaCallRequest is the Go flavor of the Xbase ao_waa_call request file:
// the "application" block (in-process DLL) is replaced by "server"
// (host, port), the call travels over TCP.
type waaCallRequest struct {
	Server struct {
		Host                   string `json:"host"`
		Port                   int    `json:"port"`
		OutputFolder           string `json:"output_folder"`
		ResponseHeaderFilename string `json:"response_header_filename"`
		ResponseBodyFilename   string `json:"response_body_filename"`
		UseProcessEnvironment  bool   `json:"use_process_environment"`
	} `json:"server"`
	Environment map[string]string          `json:"environment"`
	Payload     map[string]json.RawMessage `json:"payload"`
}

// TestWaaCall performs one real call against a live WAA server, driven
// by a JSON request file. The file comes from WAA_CALL_REQUEST or
// defaults to waa/ao_waa_call_request.json; WAA_HOST and WAA_PORT
// override the server block. Skips when there is no request file or no
// server listening.
func TestWaaCall(t *testing.T) {
	reqPath := os.Getenv("WAA_CALL_REQUEST")
	if reqPath == "" {
		reqPath = "test.json"
	}
	data, err := os.ReadFile(reqPath)
	if err != nil {
		t.Fatalf("no request file: %v", err)
	}

	var req waaCallRequest
	if err := json.Unmarshal(data, &req); err != nil {
		t.Fatalf("bad request file %s: %v", reqPath, err)
	}

	// WAA_HOST / WAA_PORT override the JSON
	host := req.Server.Host
	if v := os.Getenv("WAA_HOST"); v != "" {
		host = v
	}
	port := req.Server.Port
	if v := os.Getenv("WAA_PORT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			port = n
		}
	}

	// Environment: process environment first if requested, then the
	// environment block on top
	env := make(map[string]string)
	if req.Server.UseProcessEnvironment {
		for _, kv := range os.Environ() {
			if k, v, ok := strings.Cut(kv, "="); ok {
				env[k] = v
			}
		}
	}
	maps.Copy(env, req.Environment)

	// Payload: string or array of strings per variable
	ctx := go2waa.NewDirectWaaCtx(env)
	for name, raw := range req.Payload {
		var single string
		if err := json.Unmarshal(raw, &single); err == nil {
			ctx.SetValue(name, single)
			continue
		}
		var multi []string
		if err := json.Unmarshal(raw, &multi); err == nil {
			ctx.SetValues(name, multi)
			continue
		}
		t.Fatalf("payload %q: value must be string or array of strings", name)
	}

	// The call
	client := &go2waa.WaaClient{Host: host, Port: port}
	if err := client.Call(ctx); err != nil {
		t.Fatalf("call failed against %s:%d: %v", host, port, err)
	}

	// Write the response files: header one line per entry, CRLF
	outDir := req.Server.OutputFolder
	if outDir == "" {
		outDir = "."
	}
	if req.Server.ResponseHeaderFilename != "" {
		headerData := ""
		if len(ctx.Header) > 0 {
			headerData = strings.Join(ctx.Header, "\r\n") + "\r\n"
		}
		path := filepath.Join(outDir, req.Server.ResponseHeaderFilename)
		if err := os.WriteFile(path, []byte(headerData), 0644); err != nil {
			t.Fatalf("writing header file: %v", err)
		}
	}
	if req.Server.ResponseBodyFilename != "" {
		path := filepath.Join(outDir, req.Server.ResponseBodyFilename)
		if err := os.WriteFile(path, ctx.Body, 0644); err != nil {
			t.Fatalf("writing body file: %v", err)
		}
	}

	t.Logf("call ok: %d header lines, %d body bytes", len(ctx.Header), len(ctx.Body))
	if len(ctx.Body) == 0 {
		t.Error("empty response body")
	}
}
