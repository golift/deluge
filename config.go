package deluge

import (
	"encoding/json"
	"time"
)

// Deluge WebUI methods.
const (
	AuthLogin      = "auth.login"
	AddMagnet      = "core.add_torrent_magnet"
	AddTorrentURL  = "core.add_torrent_url"
	AddTorrentFile = "core.add_torrent_file"
	GetTorrentStat = "core.get_torrent_status"
	GetAllTorrents = "core.get_torrents_status"
	HostStatus     = "web.get_host_status"
	GeHosts        = "web.get_hosts"
)

// Config is the data needed to poll Deluge.
type Config struct {
	URL       string   `json:"url" toml:"url" xml:"url" yaml:"url"`
	Password  string   `json:"password" toml:"password" xml:"password" yaml:"password"`
	HTTPPass  string   `json:"http_pass" toml:"http_pass" xml:"http_pass" yaml:"http_pass"`
	HTTPUser  string   `json:"http_user" toml:"http_user" xml:"http_user" yaml:"http_user"`
	Timeout   Duration `json:"timeout" toml:"timeout" xml:"timeout" yaml:"timeout"`
	VerifySSL bool     `json:"verify_ssl" toml:"verify_ssl" xml:"verify_ssl" yaml:"verify_ssl"`
	Version   string   `json:"version" toml:"version" xml:"version" yaml:"version"`
}

// Duration is used to UnmarshalTOML into a time.Duration value.
type Duration struct{ time.Duration }

// UnmarshalText parses a duration type from a config file.
func (d *Duration) UnmarshalText(data []byte) (err error) {
	d.Duration, err = time.ParseDuration(string(data))
	return
}

// Response from Deluge
type Response struct {
	ID     int64           `json:"id"`
	Result json.RawMessage `json:"result"`
	Error  struct {
		Message string `json:"message"`
		Code    int    `json:"code"`
	} `json:"error"`
}

// Backend holds a WebUI's backend server data.
type Backend struct {
	ID   string
	Addr string
	Prot string
}

// XferStatus2 is the Deluge 2.0 WebUI API layout for Active Transfers.
type XferStatus2 struct {
	ActiveTime                float64 `json:"active_time"`
	SeedingTime               float64 `json:"seeding_time"`
	FinishedTime              float64 `json:"finished_time"`
	AllTimeDownload           float64 `json:"all_time_download"`
	StorageMode               string  `json:"storage_mode"`
	DistributedCopies         float64 `json:"distributed_copies"`
	DownloadPayloadRate       float64 `json:"download_payload_rate"`
	FilePriorities            []int   `json:"file_priorities"`
	Hash                      string  `json:"hash"`
	AutoManaged               bool    `json:"auto_managed"`
	IsAutoManaged             bool    `json:"is_auto_managed"`
	IsFinished                bool    `json:"is_finished"`
	MaxConnections            float64 `json:"max_connections"`
	MaxDownloadSpeed          float64 `json:"max_download_speed"`
	MaxUploadSlots            float64 `json:"max_upload_slots"`
	MaxUploadSpeed            float64 `json:"max_upload_speed"`
	Message                   string  `json:"message"`
	MoveOnCompletedPath       string  `json:"move_on_completed_path"`
	MoveOnCompleted           bool    `json:"move_on_completed"`
	MoveCompletedPath         string  `json:"move_completed_path"`
	MoveCompleted             bool    `json:"move_completed"`
	NextAnnounce              float64 `json:"next_announce"`
	NumPeers                  int64   `json:"num_peers"`
	NumSeeds                  int64   `json:"num_seeds"`
	Owner                     string  `json:"owner"`
	Paused                    bool    `json:"paused"`
	PrioritizeFirstLast       bool    `json:"prioritize_first_last"`
	PrioritizeFirstLastPieces bool    `json:"prioritize_first_last_pieces"`
	SequentialDownload        bool    `json:"sequential_download"`
	Progress                  float64 `json:"progress"`
	Shared                    bool    `json:"shared"`
	RemoveAtRatio             bool    `json:"remove_at_ratio"`
	SavePath                  string  `json:"save_path"`
	DownloadLocation          string  `json:"download_location"`
	SeedsPeersRatio           float64 `json:"seeds_peers_ratio"`
	SeedRank                  int     `json:"seed_rank"`
	State                     string  `json:"state"`
	StopAtRatio               bool    `json:"stop_at_ratio"`
	StopRatio                 float64 `json:"stop_ratio"`
	TimeAdded                 float64 `json:"time_added"`
	TotalDone                 float64 `json:"total_done"`
	TotalPayloadDownload      float64 `json:"total_payload_download"`
	TotalPayloadUpload        float64 `json:"total_payload_upload"`
	TotalPeers                int64   `json:"total_peers"`
	TotalSeeds                float64 `json:"total_seeds"`
	TotalUploaded             float64 `json:"total_uploaded"`
	TotalWanted               float64 `json:"total_wanted"`
	TotalRemaining            float64 `json:"total_remaining"`
	Tracker                   string  `json:"tracker"`
	TrackerHost               string  `json:"tracker_host"`
	Trackers                  []struct {
		URL       string `json:"url"`
		Trackerid string `json:"trackerid"`
		Tier      int    `json:"tier"`
		FailLimit int    `json:"fail_limit"`
		Source    int    `json:"source"`
		Verified  bool   `json:"verified"`
		Message   string `json:"message"`
		LastError struct {
			Value    int    `json:"value"`
			Category string `json:"category"`
		} `json:"last_error"`
		NextAnnounce     interface{}   `json:"next_announce"`
		MinAnnounce      interface{}   `json:"min_announce"`
		ScrapeIncomplete float64       `json:"scrape_incomplete"`
		ScrapeComplete   float64       `json:"scrape_complete"`
		ScrapeDownloaded float64       `json:"scrape_downloaded"`
		Fails            int64         `json:"fails"`
		Updating         bool          `json:"updating"`
		StartSent        bool          `json:"start_sent"`
		CompleteSent     bool          `json:"complete_sent"`
		Endpoints        []interface{} `json:"endpoints"`
		SendStats        bool          `json:"send_stats"`
	} `json:"trackers"`
	TrackerStatus     string      `json:"tracker_status"`
	UploadPayloadRate float64     `json:"upload_payload_rate"`
	Comment           string      `json:"comment"`
	Creator           string      `json:"creator"`
	NumFiles          float64     `json:"num_files"`
	NumPieces         float64     `json:"num_pieces"`
	PieceLength       float64     `json:"piece_length"`
	Private           bool        `json:"private"`
	TotalSize         float64     `json:"total_size"`
	Eta               json.Number `json:"eta"`
	FileProgress      []float64   `json:"file_progress"`
	Files             []struct {
		Index  int64  `json:"index"`
		Path   string `json:"path"`
		Size   int64  `json:"size"`
		Offset int64  `json:"offset"`
	} `json:"files"`
	OrigFiles []struct {
		Index  int64  `json:"index"`
		Path   string `json:"path"`
		Size   int64  `json:"size"`
		Offset int64  `json:"offset"`
	} `json:"orig_files"`
	IsSeed            bool          `json:"is_seed"`
	Peers             []interface{} `json:"peers"`
	Queue             int           `json:"queue"`
	Ratio             float64       `json:"ratio"`
	CompletedTime     float64       `json:"completed_time"`
	LastSeenComplete  float64       `json:"last_seen_complete"`
	Name              string        `json:"name"`
	Pieces            interface{}   `json:"pieces"`
	SeedMode          bool          `json:"seed_mode"`
	SuperSeeding      bool          `json:"super_seeding"`
	TimeSinceDownload float64       `json:"time_since_download"`
	TimeSinceUpload   float64       `json:"time_since_upload"`
	TimeSinceTransfer float64       `json:"time_since_transfer"`
}

// XferStatus is the Deluge 1.0 WebUI API layout for Active Transfers.
type XferStatus struct {
	Comment             string  `json:"comment"`
	ActiveTime          int64   `json:"active_time"`
	IsSeed              bool    `json:"is_seed"`
	Hash                string  `json:"hash"`
	UploadPayloadRate   int64   `json:"upload_payload_rate"`
	MoveCompletedPath   string  `json:"move_completed_path"`
	Private             bool    `json:"private"`
	TotalPayloadUpload  int64   `json:"total_payload_upload"`
	Paused              bool    `json:"paused"`
	SeedRank            int64   `json:"seed_rank"`
	SeedingTime         int64   `json:"seeding_time"`
	MaxUploadSlots      int64   `json:"max_upload_slots"`
	PrioritizeFirstLast bool    `json:"prioritize_first_last"`
	DistributedCopies   float64 `json:"distributed_copies"`
	DownloadPayloadRate int64   `json:"download_payload_rate"`
	Message             string  `json:"message"`
	NumPeers            int64   `json:"num_peers"`
	MaxDownloadSpeed    int64   `json:"max_download_speed"`
	MaxConnections      int64   `json:"max_connections"`
	Compact             bool    `json:"compact"`
	Ratio               float64 `json:"ratio"`
	TotalPeers          int64   `json:"total_peers"`
	TotalSize           int64   `json:"total_size"`
	TotalWanted         int64   `json:"total_wanted"`
	State               string  `json:"state"`
	FilePriorities      []int   `json:"file_priorities"`
	Label               string  `json:"label"`
	MaxUploadSpeed      int64   `json:"max_upload_speed"`
	RemoveAtRatio       bool    `json:"remove_at_ratio"`
	Tracker             string  `json:"tracker"`
	SavePath            string  `json:"save_path"`
	Progress            float64 `json:"progress"`
	TimeAdded           float64 `json:"time_added"`
	TrackerHost         string  `json:"tracker_host"`
	TotalUploaded       int64   `json:"total_uploaded"`
	Files               []struct {
		Index  int64  `json:"index"`
		Path   string `json:"path"`
		Offset int64  `json:"offset"`
		Size   int64  `json:"size"`
	} `json:"files"`
	TotalDone           int64         `json:"total_done"`
	NumPieces           int64         `json:"num_pieces"`
	TrackerStatus       string        `json:"tracker_status"`
	TotalSeeds          int64         `json:"total_seeds"`
	MoveOnCompleted     bool          `json:"move_on_completed"`
	NextAnnounce        int64         `json:"next_announce"`
	StopAtRatio         bool          `json:"stop_at_ratio"`
	FileProgress        []float64     `json:"file_progress"`
	MoveCompleted       bool          `json:"move_completed"`
	PieceLength         int64         `json:"piece_length"`
	AllTimeDownload     int64         `json:"all_time_download"`
	MoveOnCompletedPath string        `json:"move_on_completed_path"`
	NumSeeds            int64         `json:"num_seeds"`
	Peers               []interface{} `json:"peers"`
	Name                string        `json:"name"`
	Trackers            []struct {
		SendStats    bool        `json:"send_stats"`
		Fails        int64       `json:"fails"`
		Verified     bool        `json:"verified"`
		MinAnnounce  interface{} `json:"min_announce"`
		URL          string      `json:"url"`
		FailLimit    int64       `json:"fail_limit"`
		NextAnnounce interface{} `json:"next_announce"`
		CompleteSent bool        `json:"complete_sent"`
		Source       int64       `json:"source"`
		StartSent    bool        `json:"start_sent"`
		Tier         int64       `json:"tier"`
		Updating     bool        `json:"updating"`
	} `json:"trackers"`
	TotalPayloadDownload int64       `json:"total_payload_download"`
	IsAutoManaged        bool        `json:"is_auto_managed"`
	SeedsPeersRatio      float64     `json:"seeds_peers_ratio"`
	Queue                int64       `json:"queue"`
	NumFiles             int64       `json:"num_files"`
	Eta                  json.Number `json:"eta"`
	StopRatio            float64     `json:"stop_ratio"`
	IsFinished           bool        `json:"is_finished"`
}

// XferStatusCompat is a compatibile struct for Deluge 1 and 2 API data.
type XferStatusCompat struct {
	ActiveTime                float64     `json:"active_time"`
	SeedingTime               float64     `json:"seeding_time"`
	FinishedTime              float64     `json:"finished_time"`
	AllTimeDownload           float64     `json:"all_time_download"`
	StorageMode               string      `json:"storage_mode"`
	DistributedCopies         float64     `json:"distributed_copies"`
	DownloadPayloadRate       float64     `json:"download_payload_rate"`
	FilePriorities            []int       `json:"file_priorities"`
	Hash                      string      `json:"hash"`
	AutoManaged               bool        `json:"auto_managed"`
	IsAutoManaged             bool        `json:"is_auto_managed"`
	IsFinished                bool        `json:"is_finished"`
	MaxConnections            float64     `json:"max_connections"`
	MaxDownloadSpeed          float64     `json:"max_download_speed"`
	MaxUploadSlots            float64     `json:"max_upload_slots"`
	MaxUploadSpeed            float64     `json:"max_upload_speed"`
	Message                   string      `json:"message"`
	MoveOnCompletedPath       string      `json:"move_on_completed_path"`
	MoveOnCompleted           bool        `json:"move_on_completed"`
	MoveCompletedPath         string      `json:"move_completed_path"`
	MoveCompleted             bool        `json:"move_completed"`
	NextAnnounce              float64     `json:"next_announce"`
	NumPeers                  int64       `json:"num_peers"`
	NumSeeds                  int64       `json:"num_seeds"`
	Owner                     string      `json:"owner"`
	Paused                    bool        `json:"paused"`
	PrioritizeFirstLast       bool        `json:"prioritize_first_last"`
	PrioritizeFirstLastPieces bool        `json:"prioritize_first_last_pieces"`
	SequentialDownload        bool        `json:"sequential_download"`
	Progress                  float64     `json:"progress"`
	Shared                    bool        `json:"shared"`
	RemoveAtRatio             bool        `json:"remove_at_ratio"`
	SavePath                  string      `json:"save_path"`
	DownloadLocation          string      `json:"download_location"`
	SeedsPeersRatio           float64     `json:"seeds_peers_ratio"`
	SeedRank                  int         `json:"seed_rank"`
	State                     string      `json:"state"`
	StopAtRatio               bool        `json:"stop_at_ratio"`
	StopRatio                 float64     `json:"stop_ratio"`
	TimeAdded                 float64     `json:"time_added"`
	TotalDone                 float64     `json:"total_done"`
	TotalPayloadDownload      float64     `json:"total_payload_download"`
	TotalPayloadUpload        float64     `json:"total_payload_upload"`
	TotalPeers                int64       `json:"total_peers"`
	TotalSeeds                float64     `json:"total_seeds"`
	TotalUploaded             float64     `json:"total_uploaded"`
	TotalWanted               float64     `json:"total_wanted"`
	TotalRemaining            float64     `json:"total_remaining"`
	Tracker                   string      `json:"tracker"`
	TrackerHost               string      `json:"tracker_host"`
	TrackerStatus             string      `json:"tracker_status"`
	UploadPayloadRate         float64     `json:"upload_payload_rate"`
	Comment                   string      `json:"comment"`
	Creator                   string      `json:"creator"`
	NumFiles                  float64     `json:"num_files"`
	NumPieces                 float64     `json:"num_pieces"`
	PieceLength               float64     `json:"piece_length"`
	Private                   bool        `json:"private"`
	TotalSize                 float64     `json:"total_size"`
	Eta                       json.Number `json:"eta"`
	FileProgress              []float64   `json:"file_progress"`
	Files                     []struct {
		Index  int64  `json:"index"`
		Path   string `json:"path"`
		Size   int64  `json:"size"`
		Offset int64  `json:"offset"`
	} `json:"files"`
	OrigFiles []struct {
		Index  int64  `json:"index"`
		Path   string `json:"path"`
		Size   int64  `json:"size"`
		Offset int64  `json:"offset"`
	} `json:"orig_files"`
	IsSeed            bool          `json:"is_seed"`
	Peers             []interface{} `json:"peers"`
	Queue             int64         `json:"queue"`
	Ratio             float64       `json:"ratio"`
	CompletedTime     float64       `json:"completed_time"`
	LastSeenComplete  float64       `json:"last_seen_complete"`
	Name              string        `json:"name"`
	Pieces            interface{}   `json:"pieces"`
	SeedMode          bool          `json:"seed_mode"`
	SuperSeeding      bool          `json:"super_seeding"`
	TimeSinceDownload float64       `json:"time_since_download"`
	TimeSinceUpload   float64       `json:"time_since_upload"`
	TimeSinceTransfer float64       `json:"time_since_transfer"`
	Label             string        `json:"label"`
	Trackers          []struct {
		SendStats bool    `json:"send_stats"`
		Source    float64 `json:"source"`
		StartSent bool    `json:"start_sent"`
		URL       string  `json:"url"`
		Trackerid string  `json:"trackerid"`
		Tier      float64 `json:"tier"`
		FailLimit int64   `json:"fail_limit"`
		Verified  bool    `json:"verified"`
		Message   string  `json:"message"`
		LastError struct {
			Value    int    `json:"value"`
			Category string `json:"category"`
		} `json:"last_error"`
		NextAnnounce     interface{}   `json:"next_announce"`
		MinAnnounce      interface{}   `json:"min_announce"`
		ScrapeIncomplete float64       `json:"scrape_incomplete"`
		ScrapeComplete   float64       `json:"scrape_complete"`
		ScrapeDownloaded float64       `json:"scrape_downloaded"`
		Fails            int64         `json:"fails"`
		Updating         bool          `json:"updating"`
		CompleteSent     bool          `json:"complete_sent"`
		Endpoints        []interface{} `json:"endpoints"`
	} `json:"trackers"`
}
