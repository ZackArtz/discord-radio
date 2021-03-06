package commands

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func queue(s *discordgo.Session, m *discordgo.MessageCreate) {
	guildSession := safeGetGuildSession(s, m.Message.GuildID)
	guildSession.RWMutex.RLock()
	if len(guildSession.Queue) == 0 {
		s.ChannelMessageSend(m.Message.ChannelID, fmt.Sprintf("%s nothing in the queue.", m.Message.Author.Mention()))
		guildSession.RWMutex.RUnlock()
		return
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("%s here is the queue:\n", m.Message.Author.Mention()))

	b.WriteString(fmt.Sprintf("⏯ **%s**    ▶️️%s   ⏫%s\n", guildSession.Queue[0].Title, guildSession.Queue[0].ChannelTitle, guildSession.Queue[0].Author))
	for index, queueItem := range guildSession.Queue[1:] {
		b.WriteString(fmt.Sprintf("`️%d.` **%s**   ⬆️%s   ⏫%s\n", index+2, queueItem.Title, queueItem.ChannelTitle, queueItem.Author))
	}
	guildSession.RWMutex.RUnlock()
	s.ChannelMessageSend(m.Message.ChannelID, b.String())
}
