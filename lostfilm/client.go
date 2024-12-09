package lostfilm

import (
	"errors"
	"github.com/PuerkitoBio/goquery"
	"github.com/mmcdole/gofeed"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"plexTorrentDownloader/torrent"
	"strconv"
	"strings"
	"time"
)

type VideoQuality int32

const (
	SD VideoQuality = iota
	HD
	FullHD
	Unknown
)

func VideoQualityFromString(value string) VideoQuality {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "sd":
		return SD
	case "hd":
		return HD
	case "fullhd":
		return FullHD
	}

	return Unknown
}

func parseVideoQuality(label string) VideoQuality {
	switch strings.ToLower(strings.TrimSpace(label)) {
	case "sd":
		return SD
	case "mp4":
		return HD
	case "1080":
		return FullHD
	}

	return Unknown
}

type ContentItem struct {
	Name         string
	OriginalName string
	EpisodeName  string
	Episode      uint8
	Season       uint8

	PublicationDate time.Time

	client     *Client
	contentUrl string
}

func (content ContentItem) Download(torrentClient torrent.Client, quality VideoQuality, path string) error {
	lostFilmContentId, err := content.client.getLostFilmContentId(content.contentUrl)
	if err != nil {
		return err
	}
	link, err := content.client.getInsearchLink(lostFilmContentId)
	if err != nil {
		return err
	}
	torrentFile, err := content.client.getTorrentFileFromInsearchLink(link, quality)
	if err != nil {
		return err
	}

	return torrentClient.AddTorrent(torrentFile, path)
}

func (client *Client) getLostFilmContentId(url string) (int, error) {
	resp, err := client.httpClient.Get(url)
	if err != nil {
		return 0, err
	}

	defer resp.Body.Close()
	page, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return 0, err
	}
	onClickValue, exist := page.Find("div.external-btn").Attr("onclick")
	if !exist {
		return 0, errors.New("no link with lostFilm content Id")
	}

	return strconv.Atoi(onClickValue[len("PlayEpisode('"):(len(onClickValue) - 2)])
}

func (client *Client) getTorrentFileFromInsearchLink(insearchLink string, requestedQuality VideoQuality) ([]byte, error) {
	resp, err := client.httpClient.Get(insearchLink)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	page, err := goquery.NewDocumentFromReader(resp.Body)

	if err != nil {
		return nil, err
	}

	links := page.Find("div.inner-box--list>div.inner-box--item")

	torrentLink, exist := links.FilterFunction(func(_ int, torrentSource *goquery.Selection) bool {
		quality := parseVideoQuality(torrentSource.Find("div.inner-box--label").Text())

		return quality == requestedQuality
	}).Find("div.inner-box--link.main>a").Attr("href")

	if !exist {
		return nil, errors.New("no torrent file link")
	}

	torrentFileResponse, err := client.httpClient.Get(torrentLink)

	if err != nil {
		return nil, err
	}

	defer torrentFileResponse.Body.Close()

	return io.ReadAll(torrentFileResponse.Body)
}

func NewClient(sessionId string) (*Client, error) {
	sessionCookie := &http.Cookie{
		Name:   "lf_session",
		Value:  sessionId,
		Path:   "/",
		Domain: ".lostfilm.tv",
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	tmp, err := url.Parse("https://www.lostfilm.tv/v_search.php")
	if err != nil {
		return nil, err
	}

	jar.SetCookies(tmp, []*http.Cookie{sessionCookie})

	client := &http.Client{
		Jar: jar,
	}

	return &Client{
		client,
	}, nil
}

type Client struct {
	httpClient *http.Client
}

func (client *Client) getInsearchLink(lostFilmContentId int) (string, error) {
	resp, err := client.httpClient.Get("https://www.lostfilm.tv/v_search.php?a=" + strconv.Itoa(lostFilmContentId))

	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	page, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", err
	}
	link, exist := page.Find("a").Attr("href")
	if !exist {
		return "", errors.New("no insearch link")
	}
	return link, nil
}

func (client *Client) GetNewItems() ([]ContentItem, error) {
	fp := gofeed.NewParser()
	feed, err := fp.ParseURL("https://www.lostfilm.tv/rss.xml")
	if err != nil {
		return nil, err
	}

	result := make([]ContentItem, 0)

	for _, item := range feed.Items {
		contentItem, err := parseTitle(item.Title)

		if err != nil {
			continue
		}

		contentItem.client = client
		contentItem.PublicationDate = *item.PublishedParsed
		contentItem.contentUrl = strings.Replace(item.Link, "mr/", "", 1)

		result = append(result, contentItem)
	}

	return result, nil
}
