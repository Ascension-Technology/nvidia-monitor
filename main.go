package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
)

func main() {
	discord, err := discordgo.New("Bot " + os.Getenv("ASCENSION_MONITOR_TOKEN"))

	discord.AddHandler(messageCreate)

	// In this example, we only care about receiving message events.
	discord.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsGuildMessages)

	// Open a websocket connection to Discord and begin listening.
	err = discord.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	buildMonitors()

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	discord.Close()
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the authenticated bot has access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	if m.Author.ID == s.State.User.ID {
		return
	}
	// If the message is "ping" reply with "Pong!"
	if m.Content == "ping" {
		s.ChannelMessageSend(m.ChannelID, "Pong!")
	}

	// If the message is "pong" reply with "Ping!"
	if m.Content == "pong" {
		s.ChannelMessageSend(m.ChannelID, "Ping!")
	}
}

// Monitor Struct
type Monitor struct {
	URL               string        `json:"url"`
	OutOfStockKeyword string        `json:"outOfStockKeyword"`
	Interval          time.Duration `json:"interval"`
	FriendlyName      string        `json:"friendlyName"`
}

// Monitors Struct
type Monitors struct {
	Monitors []Monitor `json:"monitors"`
}

func buildMonitors() {
	jsonFile, err := os.Open("monitors.json")

	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println("Successfully Opened monitors.json")

	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	var monitors Monitors

	json.Unmarshal(byteValue, &monitors)

	for i := 0; i < len(monitors.Monitors); i++ {
		monitor := monitors.Monitors[i]
		interval := monitor.Interval
		friendlyName := monitor.FriendlyName

		fmt.Printf("Checking %s every %d seconds\n", friendlyName, interval)

		ticker := time.NewTicker(interval * time.Second)
		quit := make(chan struct{})

		// check stock every 10 seconds
		go func() {
			for {
				select {
				case <-ticker.C:
					fmt.Printf("Checking if %s in stock...\n", monitor.FriendlyName)

					start := time.Now()
					checkStock(monitor)
					elapsed := time.Since(start)
					fmt.Printf("Took %s to check %s stock\n", elapsed, monitor.FriendlyName)
				case <-quit:
					ticker.Stop()
					return
				}
			}
		}()

	}
}

func checkStock(monitor Monitor) {
	client := &http.Client{}

	userAgent := "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:84.0) Gecko/20100101 Firefox/84.0"

	req, err := http.NewRequest("GET", monitor.URL, nil)

	if err != nil {
		log.Fatalln(err)
	}

	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)

	if err != nil {
		log.Fatalln(err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	html := string(body)

	if strings.Contains(html, monitor.OutOfStockKeyword) {
		fmt.Printf("%s Out of Stock\n", monitor.FriendlyName)
	} else {
		fmt.Printf("%s IN STOCK!!!!!!!!!\n", monitor.FriendlyName)
	}

}
