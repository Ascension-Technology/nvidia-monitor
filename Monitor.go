package main

import "time"

// Monitor Struct
type Monitor struct {
	URL          string        `json:"url"`
	Interval     time.Duration `json:"interval"`
	FriendlyName string        `json:"friendlyName"`
	Enabled      bool          `json:"enabled"`
	ChannelID    string        `json:"channelID"`
}

// Monitors Struct
type Monitors struct {
	Monitors []Monitor `json:"monitors"`
}
