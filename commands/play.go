package commands

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/bottleneckco/radio-clerk/models"

	"google.golang.org/api/youtube/v3"

	"github.com/bwmarrin/discordgo"
)

var tempSearchResultsCache = make(map[string][]*youtube.SearchResult)

func play(s *discordgo.Session, m *discordgo.MessageCreate) {
	var b strings.Builder
	if url, err := url.ParseRequestURI(m.Content); err == nil {
		// URL
		var videoIDs []string
		if len(url.Query().Get("list")) != 0 {
			// Playlist URL
			playlistResponse, err := youtubeService.PlaylistItems.List("contentDetails").PlaylistId(url.Query().Get("list")).MaxResults(50).Do()
			if err != nil {
				s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Error occurred: %s", err))
				return
			}
			for _, playlistItem := range playlistResponse.Items {
				videoIDs = append(videoIDs, playlistItem.ContentDetails.VideoId)
			}
		} else if len(url.Query().Get("v")) != 0 {
			// Plain video URL
			videoIDs = append(videoIDs, url.Query().Get("v"))
		} else {
			s.ChannelMessageSend(m.ChannelID, "Unsupported URL")
			return
		}
		youtubeListings, err := youtubeService.Videos.List("snippet").Id(strings.Join(videoIDs, ",")).Do()
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Error occurred: %s", err))
			return
		}
		for _, youtubeListing := range youtubeListings.Items {
			Queue = append(Queue, models.QueueItem{
				Title:        youtubeListing.Snippet.Title,
				ChannelTitle: youtubeListing.Snippet.ChannelTitle,
				Author:       m.Author.Username,
				VideoID:      youtubeListing.Id,
			})
		}

		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s enqueued %d videos`\n", m.Author.Mention(), len(videoIDs)))
	} else {
		maxResults, _ := strconv.ParseInt(os.Getenv("BOT_NUM_RESULTS"), 10, 64)
		call := youtubeService.Search.List("snippet").Q(m.Content).MaxResults(maxResults)
		response, err := call.Do()
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Error occurred: %s", err))
			return
		}

		log.Printf("[PLAY] Found %d results", len(response.Items))

		b.WriteString(fmt.Sprintf("%s, here are your search results:\n", m.Author.Mention()))

		for index, item := range response.Items {
			b.WriteString(fmt.Sprintf("`%d.` **%s** `%s`\n", index+1, item.Snippet.Title, item.Snippet.ChannelTitle))
		}
		b.WriteString("\nreply with a single number, no command needed. Anything else will cancel the search.")

		// Cache for response
		tempSearchResultsCache[m.Author.ID] = response.Items

		var awaitFuncRemove func()

		awaitFuncRemove = s.AddHandler(func(ss *discordgo.Session, mm *discordgo.MessageCreate) {
			if m.Author.ID != mm.Author.ID {
				return
			}
			choice, err := strconv.ParseInt(mm.Content, 10, 64)
			if err == nil && (choice-1 < 0 || choice-1 >= int64(len(tempSearchResultsCache[mm.Author.ID]))) {
				// Valid integer but out of bounds
				ss.ChannelMessageSend(mm.ChannelID, fmt.Sprintf("%s Invalid choice, try again", mm.Author.Mention()))
				return
			} else if err == nil {
				chosenItem := tempSearchResultsCache[mm.Author.ID][choice-1]
				Queue = append(Queue, models.QueueItem{
					Title:        chosenItem.Snippet.Title,
					ChannelTitle: chosenItem.Snippet.ChannelTitle,
					Author:       mm.Author.Username,
					VideoID:      chosenItem.Id.VideoId,
				})
				ss.ChannelMessageSend(mm.ChannelID, fmt.Sprintf("%s enqueued:\n`️➕`**%s** `%s`\n", mm.Author.Mention(), chosenItem.Snippet.Title, chosenItem.Snippet.ChannelTitle))
			} else {
				ss.ChannelMessageSend(mm.ChannelID, fmt.Sprintf("%s Search cancelled", mm.Author.Mention()))
			}
			delete(tempSearchResultsCache, mm.Author.ID)
			awaitFuncRemove()
		})
	}

	s.ChannelMessageSend(m.ChannelID, b.String())
}
