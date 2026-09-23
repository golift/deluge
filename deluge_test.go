package deluge_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"golift.io/deluge"
)

const exampleURL = "http://deluge.example/json"

func TestNewNoAuthURLAndBasicAuth(t *testing.T) {
	t.Parallel()

	client, err := deluge.NewNoAuth(&deluge.Config{
		URL:      exampleURL,
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

	if req.URL.String() != exampleURL {
		t.Fatalf("url %s", req.URL)
	}

	wantAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("alice:secret"))
	if req.Header.Get("Authorization") != wantAuth {
		t.Fatalf("auth %q", req.Header.Get("Authorization"))
	}

	if req.Header.Get("Content-Type") != "application/json" {
		t.Fatalf("content-type %q", req.Header.Get("Content-Type"))
	}

	if got := requestID(t, req); got != 1 {
		t.Fatalf("id %d", got)
	}

	req, err = client.DelReq(context.Background(), deluge.GetAllTorrents, []string{"", ""})
	if err != nil {
		t.Fatal(err)
	}

	if got := requestID(t, req); got != 2 {
		t.Fatalf("id %d", got)
	}
}

func requestID(t *testing.T, req *http.Request) int64 {
	t.Helper()

	body, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatal(err)
	}

	var call struct {
		ID int64 `json:"id"`
	}

	err = json.Unmarshal(body, &call)
	if err != nil {
		t.Fatal(err)
	}

	return call.ID
}

func TestNewLoginAndVersion(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(fakeDeluge())
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

func fakeDeluge() http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
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

		writeDelugeMethod(resp, call.Method)
	})
}

func writeDelugeMethod(resp http.ResponseWriter, method string) {
	switch method {
	case deluge.AuthLogin:
		_, _ = resp.Write([]byte(`{"id":1,"result":true}`))
	case deluge.GeHosts:
		_, _ = resp.Write([]byte(`{"id":2,"result":[["abc","127.0.0.1",58846,"http"]]}`))
	case deluge.HostStatus:
		_, _ = resp.Write([]byte(`{"id":3,"result":["Online","127.0.0.1:58846","2.1.1"]}`))
	default:
		http.Error(resp, method, http.StatusBadRequest)
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

func TestDelReqConcurrentIDs(t *testing.T) {
	t.Parallel()

	client, err := deluge.NewNoAuth(&deluge.Config{URL: exampleURL})
	if err != nil {
		t.Fatal(err)
	}

	const calls = 32

	var wait sync.WaitGroup

	ids := make([]int64, calls)
	wait.Add(calls)

	for index := range ids {
		go func(index int) {
			defer wait.Done()

			ids[index] = delReqID(t, client)
		}(index)
	}

	wait.Wait()

	seen := make(map[int64]struct{}, calls)
	for _, id := range ids {
		if _, ok := seen[id]; ok || id == 0 {
			t.Fatalf("id %d in %v", id, ids)
		}

		seen[id] = struct{}{}
	}
}

func delReqID(t *testing.T, client *deluge.Deluge) int64 {
	t.Helper()

	req, err := client.DelReq(context.Background(), deluge.AuthLogin, []string{"x"})
	if err != nil {
		t.Errorf("DelReq: %v", err)

		return 0
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		t.Errorf("read body: %v", err)

		return 0
	}

	var call struct {
		ID int64 `json:"id"`
	}

	err = json.Unmarshal(body, &call)
	if err != nil {
		t.Errorf("json: %v", err)

		return 0
	}

	return call.ID
}
