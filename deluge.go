package deluge

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// Deluge is what you get for providing a password.
type Deluge struct {
	*http.Client
	URL      string
	auth     string
	id       int
	Version  string             // Currently unused, for display purposes only.
	Backends map[string]Backend // Currently unused, for display purposes only.
	DebugLog func(msg string, fmt ...interface{})
}

// New creates a http.Client with authenticated cookies.
// Used to make additional, authenticated requests to the APIs.
func New(config Config) (*Deluge, error) {
	// The cookie jar is used to auth Deluge.
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, errors.Wrap(err, "cookiejar.New(nil)")
	}
	if !strings.HasSuffix(config.URL, "/") {
		config.URL += "/"
	}
	config.URL += "json"

	// This app allows http auth, in addition to deluge web password.
	if both := config.HTTPUser + ":" + config.HTTPPass; both != ":" {
		config.HTTPUser = "Basic " + base64.StdEncoding.EncodeToString([]byte(both))
	} else {
		config.HTTPUser = ""
	}

	deluge := &Deluge{
		URL:      config.URL,
		auth:     config.HTTPUser,
		Backends: make(map[string]Backend, 0),
		Client: &http.Client{
			Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: config.VerifySSL}},
			Jar:       jar,
			Timeout:   config.Timeout.Round(time.Millisecond),
		},
	}
	if err := deluge.Login(config.Password); err != nil {
		return deluge, err
	}
	if deluge.Version = config.Version; deluge.Version == "" {
		if err := deluge.setVersion(); err != nil {
			return deluge, err
		}
	}
	return deluge, nil
}

// Login sets the cookie jar with authentication information.
func (d *Deluge) Login(password string) error {
	// This []string{config.Password} line is how you send auth creds. It's weird.
	req, err := d.DelReq(AuthLogin, []string{password})
	if err != nil {
		return errors.Wrap(err, "DelReq(AuthLogin, json)")
	}
	resp, err := d.Do(req)
	if err != nil {
		return errors.Wrap(err, "d.Do(req)")
	}
	defer func() {
		_, _ = io.Copy(ioutil.Discard, resp.Body)
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("authentication failed: %v[%v] (status: %v/%v)",
			req.URL.String(), AuthLogin, resp.StatusCode, resp.Status)
	}
	return nil
}

// setVersion digs into the first server in the web UI to find the version.
// This is currently unused in this libyrar, and provided for display only.
func (d *Deluge) setVersion() error {
	response, err := d.Get(GeHosts, []string{})
	if err != nil {
		return err
	}
	// This method returns a "mixed list" which requires an interface.
	// Deluge devs apparently hate Go. :(
	servers := make([][]interface{}, 0)
	if err := json.Unmarshal(response.Result, &servers); err != nil {
		d.logPayload(response.Result)
		return errors.Wrap(err, "json.Unmarshal(rawResult1)")
	}

	// Store each server info (so consumers can access them easily).
	serverID := ""
	for _, server := range servers {
		serverID = server[0].(string)
		d.Backends[serverID] = Backend{
			ID:   serverID,
			Addr: server[1].(string) + ":" + strconv.FormatFloat(server[2].(float64), 'f', 0, 64),
			Prot: server[3].(string),
		}
	}

	// Store the last server's version as "the version"
	response, err = d.Get(HostStatus, []string{serverID})
	if err != nil {
		return err
	}
	server := make([]string, 0)
	if err = json.Unmarshal(response.Result, &server); err != nil {
		d.logPayload(response.Result)
		return errors.Wrap(err, "json.Unmarshal(rawResult2)")
	}
	if len(server) != 3 {
		return errors.Errorf("invalid data returned while checking version")
	}
	d.Version = server[2]
	return nil
}

// DelReq is a small helper function that adds headers and marshals the json.
func (d Deluge) DelReq(method string, params interface{}) (req *http.Request, err error) {
	d.id++
	paramMap := map[string]interface{}{"method": method, "id": d.id, "params": params}
	if data, errr := json.Marshal(paramMap); errr != nil {
		return req, errors.Wrap(errr, "json.Marshal(params)")
	} else if req, err = http.NewRequest("POST", d.URL, bytes.NewBuffer(data)); err == nil {
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
func (d Deluge) GetXfers() (map[string]*XferStatus, error) {
	xfers := make(map[string]*XferStatus)
	if response, err := d.Get(GetAllTorrents, []string{"", ""}); err != nil {
		return xfers, errors.Wrap(err, "get(GetAllTorrents)")
	} else if err := json.Unmarshal(response.Result, &xfers); err != nil {
		d.logPayload(response.Result)
		return xfers, errors.Wrap(err, "json.Unmarshal(xfers)")
	}
	return xfers, nil
}

// GetXfersCompat gets all the Transfers from Deluge 1.x or 2.x.
// Depend on what you're actually trying to do, this is likely the best method to use.
// This will return a combined struct hat has data for Deluge 1 and Deluge 2.
// All of the data for either version will be made available with this method.
func (d Deluge) GetXfersCompat() (map[string]*XferStatusCompat, error) {
	xfers := make(map[string]*XferStatusCompat)
	response, err := d.Get(GetAllTorrents, []string{"", ""})
	if err != nil {
		return xfers, errors.Wrap(err, "get(GetAllTorrents)")
	}
	if err := json.Unmarshal(response.Result, &xfers); err != nil {
		d.logPayload(response.Result)
		return xfers, errors.Wrap(err, "json.Unmarshal(xfers)")
	}
	return xfers, nil
}

// Get a response from Deluge
func (d Deluge) Get(method string, params interface{}) (*Response, error) {
	response := new(Response)
	req, err := d.DelReq(method, params)
	if err != nil {
		return response, errors.Wrap(err, "d.DelReq")
	}
	resp, err := d.Do(req)
	if err != nil {
		return response, errors.Wrap(err, "d.Do")
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return response, errors.Wrap(err, "ioutil.ReadAll")
	}
	if err = json.Unmarshal(body, &response); err != nil {
		d.logPayload(response.Result)
		return response, errors.Wrap(err, "json.Unmarshal(response)")
	}
	if response.Error.Code != 0 {
		return response, errors.New("deluge error: " + response.Error.Message)
	}
	return response, nil
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
