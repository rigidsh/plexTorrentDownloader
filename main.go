package main

import (
	"fmt"
	"github.com/caarlos0/env/v11"
	"net/url"
	"os"
	"plexTorrentDownloader/lostfilm"
	"plexTorrentDownloader/torrent"
	"strconv"
	"strings"
	"time"
)

var lostFilmClient *lostfilm.Client
var torrentClient torrent.Client
var interestContent = map[string]bool{}

type Config struct {
	LostFilmToken        string        `env:"LOST_FILM_TOKEN"`
	TransmissionProtocol string        `env:"TRANSMISSION_PROTOCOL" envDefault:"http"`
	TransmissionPort     int           `env:"TRANSMISSION_PORT" envDefault:"9091"`
	TransmissionHost     string        `env:"TRANSMISSION_HOST"`
	TransmissionUsername string        `env:"TRANSMISSION_USERNAME"`
	TransmissionPassword string        `env:"TRANSMISSION_PASSWORD"`
	TransmissionRPCPath  string        `env:"TRANSMISSION_RPC_PATH" envDefault:"/transmission/rpc"`
	InterestContent      []string      `env:"INTEREST_CONTENT"`
	DownloadPath         string        `env:"DOWNLOAD_PATH" envDefault:"/downloads/complete/tvShows"`
	DownloadQuality      string        `env:"DOWNLOAD_QUALITY" envDefault:"FullHD"`
	JobInterval          time.Duration `env:"JOB_INTERVAL_MINUTES" envDefault:"10m"`
}

var config Config

func main() {
	err := env.Parse(&config)

	if err != nil {
		fmt.Println("Error parsing config:", err)
		return
	}

	for _, contentName := range config.InterestContent {
		interestContent[strings.ToLower(contentName)] = true

	}

	lostFilmClient, err = lostfilm.NewClient(config.LostFilmToken)

	if err != nil {
		fmt.Println(err)
		return
	}

	rpcUrl, err := url.ParseRequestURI(fmt.Sprintf("%s://%s:%s@%s:%d%s",
		config.TransmissionProtocol,
		config.TransmissionUsername,
		config.TransmissionPassword,
		config.TransmissionHost,
		config.TransmissionPort,
		config.TransmissionRPCPath,
	))
	if err != nil {
		fmt.Println(err)
		return
	}

	torrentClient, err = torrent.NewTransmissionRemoteTorrent(rpcUrl)
	if err != nil {
		fmt.Println(err)
		return
	}

	for {
		fmt.Println("Start new search")
		checkNewDownloads()
		fmt.Println("Search completed")
		time.Sleep(config.JobInterval)
	}

}

func checkNewDownloads() {
	checkAfter := getLastContentDate()
	items, err := lostFilmClient.GetNewItems()
	if err != nil {
		fmt.Println(err)
		return
	}

	newLastContentDate := checkAfter

	for _, item := range items {
		if item.PublicationDate.After(checkAfter) {
			fmt.Printf("Find new item %s (S%dE%d)...", item.OriginalName, item.Season, item.Episode)

			if item.PublicationDate.After(newLastContentDate) {
				newLastContentDate = item.PublicationDate
			}

			if value, ok := interestContent[strings.ToLower(item.OriginalName)]; value && ok {
				fmt.Println("Interesting. Add to download")
				err := item.Download(torrentClient, lostfilm.VideoQualityFromString(config.DownloadQuality), config.DownloadPath)
				if err != nil {
					fmt.Printf("Error: can't add to downloads: %s\n", err)
				}
			} else {
				fmt.Println("Skip")
			}
		}
	}

	updateLastContentDate(newLastContentDate)

}

var lastContentDate = time.UnixMicro(0) //time.Now()

func getLastContentDate() time.Time {
	data, err := os.ReadFile("lastUpdateMarker")
	if err == nil {
		value, err := strconv.ParseInt(string(data), 10, 0)
		if err == nil {
			lastContentDate = time.UnixMicro(value)
		}
	}
	return lastContentDate
}

func updateLastContentDate(newDate time.Time) {
	lastContentDate = newDate
	err := os.WriteFile("lastUpdateMarker", []byte(strconv.FormatInt(lastContentDate.UnixMicro(), 10)), 0644)
	if err != nil {
		fmt.Println("Warning: Failed to update last content date")
	}
}
