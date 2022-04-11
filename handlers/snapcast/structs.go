package snapcast

// Volume rapresents the volume of a device
type Volume struct {
	Muted   bool `json:"muted" mapstructure:"muted"`
	Percent int  `json:"percent" mapstructure:"percent"`
}

// Config is the snapcast config
type Config struct {
	Instance int    `json:"instance" mapstructure:"instance"`
	Latency  int    `json:"latency" mapstructure:"latency"`
	Name     string `json:"name" mapstructure:"name"`
	Volume   Volume `json:"volume" mapstructure:"volume"`
}

// Client is a snapcast client
type Client struct {
	Config     Config      `json:"config" mapstructure:"config"`
	Connected  bool        `json:"connected" mapstructure:"connected"`
	Host       interface{} `json:"host" mapstructure:"host"`
	ID         string      `json:"id" mapstructure:"id"`
	LastSeen   interface{}
	Snapclient interface{}
}

type Group struct {
	Clients  []Client `json:"clients" mapstructure:"clients"`
	ID       string   `json:"id" mapstructure:"id"`
	Muted    bool     `json:"muted" mapstructure:"muted"`
	Name     string   `json:"name" mapstructure:"name"`
	StreamID string   `json:"stream_id" mapstructure:"stream_id"`
}

type GroupStreamChanged struct {
	ID       string `json:"id,omitempty" mapstructure:"id"`
	StreamID string `json:"stream_id" mapstructure:"stream_id"`
}

type URL struct {
	Raw string
}

type Stream struct {
	ID     string
	Status string
	URI    URL
}

type ServerStatus struct {
	Groups  []Group
	Streams []Stream
}

type Status struct {
	Server ServerStatus
}

type ClientVolume struct {
	ID     string `json:"id" mapstructure:"id"`
	Volume Volume `json:"volume" mapstructure:"volume"`
}

type SimplifiedClient struct {
	ID string `json:"id" mapstructure:"id"`
}

type SimplifiedGroup struct {
	Clients  []SimplifiedClient `json:"clients" mapstructure:"clients"`
	ID       string             `json:"id" mapstructure:"id"`
	Name     string             `json:"name" mapstructure:"name"`
	StreamID string             `json:"stream_id" mapstructure:"stream_id"`
}

type SimplifiedServerStatus struct {
	Groups  []SimplifiedGroup
	Streams []Stream
}

type SimplifiedStatus struct {
	Server SimplifiedServerStatus
}
