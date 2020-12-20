package main

import "time"

// Monitor Struct
type Monitor struct {
	URL          string        `json:"url"`
	Interval     time.Duration `json:"interval"`
	FriendlyName string        `json:"friendlyName"`
	Enabled      bool          `json:"enabled"`
	ChannelID    string        `json:"channelID"`
	Keywords     Keywords      `json:"keywords"`
}

// Monitors Struct
type Monitors struct {
	Monitors []Monitor `json:"monitors"`
}

// Keywords struct -- how we dynamically check stock on several sites
type Keywords struct {
	Positive string `json:"positive"`
	Negative string `json:"negative"`
	Selector string `json:"selector"`
}
