package main

// Config represents the configuration of the app.
type Config struct {
	Port int32 `json:"port"`
}

// NewConfig gets a new config from command line arguments.
func NewConfig() (Config, error) {
	return Config{
		Port: 57120,
	}, nil
}
