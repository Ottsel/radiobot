package main

import (
	"C"
	"flag"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)
import "strconv"

var (
	botID string
)

func main() {
	var (
		Token = flag.String("t", "", "Discord Authentication Token")
	)
	flag.Parse()

	dg, e := discordgo.New("Bot " + *Token)
	if err(e, "") {
		return
	}
	dg.AddHandler(messageCreate)
	dg.AddHandler(onGuildCreate)
	e = dg.Open()
	if err(e, "") {
		return
	}
	<-make(chan struct{})
	return
}
func onGuildCreate(s *discordgo.Session, event *discordgo.GuildCreate) {
	if event.Guild.Unavailable == true {
		return
	}
	if !isConfigured(event.Guild, s) {
		s.ChannelMessageSend(event.Guild.ID, "Not configured. Type `@"+s.State.User.Username+"#"+s.State.User.Discriminator+"` `config` in the channel you would like to keep commands in.")
		return
	}
	listSources(s, event.Guild)
	return
}
func messageCreate(s *discordgo.Session, mc *discordgo.MessageCreate) {
	if !mc.Author.Bot {
		for _, r := range mc.Mentions {
			if r.Username == s.State.User.Username {
				message := strings.Replace(mc.Content, "  ", " ", -1)
				if !strings.Contains(message, "  ") {

					g := getGuild(s, mc.ChannelID)
					sources := getSources(g, 0)

					config(g, s)
					listSources(s, g)

					params := strings.Split(message, " ")
					switch strings.ToLower(params[1]) {
					case "config":
						if !isConfigured(g, s) {
							if authenticate(s, g.ID, mc.Author) {
								m, e := s.ChannelMessageSend(mc.ChannelID, "Listing available sources and pinning message...")
								if err(e, "Couldn't post message: "+m.Content) {
									return
								}
								writeConfig(g, mc.ChannelID, m.ID)
								listSources(s, g)
								e = s.ChannelMessagePin(mc.ChannelID, m.ID)
								if err(e, "") {
									return
								}
							}
						}
						return
					case "add":
						if len(params) == 4 && isConfigured(g, s) {
							if strings.Contains(sources, params[2]) {
								s.ChannelMessageSend(mc.ChannelID, "A source by the name "+params[2]+" already exists.")
								return
							}
							if addSource(g, params[2], params[3]) {
								s.ChannelMessageSend(mc.ChannelID, ("Added source: `" + params[2] + "`"))
								return
							}
							s.ChannelMessageSend(mc.ChannelID, ("Failed to add source: `" + params[2] + "`"))
							return
						}
						s.ChannelMessageSend(mc.ChannelID, "Usage: `@"+s.State.User.Username+"#"+s.State.User.Discriminator+"` add `name` `sourceURL`")
						return
					case "play":
						if len(params) == 3 && isConfigured(g, s) {
							if running {
								addToQueue(s, g, mc.Author, params[2])
								pos := strconv.Itoa(len(queue) - (queueIndex))
								s.ChannelMessageSend(mc.ChannelID, ("You are position " + pos + " in the queue. (Use the `next` command to skip)"))
								s.UpdateStatus(0, "from Queue")
								return
							}
							s.UpdateStatus(0, "from Source")
							playSound(s, g, mc.Author, params[2])
							return
						}
						s.ChannelMessageSend(mc.ChannelID, "Usage: `@"+s.State.User.Username+"#"+s.State.User.Discriminator+"` play `sourceURL`")
						return
					case "next":
						if isConfigured(g, s) {
							nextInQueue()
							return
						}
					case "stop":
						if isConfigured(g, s) {
							queueIndex = 0
							queue = queue[:0]
							s.UpdateStatus(0, "")
							KillPlayer()
							return
						}
					case "help":
						if isConfigured(g, s) {
							help(s, mc)
							return
						}
					default:
						if isConfigured(g, s) {
							source := getSourceByName(g, strings.ToLower(params[1]))
							if source != "" {
								if running {
									addToQueue(s, g, mc.Author, source)
									pos := strconv.Itoa(len(queue) - (queueIndex))
									s.ChannelMessageSend(mc.ChannelID, ("You are position " + pos + " in the queue. (Use the `next` command to skip)"))
									return
								}
								s.UpdateStatus(0, strings.ToTitle(params[1]))
								playSound(s, g, mc.Author, source)
								s.ChannelMessageSend(mc.ChannelID, "No cached source by name of: "+strings.ToLower(params[1]))
								return
							}
						}
					}
					if mc.ChannelID != cfg.CommandChannelID {
						e := s.ChannelMessageDelete(mc.ChannelID, mc.ID)
						if err(e, "Couldn't delete message:"+mc.Content) {
							return
						}
					}
				}
			}
		}
	}
}
func playSound(s *discordgo.Session, g *discordgo.Guild, user *discordgo.User, source string) {
	if isConfigured(g, s) {
		config(g, s)
		var userVC string
		for _, v := range g.VoiceStates {
			if v.UserID == user.ID {
				userVC = v.ChannelID
			}
		}
		vc, e := s.ChannelVoiceJoin(g.ID, userVC, false, false)
		if err(e, "") {
			return
		}
		go func() {
			time.Sleep(time.Millisecond * 200)
			g, e = s.Guild(g.ID)
			for _, v := range g.VoiceStates {
				if v.UserID == s.State.User.ID {
					if v.ChannelID == userVC {
						listSources(s, g)
						log.Println("Attempting to play audio from source \"" + source + "\" for user: " + user.Username)
						KillPlayer()
						time.Sleep(time.Millisecond * 200)
						PlayAudioFile(vc, (source))
					}
				}
			}
		}()
	}
}
func listSources(s *discordgo.Session, g *discordgo.Guild) {
	config(g, s)
	messageText := getSources(g, 0)
	if messageText != "" {
		m, e := s.ChannelMessageEdit(cfg.CommandChannelID, cfg.ListMessageID, messageText)
		if err(e, "Couldn't edit source list message with ID: "+m.ID) {
			return
		} else {
			log.Println("Sources updated")
		}
	} else {
		m, e := s.ChannelMessageEdit(cfg.CommandChannelID, cfg.ListMessageID, "(No cached streaming sources. Yet...)")
		if err(e, "Couldn't edit source list message with ID: "+m.ID) {
			return
		} else {
			log.Println("Sources updated")
		}
	}
}
