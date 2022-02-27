package deluge

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/publicsuffix"
	"golift.io/datacounter"
)

// Custom errors.
var (
	ErrInvalidVersion = fmt.Errorf("invalid data returned while checking version")
	ErrDelugeError    = fmt.Errorf("deluge error")
	ErrAuthFailed     = fmt.Errorf("authentication failed")
)

type Client struct {
	*http.Client
	cookie bool
}

// Deluge is what you get for providing a password.
type Deluge struct {
	*Client
	*Config
	auth     string
	id       int
	Version  string             // Currently unused, for display purposes only.
	Backends map[string]Backend // Currently unused, for display purposes only.
	DebugLog func(msg string, fmt ...interface{})
}

// NewNoAuth returns a Deluge object without authenticating or trying to connect.
func NewNoAuth(config *Config) (int64, *Deluge, error) {
	return newConfig(config, false)
}

// New creates a http.Client with authenticated cookies.
// Used to make additional, authenticated requests to the APIs.
func New(config *Config) (int64, *Deluge, error) {
	return newConfig(config, true)
}

func newConfig(config *Config, login bool) (int64, *Deluge, error) {
	var bytes int64

	// The cookie jar is used to auth Deluge.
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return bytes, nil, fmt.Errorf("cookiejar.New(publicsuffix): %w", err)
	}

	config.URL = strings.TrimSuffix(strings.TrimSuffix(config.URL, "/json"), "/") + "/json"

	// This app allows http auth, in addition to deluge web password.
	if both := config.HTTPUser + ":" + config.HTTPPass; both != ":" {
		config.HTTPUser = "Basic " + base64.StdEncoding.EncodeToString([]byte(both))
	} else {
		config.HTTPUser = ""
	}

	deluge := &Deluge{
		Config:   config,
		auth:     config.HTTPUser,
		Backends: make(map[string]Backend),
		DebugLog: config.DebugLog,
		Client: &Client{
			Client: &http.Client{
				Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: config.VerifySSL}}, //nolint:gosec
				Jar:       jar,
				Timeout:   config.Timeout.Round(time.Millisecond),
			},
		},
	}

	if !login {
		return bytes, deluge, nil
	}

	size, err := deluge.Login()
	bytes += size

	if err != nil {
		return bytes, deluge, err
	}

	if deluge.Version = config.Version; deluge.Version == "" {
		size, err = deluge.setVersion()
		bytes += size

		if err != nil {
			return bytes, deluge, err
		}
	}

	return bytes, deluge, nil
}

// Login sets the cookie jar with authentication information.
func (d *Deluge) Login() (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout.Duration)
	defer cancel()

	// This []string{config.Password} line is how you send auth creds. It's weird.
	req, err := d.DelReq(ctx, AuthLogin, []string{d.Password})
	if err != nil {
		return 0, fmt.Errorf("DelReq(AuthLogin, json): %w", err)
	}

	resp, err := d.Do(req)
	if err != nil {
		return 0, fmt.Errorf("d.Do(req): %w", err)
	}
	defer resp.Body.Close()

	size, _ := io.Copy(ioutil.Discard, resp.Body) // must read body to avoid memory leak.

	if resp.StatusCode != http.StatusOK {
		return size, fmt.Errorf("%w: %v[%v] (status: %v/%v)",
			ErrAuthFailed, req.URL.String(), AuthLogin, resp.StatusCode, resp.Status)
	}

	d.Client.cookie = true

	return size, nil
}

// setVersion digs into the first server in the web UI to find the version.
func (d *Deluge) setVersion() (int64, error) {
	bytes, response, err := d.Get(GeHosts, []string{})
	if err != nil {
		return bytes, err
	}

	// This method returns a "mixed list" which requires an interface.
	// Deluge devs apparently hate Go. :(
	servers := make([][]interface{}, 0)
	if err := json.Unmarshal(response.Result, &servers); err != nil {
		d.logPayload(response.Result)
		return bytes, fmt.Errorf("json.Unmarshal(rawResult1): %w", err)
	}

	serverID := ""

	// Store each server info (so consumers can access them easily).
	for _, server := range servers {
		serverID, _ = server[0].(string)
		d.Backends[serverID] = Backend{
			ID:   serverID,
			Addr: server[1].(string) + ":" + strconv.FormatFloat(server[2].(float64), 'f', 0, 64), //nolint:gomnd
			Prot: server[3].(string),
		}
	}

	total := bytes

	// Store the last server's version as "the version"
	bytes, response, err = d.Get(HostStatus, []string{serverID})
	if err != nil {
		return total + bytes, err
	}

	total += bytes

	server := make([]interface{}, 0)
	if err = json.Unmarshal(response.Result, &server); err != nil {
		d.logPayload(response.Result)
		return total, fmt.Errorf("json.Unmarshal(rawResult2): %w", err)
	}

	const payloadSegments = 3

	if len(server) < payloadSegments {
		d.logPayload(response.Result)
		return total, ErrInvalidVersion
	}

	// Version comes last in the mixed list.
	var ok bool
	if d.Version, ok = server[len(server)-1].(string); !ok {
		return total, ErrInvalidVersion
	}

	return total, nil
}

// DelReq is a small helper function that adds headers and marshals the json.
func (d Deluge) DelReq(ctx context.Context, method string, params interface{}) (req *http.Request, err error) {
	d.id++

	paramMap := map[string]interface{}{"method": method, "id": d.id, "params": params}
	if data, errr := json.Marshal(paramMap); errr != nil {
		return req, fmt.Errorf("json.Marshal(params): %w", err)
	} else if req, err = http.NewRequestWithContext(ctx, "POST", d.URL, bytes.NewBuffer(data)); err == nil {
		if d.auth != "" {
			// In case Deluge is also behind HTTP auth.
			req.Header.Add("Authorization", d.auth)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Add("Accept", "application/json")
	}

	return
}

// GetXfers gets all the Transfers from Deluge.
func (d Deluge) GetXfers() (int64, map[string]*XferStatus, error) {
	xfers := make(map[string]*XferStatus)

	bytes, response, err := d.Get(GetAllTorrents, []string{"", ""})
	if err != nil {
		return bytes, xfers, fmt.Errorf("get(GetAllTorrents): %w", err)
	}

	if err := json.Unmarshal(response.Result, &xfers); err != nil {
		d.logPayload(response.Result)
		return bytes, xfers, fmt.Errorf("json.Unmarshal(xfers): %w", err)
	}

	return bytes, xfers, nil
}

// GetXfersCompat gets all the Transfers from Deluge 1.x or 2.x.
// Depend on what you're actually trying to do, this is likely the best method to use.
// This will return a combined struct hat has data for Deluge 1 and Deluge 2.
// All of the data for either version will be made available with this method.
func (d Deluge) GetXfersCompat() (int64, map[string]*XferStatusCompat, error) {
	xfers := make(map[string]*XferStatusCompat)

	bytes, response, err := d.Get(GetAllTorrents, []string{"", ""})
	if err != nil {
		return bytes, xfers, fmt.Errorf("get(GetAllTorrents): %w", err)
	}

	if err := json.Unmarshal(response.Result, &xfers); err != nil {
		d.logPayload(response.Result)
		return bytes, xfers, fmt.Errorf("json.Unmarshal(xfers): %w", err)
	}

	return bytes, xfers, nil
}

// Get a response from Deluge.
func (d Deluge) Get(method string, params interface{}) (int64, *Response, error) {
	var response Response

	if !d.cookie {
		if size, err := d.Login(); err != nil {
			return size, &response, err
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout.Duration)
	defer cancel()

	req, err := d.DelReq(ctx, method, params)
	if err != nil {
		return 0, &response, fmt.Errorf("d.DelReq: %w", err)
	}

	resp, err := d.Do(req)
	if err != nil {
		d.Client.cookie = false
		return 0, &response, fmt.Errorf("d.Do: %w", err)
	}
	defer resp.Body.Close()

	counter := datacounter.NewReaderCounter(resp.Body)

	if err = json.NewDecoder(counter).Decode(&response); err != nil {
		d.logPayload(response.Result)
		return int64(counter.Count()), &response, fmt.Errorf("json.Unmarshal(response): %w", err)
	}

	if response.Error.Code != 0 {
		return int64(counter.Count()), &response, fmt.Errorf("%w: %s", ErrDelugeError, response.Error.Message)
	}

	return int64(counter.Count()), &response, nil
}

// Log logs a debug message.
func (d *Deluge) Log(msg string, fmt ...interface{}) {
	if d.DebugLog != nil {
		d.DebugLog(msg, fmt...)
	}
}

// logPayload writes a json payload to output. Used for debugging API data.
func (d *Deluge) logPayload(result json.RawMessage) {
	out, err := result.MarshalJSON()

	d.Log("Failed Payload:\n%s\n", string(out))

	if err != nil {
		d.Log("Payload Marshal Error: %v", err)
	}
}
