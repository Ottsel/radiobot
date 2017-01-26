package main

import (
	"C"
	"flag"
	"github.com/bwmarrin/discordgo"
	"log"
	"strings"
	"time"
)

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
		s.ChannelMessageSend(event.Guild.ID, "Not configured. Type `"+cfg.CommandKey+"config` in the channel you would like to keep commands in.")
	} else {
		listSources(s, event.Guild)
	}
}
func messageCreate(s *discordgo.Session, mc *discordgo.MessageCreate) {
	if !mc.Author.Bot {
		g := getGuild(s, mc.ChannelID)
		config(g, s)

		/*
		 *	Admin commands
		 */

		//Configure via Discord
		if strings.HasPrefix(mc.Content, cfg.CommandKey+"config") {
			if !isConfigured(g, s) {
				//if authenticate(s, g.ID, mc.Author) {
				m, e := s.ChannelMessageSend(mc.ChannelID, "Listing available sounds and pinning message...")
				if err(e, "Couldn't post message: "+m.Content) {
					return
				}
				writeConfig(g, mc.ChannelID, m.ID)
				listSources(s, g)
				e = s.ChannelMessagePin(m.ChannelID, m.ID)
				if err(e, "") {
					return
				}
				//}
			}
			return
		}
		if isConfigured(g, s) {
			//Post a list of commands
			if strings.HasPrefix(mc.Content, cfg.CommandKey+"help") {
				help(s, mc)
				return
			}

			/*
			 *	Soundboard commands
			 */

			if strings.HasPrefix(mc.Content, cfg.CommandKey) {
				channel, e := s.Channel(mc.ChannelID)
				if err(e, "") {
					return
				}
				if channel.ID != cfg.CommandChannelID {
					e := s.ChannelMessageDelete(channel.ID, mc.ID)
					if err(e, "Couldn't delete message:"+mc.Content) {
						return
					}
				}
				//Stop the player
				if mc.Content == cfg.CommandKey+"stop" {
					s.UpdateStatus(0, "")
					KillPlayer()
					return
				}
				//Play from source
				if strings.HasPrefix(mc.Content, cfg.CommandKey+"play ") {
					source := strings.Replace(mc.Content, cfg.CommandKey+"play ", "", -1)
					playSound(s, g, mc.Author, source)
					return
				}
				if strings.HasPrefix(mc.Content, cfg.CommandKey+"add ") {
					command := strings.Replace(mc.Content, cfg.CommandKey+"add ", "", -1)
					if strings.Count(command, " ") == 1 {
						params := strings.Split(command, " ")
						switch params[0] {
						case "play":
							s.ChannelMessageSend(mc.ChannelID, "Invalid name: "+params[0])
							return
						case "add":
							s.ChannelMessageSend(mc.ChannelID, "Invalid name: "+params[0])
							return
						case "help":
							s.ChannelMessageSend(mc.ChannelID, "Invalid name: "+params[0])
							return
						case "config":
							s.ChannelMessageSend(mc.ChannelID, "Invalid name: "+params[0])
							return
						case "stop":
							s.ChannelMessageSend(mc.ChannelID, "Invalid name: "+params[0])
							return
						default:
							sources := getSources(g, 0)
							if strings.Contains(sources, params[0]) {
								s.ChannelMessageSend(mc.ChannelID, "A source by that name already exists.")
								return
							} else {
								if addSource(g, params[0], params[1]) {
									s.ChannelMessageSend(mc.ChannelID, ("Added source: `" + params[0] + "`"))
									listSources(s, g)
								}
								return
							}
						}

					} else {
						s.ChannelMessageSend(mc.ChannelID, "Usage: "+cfg.CommandKey+"add `name` `sourceURL`")
						return
					}
				}
				name := strings.ToLower(strings.Replace(mc.Content, cfg.CommandKey, "", -1))
				source := getSourceByName(g, name)
				if source != "" {
					s.UpdateStatus(0, strings.ToTitle(name))
					playSound(s, g, mc.Author, source)
				} else {
					s.ChannelMessageSend(mc.ChannelID, "No cached source by name of: "+strings.ToLower(name))
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
		m, e := s.ChannelMessageEdit(cfg.CommandChannelID, cfg.ListMessageID, cfg.CommandKey+messageText)
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
