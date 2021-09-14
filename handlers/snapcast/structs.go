package snapcast

// Volume rapresents the volume of a device
type Volume struct {
	Muted   bool
	Percent int
}

// Config is the snapcast config
type Config struct {
	Instance int
	Latency  int
	Name     string
	Volume   Volume
}

// Client is a snapcast client
type Client struct {
	Config     Config
	Connected  bool
	Host       interface{}
	ID         string
	LastSeen   interface{}
	Snapclient interface{}
}

type Group struct {
	Clients  []Client `json:"clients" mapstructure:"clients"`
	ID       string   `json:"id" mapstructure:"id"`
	Muted    bool
	Name     string
	StreamID string `json:"stream_id" mapstructure:"stream_id"`
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
	ID     string
	Volume Volume
}
