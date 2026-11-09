package mcpserver

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInitializeAndToolsList(t *testing.T) {
	s := &Server{APIURL: "http://x", APIKey: "k", Client: &http.Client{}, In: strings.NewReader(""), Out: &bytes.Buffer{}}
	// initialize
	var buf bytes.Buffer
	s.Out = &buf
	s.handle(rpcReq{JSONRPC: "2.0", ID: json.RawMessage("1"), Method: "initialize"})
	// tools/list
	buf.Reset()
	s.handle(rpcReq{JSONRPC: "2.0", ID: json.RawMessage("2"), Method: "tools/list"})
	// call directly
	res, err := s.handle(rpcReq{JSONRPC: "2.0", ID: json.RawMessage("2"), Method: "tools/list"})
	if err != nil {
		t.Fatal(err)
	}
	m := res.(map[string]any)
	tools := m["tools"].([]map[string]any)
	if len(tools) != 5 {
		t.Fatalf("expected 5 tools got %d", len(tools))
	}
	_ = buf
	_ = http.MethodGet
	_ = httptest.NewServer
}

func TestCreateDevisAmbiguity(t *testing.T) {
	s := &Server{APIURL: "http://x", APIKey: "k", Client: &http.Client{}}
	res, err := s.callTool("create_devis", json.RawMessage(`{"client":"c","lignes":[],"valide_jusqu_au":"2026-01-01","delai_validite_jours":"30"}`))
	if err != nil {
		t.Fatal(err)
	}
	if res["status"] != "clarification" {
		t.Fatalf("expected clarification got %v", res["status"])
	}
}

func TestRunStdio(t *testing.T) {
	in := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"ping"}` + "\n")
	var out bytes.Buffer
	s := &Server{APIURL: "http://x", APIKey: "k", Client: &http.Client{}, In: in, Out: &out}
	if err := s.Run(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "result") {
		t.Fatalf("no result: %s", out.String())
	}
}
