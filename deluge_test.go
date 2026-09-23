package deluge_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"golift.io/deluge"
)

func TestNewNoAuthURLAndBasicAuth(t *testing.T) {
	t.Parallel()

	client, err := deluge.NewNoAuth(&deluge.Config{
		URL:      "http://deluge.example/json",
		HTTPUser: "alice",
		HTTPPass: "secret",
	})
	if err != nil {
		t.Fatal(err)
	}

	req, err := client.DelReq(context.Background(), deluge.GetAllTorrents, []string{"", ""})
	if err != nil {
		t.Fatal(err)
	}

	if req.URL.String() != "http://deluge.example/json" {
		t.Fatalf("url %s", req.URL)
	}

	wantAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("alice:secret"))
	if req.Header.Get("Authorization") != wantAuth {
		t.Fatalf("auth %q", req.Header.Get("Authorization"))
	}

	if req.Header.Get("Content-Type") != "application/json" {
		t.Fatalf("content-type %q", req.Header.Get("Content-Type"))
	}
}

func TestNewLoginAndVersion(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/json" {
			http.Error(resp, "bad path", http.StatusNotFound)

			return
		}

		body, err := io.ReadAll(req.Body)
		if err != nil {
			http.Error(resp, err.Error(), http.StatusInternalServerError)

			return
		}

		var call struct {
			Method string `json:"method"`
		}
		err = json.Unmarshal(body, &call)
		if err != nil {
			http.Error(resp, err.Error(), http.StatusBadRequest)

			return
		}

		switch call.Method {
		case deluge.AuthLogin:
			_, _ = resp.Write([]byte(`{"id":1,"result":true}`))
		case deluge.GeHosts:
			_, _ = resp.Write([]byte(`{"id":2,"result":[["abc","127.0.0.1",58846,"http"]]}`))
		case deluge.HostStatus:
			_, _ = resp.Write([]byte(`{"id":3,"result":["Online","127.0.0.1:58846","2.1.1"]}`))
		default:
			http.Error(resp, call.Method, http.StatusBadRequest)
		}
	}))
	t.Cleanup(srv.Close)

	client, err := deluge.New(context.Background(), &deluge.Config{
		URL:      srv.URL + "/",
		Password: "deluge",
	})
	if err != nil {
		t.Fatal(err)
	}

	if client.Version != "2.1.1" {
		t.Fatalf("version %q", client.Version)
	}

	backend, ok := client.Backends["abc"]
	if !ok {
		t.Fatalf("backends %#v", client.Backends)
	}

	if backend.Addr != "127.0.0.1:58846" || backend.Prot != "http" {
		t.Fatalf("backend %#v", backend)
	}
}

func TestNewAuthFailed(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, _ *http.Request) {
		resp.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(srv.Close)

	_, err := deluge.New(context.Background(), &deluge.Config{URL: srv.URL, Password: "nope"})
	if !errors.Is(err, deluge.ErrAuthFailed) {
		t.Fatalf("err %v", err)
	}
}
