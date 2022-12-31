package deluge

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"strconv"
	"strings"

	"golang.org/x/net/publicsuffix"
)

// Custom errors.
var (
	ErrInvalidVersion = fmt.Errorf("invalid data returned while checking version")
	ErrDelugeError    = fmt.Errorf("deluge error")
	ErrAuthFailed     = fmt.Errorf("authentication failed")
)

// Deluge is what you get for providing a password.
// Version and Backends are only filled if you call New().
type Deluge struct {
	password string
	url      string
	auth     string
	id       int
	client   *http.Client
	Version  string             // Currently unused, for display purposes only.
	Backends map[string]Backend // Currently unused, for display purposes only.
}

// NewNoAuth returns a Deluge object without authenticating or trying to connect.
func NewNoAuth(config *Config) (*Deluge, error) {
	return newConfig(context.TODO(), config, false)
}

// New creates a http.Client with authenticated cookies.
// Used to make additional, authenticated requests to the APIs.
func New(ctx context.Context, config *Config) (*Deluge, error) {
	return newConfig(ctx, config, true)
}

func newConfig(ctx context.Context, config *Config, login bool) (*Deluge, error) {
	// The cookie jar is used to auth Deluge.
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return nil, fmt.Errorf("cookiejar.New(publicsuffix): %w", err)
	}

	delugeURL := strings.TrimSuffix(strings.TrimSuffix(config.URL, "/json"), "/") + "/json"

	// This app allows http auth, in addition to deluge web password.
	auth := config.HTTPUser + ":" + config.HTTPPass
	if auth != ":" {
		auth = "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
	} else {
		auth = ""
	}

	httpClient := config.Client
	if httpClient == nil {
		httpClient = &http.Client{}
	}

	httpClient.Jar = jar

	deluge := &Deluge{
		auth:     auth,
		Backends: make(map[string]Backend),
		password: config.Password,
		url:      delugeURL,
		client:   httpClient,
	}

	if !login {
		return deluge, nil
	}

	if err := deluge.LoginContext(ctx); err != nil {
		return deluge, err
	}

	if deluge.Version = config.Version; deluge.Version == "" {
		if err = deluge.setVersion(ctx); err != nil {
			return deluge, err
		}
	}

	return deluge, nil
}

// Login sets the cookie jar with authentication information.
func (d *Deluge) Login() error {
	return d.LoginContext(context.Background())
}

// LoginContext sets the cookie jar with authentication information.
func (d *Deluge) LoginContext(ctx context.Context) error {
	// This line is how you send auth creds.
	req, err := d.DelReq(ctx, AuthLogin, []string{d.password})
	if err != nil {
		return fmt.Errorf("DelReq(AuthLogin, json): %w", err)
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("d.Do(req): %w", err)
	}
	defer resp.Body.Close()

	_, _ = io.Copy(io.Discard, resp.Body) // must read body to avoid memory leak.

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: %v[%v] (status: %v/%v)",
			ErrAuthFailed, req.URL.String(), AuthLogin, resp.StatusCode, resp.Status)
	}

	return nil
}

// setVersion digs into the first server in the web UI to find the version.
func (d *Deluge) setVersion(ctx context.Context) error {
	response, err := d.Get(ctx, GeHosts, []string{})
	if err != nil {
		return err
	}

	// This method returns a "mixed list" which requires an interface.
	// Deluge devs apparently hate Go. :(
	servers := make([][]interface{}, 0)
	if err := json.Unmarshal(response.Result, &servers); err != nil {
		return fmt.Errorf("json.Unmarshal(rawResult1): %w", err)
	}

	serverID := ""

	// Store each server info (so consumers can access them easily).
	for _, server := range servers {
		serverID, _ = server[0].(string)
		backend := Backend{ID: serverID}
		backend.Addr, _ = server[1].(string)
		val, _ := server[2].(float64)
		backend.Addr += ":" + strconv.FormatFloat(val, 'f', 0, 64) //nolint:gomnd,nolintlint
		backend.Prot, _ = server[3].(string)
		d.Backends[serverID] = backend
	}

	// Store the last server's version as "the version"
	response, err = d.Get(ctx, HostStatus, []string{serverID})
	if err != nil {
		return err
	}

	server := make([]interface{}, 0)
	if err = json.Unmarshal(response.Result, &server); err != nil {
		return fmt.Errorf("json.Unmarshal(rawResult2): %w", err)
	}

	const payloadSegments = 3

	if len(server) < payloadSegments {
		return ErrInvalidVersion
	}

	// Version comes last in the mixed list.
	var ok bool
	if d.Version, ok = server[len(server)-1].(string); !ok {
		return ErrInvalidVersion
	}

	return nil
}

// DelReq is a small helper function that adds headers and marshals the json.
func (d Deluge) DelReq(ctx context.Context, method string, params interface{}) (*http.Request, error) {
	d.id++

	paramMap := map[string]interface{}{"method": method, "id": d.id, "params": params}

	data, err := json.Marshal(paramMap)
	if err != nil {
		return nil, fmt.Errorf("json.Marshal(params): %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, d.url, bytes.NewBuffer(data))
	if err != nil {
		return req, fmt.Errorf("creating request: %w", err)
	}

	if d.auth != "" {
		// In case Deluge is also behind HTTP auth.
		req.Header.Add("Authorization", d.auth)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")

	return req, nil
}

// GetXfers gets all the Transfers from Deluge.
func (d *Deluge) GetXfers() (map[string]*XferStatus, error) {
	return d.GetXfersContext(context.Background())
}

func (d *Deluge) GetXfersContext(ctx context.Context) (map[string]*XferStatus, error) {
	xfers := make(map[string]*XferStatus)

	response, err := d.Get(ctx, GetAllTorrents, []string{"", ""})
	if err != nil {
		return nil, fmt.Errorf("get(GetAllTorrents): %w", err)
	}

	if err := json.Unmarshal(response.Result, &xfers); err != nil {
		return nil, fmt.Errorf("json.Unmarshal(xfers): %w", err)
	}

	return xfers, nil
}

// GetXfersCompat gets all the Transfers from Deluge 1.x or 2.x.
// Depend on what you're actually trying to do, this is likely the best method to use.
// This will return a combined struct hat has data for Deluge 1 and Deluge 2.
// All of the data for either version will be made available with this method.
func (d *Deluge) GetXfersCompat() (map[string]*XferStatusCompat, error) {
	return d.GetXfersCompatContext(context.Background())
}

func (d *Deluge) GetXfersCompatContext(ctx context.Context) (map[string]*XferStatusCompat, error) {
	xfers := make(map[string]*XferStatusCompat)

	response, err := d.Get(ctx, GetAllTorrents, []string{"", ""})
	if err != nil {
		return nil, fmt.Errorf("get(GetAllTorrents): %w", err)
	}

	if err := json.Unmarshal(response.Result, &xfers); err != nil {
		return nil, fmt.Errorf("json.Unmarshal(xfers): %w", err)
	}

	return xfers, nil
}

// Get a response from Deluge.
func (d *Deluge) Get(ctx context.Context, method string, params interface{}) (*Response, error) {
	return d.req(ctx, method, params, true)
}

func (d *Deluge) req(ctx context.Context, method string, params interface{}, loop bool) (*Response, error) {
	req, err := d.DelReq(ctx, method, params)
	if err != nil {
		return nil, fmt.Errorf("d.DelReq: %w", err)
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("d.Do: %w", err)
	}
	defer resp.Body.Close()

	var response Response
	if err = json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("json.Unmarshal(response): %w", err)
	}

	if response.Error.Code != 0 {
		if err := d.LoginContext(ctx); err != nil {
			return nil, err
		}

		if loop {
			return d.req(ctx, method, params, false)
		}

		return &response, fmt.Errorf("%w: %s", ErrDelugeError, response.Error.Message)
	}

	return &response, nil
}
