package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/bwmarrin/discordgo"
)

var createdEmbeds []*discordgo.Message

func main() {
	// auth. to discord
	discord, err := discordgo.New("Bot " + os.Getenv("ASCENSION_MONITOR_TOKEN"))

	discord.AddHandler(PingPong)

	// In this example, we only care about receiving message events.
	discord.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsGuildMessages)

	// Open a websocket connection to Discord and begin listening.
	err = discord.Open()
	if err != nil {
		log.Fatalln("error opening connection,", err)
		return
	}

	// create scheduled tasks to monitor each item in monitors.json
	buildMonitors(discord)

	// Wait here until CTRL-C or other term signal is received.
	log.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// clean up embeds on exit to not clutter the channels
	for _, embed := range createdEmbeds {
		err := discord.ChannelMessageDelete(embed.ChannelID, embed.ID)
		if err != nil {
			log.Printf("Error deleting embed in channel %s: %v", embed.ChannelID, err)
		}
	}

	// Cleanly close down the Discord session.
	discord.Close()
}

func initLogger() {
	f, err := os.OpenFile("text.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalln(err)
	}
	defer f.Close()

	wrt := io.MultiWriter(os.Stdout, f)
	log.SetOutput(wrt)

	log.Println("Initialized log")
}

func buildMonitors(s *discordgo.Session) {
	configFile := os.Getenv("CONFIG_FILE")
	jsonFile, err := os.Open(configFile)
	postToDisord, err := strconv.ParseBool(os.Getenv("POST_TO_DISCORD"))

	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("Successfully Opened %s\n", configFile)

	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	var monitors Monitors
	json.Unmarshal(byteValue, &monitors)

	// Create goroutine for each enabled monitor in configFile
	for i := 0; i < len(monitors.Monitors); i++ {
		monitor := monitors.Monitors[i]

		if monitor.Enabled {
			log.Printf("Checking %s every %d seconds\n", monitor.FriendlyName, monitor.Interval)

			if postToDisord {
				embed, err := s.ChannelMessageSendEmbed(monitor.ChannelID, NewGenericEmbed(monitor.FriendlyName, fmt.Sprintf("Checking [%s](%s) stock every %d seconds\n", monitor.FriendlyName, monitor.URL, monitor.Interval)))
				if err != nil {
					log.Printf("Error creating alert embed for %s", monitor.FriendlyName)
				}
				createdEmbeds = append(createdEmbeds, embed)
			}

			ticker := time.NewTicker(monitor.Interval * time.Second)
			quit := make(chan struct{})

			// check stock every n seconds
			go func() {
				for {
					select {
					case <-ticker.C:
						log.Printf("Checking if %s in stock...\n", monitor.FriendlyName)

						start := time.Now()
						checkStock(monitor, s)
						elapsed := time.Since(start)
						log.Printf("Took %s to check %s stock\n", elapsed, monitor.FriendlyName)
					case <-quit:
						ticker.Stop()
						return
					}
				}
			}()
		}
	}
}

func checkStock(monitor Monitor, discord *discordgo.Session) {
	postToDisord, err := strconv.ParseBool(os.Getenv("POST_TO_DISCORD"))

	client := &http.Client{}

	userAgent := "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:84.0) Gecko/20100101 Firefox/84.0"

	req, err := http.NewRequest("GET", monitor.URL, nil)

	if err != nil {
		log.Println(err)
	}

	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)

	if err != nil {
		log.Println(err)
	}

	defer resp.Body.Close()
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Println(err)
	}

	doc.Find(monitor.Keywords.Selector).Each(func(i int, s *goquery.Selection) {
		buttonText := s.Text()

		if (buttonText == monitor.Keywords.Positive || buttonText == "See Details") && postToDisord {
			successEmbed := buildSuccessEmbed(doc, monitor)

			discord.ChannelMessageSendEmbed(monitor.ChannelID, &successEmbed)
		}
		log.Printf("-----%s-----%s", monitor.FriendlyName, buttonText)

	})
}

func buildSuccessEmbed(doc *goquery.Document, monitor Monitor) discordgo.MessageEmbed {
	var embed discordgo.MessageEmbed

	embed.Type = discordgo.EmbedTypeRich
	embed.URL = monitor.URL
	embed.Title = monitor.FriendlyName

	// build custom embed fields
	var skuValue string
	var priceValue string
	var typeValue string

	skuValue = "SKU Not Found"
	priceValue = "Price Not Found"
	typeValue = "Online"

	if monitor.Keywords.SKUSelector != "" {
		doc.Find(monitor.Keywords.SKUSelector).Each(func(i int, s *goquery.Selection) {
			skuValue = s.Text()
		})
	}

	if monitor.Keywords.PriceSelector != "" {
		doc.Find(monitor.Keywords.PriceSelector).Each(func(i int, s *goquery.Selection) {
			priceValue = s.Text()[strings.Index(s.Text(), "$"):]
		})
	}

	if monitor.Keywords.TypeSelector != "" {
		doc.Find(monitor.Keywords.TypeSelector).Each(func(i int, s *goquery.Selection) {
			typeValue = s.Text()
		})
	}

	priceField := discordgo.MessageEmbedField{
		Name:   "Price",
		Value:  priceValue,
		Inline: true,
	}

	skuField := discordgo.MessageEmbedField{
		Name:   "SKU",
		Value:  skuValue,
		Inline: true,
	}

	typeField := discordgo.MessageEmbedField{
		Name:   "Type",
		Value:  typeValue,
		Inline: true,
	}

	embed.Fields = append(embed.Fields, &priceField)
	embed.Fields = append(embed.Fields, &skuField)
	embed.Fields = append(embed.Fields, &typeField)

	return embed
}
