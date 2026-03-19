package feed

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"time"

	"github.com/svdx9/go-podcaster/internal/episode/repository"
)

type Channel struct {
	XMLName     xml.Name     `xml:"channel"`
	Title       string       `xml:"title"`
	Description string       `xml:"description"`
	Link        string       `xml:"link"`
	Language    string       `xml:"language"`
	Category    string       `xml:"itunes:category,omitempty"`
	Image       *ItunesImage `xml:"itunes:image,omitempty"`
	Author      string       `xml:"itunes:author,omitempty"`
	Items       []Item       `xml:"item"`
}

type Item struct {
	XMLName     xml.Name  `xml:"item"`
	Title       string    `xml:"title"`
	Description string    `xml:"description"`
	PubDate     string    `xml:"pubDate"`
	Author      string    `xml:"itunes:author,omitempty"`
	Duration    string    `xml:"itunes:duration,omitempty"`
	Enclosure   Enclosure `xml:"enclosure"`
}

type Enclosure struct {
	URL    string `xml:"url,attr"`
	Length int64  `xml:"length,attr"`
	Type   string `xml:"type,attr"`
}

type ItunesImage struct {
	Href string `xml:"href,attr"`
}

type Generator struct {
	repo               repository.Repository
	baseURL            string
	podcastTitle       string
	podcastDescription string
	podcastAuthor      string
	podcastLanguage    string
	podcastCategory    string
	podcastImageURL    string
}

func New(repo repository.Repository, baseURL, title, description, author, language, category, imageURL string) *Generator {
	return &Generator{
		repo:               repo,
		baseURL:            baseURL,
		podcastTitle:       title,
		podcastDescription: description,
		podcastAuthor:      author,
		podcastLanguage:    language,
		podcastCategory:    category,
		podcastImageURL:    imageURL,
	}
}

func (g *Generator) Render(ctx context.Context, w io.Writer) error {
	episodes, err := g.repo.ListAll(ctx)
	if err != nil {
		return fmt.Errorf("failed to list episodes: %w", err)
	}

	items := make([]Item, len(episodes))
	for i, ep := range episodes {
		items[i] = g.episodeToItem(ep)
	}

	channel := Channel{
		XMLName:     xml.Name{Local: "channel", Space: ""},
		Title:       g.podcastTitle,
		Description: g.podcastDescription,
		Link:        g.baseURL,
		Language:    g.podcastLanguage,
		Author:      g.podcastAuthor,
		Category:    g.podcastCategory,
		Items:       items,
		Image:       &ItunesImage{Href: g.podcastImageURL},
	}

	var image *ItunesImage
	if g.podcastImageURL != "" {
		image = &ItunesImage{Href: g.podcastImageURL}
	}
	channel.Image = image

	feed := struct {
		XMLName     xml.Name `xml:"rss"`
		Version     string   `xml:"version,attr"`
		XmlnsItunes string   `xml:"xmlns:itunes,attr"`
		Channel     Channel  `xml:"channel"`
	}{
		XMLName:     xml.Name{Local: "rss", Space: ""},
		Version:     "2.0",
		XmlnsItunes: "http://www.itunes.com/dtds/podcast-1.0.dtd",
		Channel:     channel,
	}

	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	_, err = io.WriteString(w, xml.Header)
	if err != nil {
		return err
	}
	return enc.Encode(feed)
}

func (g *Generator) episodeToItem(ep repository.Episode) Item {
	pubDate := ep.PubDate.Format(time.RFC1123Z)

	item := Item{
		XMLName:     xml.Name{Local: "item", Space: ""},
		Title:       ep.Title,
		Description: ep.Description,
		PubDate:     pubDate,
		Author:      ep.Author,
		Duration:    formatDuration(ep.DurationSecs),
		Enclosure: Enclosure{
			URL:    fmt.Sprintf("%s/files/%s/%s", g.baseURL, ep.UUID, ep.FileName),
			Length: ep.FileSize,
			Type:   ep.MimeType,
		},
	}

	return item
}

func formatDuration(secs int) string {
	h := secs / 3600
	m := (secs % 3600) / 60
	s := secs % 60

	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}
