package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/bwmarrin/discordgo"
)

var (
	workDir    string
	sourcePath string
	configPath string
	queueIndex int
	configText []byte = []byte("{\n\t\"CommandChannelID\": \"\",\n \n\t\"ListMessageID\": \"\"\n}")
)

type Configuration struct {
	CommandChannelID string
	ListMessageID    string
}

type QueueItem struct {
	Session *discordgo.Session
	Guild   *discordgo.Guild
	User    *discordgo.User
	Source  string
}

var (
	queue []QueueItem
	cfg   Configuration
	item  QueueItem
)

func authenticate(s *discordgo.Session, g string, u *discordgo.User) bool {
	user, e := s.GuildMember(g, u.ID)
	if err(e, "") {
		return false
	}
	roles, e := s.GuildRoles(g)
	if err(e, "") {
		return false
	}
	for _, ar := range roles {
		if ar.Permissions == 8 {
			for _, r := range user.Roles {
				if r == ar.ID {
					return true
				}
			}
			return false
		}
	}
	log.Println("Make sure admins only have the permission \"Administrator,\" they override other permissions anyway. ;)")
	return false
}
func config(g *discordgo.Guild, s *discordgo.Session) {
	workDir = strings.ToLower(strings.Replace(s.State.User.Username+"/"+g.Name, " ", "", -1))
	configPath = workDir + "/config.json"
	sourcePath = workDir + "/sources.txt"
	if _, e := os.Stat(configPath); os.IsNotExist(e) {
		os.MkdirAll((workDir), os.ModePerm)
		log.Println("No working directory found, creating one.")
	}
	if _, e := os.Stat(configPath); os.IsNotExist(e) {
		os.Create(configPath)
		ioutil.WriteFile(configPath, configText, os.ModePerm)
		log.Println("No '" + configPath + "' file found, creating one.")
	}
	if _, e := os.Stat(sourcePath); os.IsNotExist(e) {
		os.Create(sourcePath)
		log.Println("No '" + sourcePath + "' file found, creating one.")
	}
	configFile, _ := os.Open(configPath)
	decoder := json.NewDecoder(configFile)
	cfg = Configuration{}
	e := decoder.Decode(&cfg)
	if err(e, "") {
		return
	}
}
func isConfigured(g *discordgo.Guild, s *discordgo.Session) bool {
	config(g, s)
	if cfg.CommandChannelID == "" || cfg.ListMessageID == "" {
		return false
	}
	return true
}
func writeConfig(g *discordgo.Guild, c string, m string) {
	newConfigText := []byte("{\n\t\"CommandChannelID\": \"" + c + "\",\n \n\t\"ListMessageID\": \"" + m + "\"\n}")
	ioutil.WriteFile(configPath, newConfigText, os.ModePerm)
	log.Println("Configuring Guild: " + g.Name)
}
func addSource(g *discordgo.Guild, name string, source string) bool {
	sourceText, e := ioutil.ReadFile(sourcePath)
	if err(e, "") {
		return false
	}
	newSourceText := name + ", " + source + "\n"
	ioutil.WriteFile(sourcePath, []byte((string(sourceText) + newSourceText)), os.ModePerm)
	return true
}
func getSources(g *discordgo.Guild, i int) string {
	text := ""
	sourceText, e := ioutil.ReadFile(sourcePath)
	if err(e, "") {
		return text
	}
	if string(sourceText) != "" {
		sourceSplit := strings.Split(string(sourceText), "\n")
		for _, r := range sourceSplit {
			s := strings.Split(r, ", ")
			if i == 0 {
				text += (s[0] + "\n")
			} else if i == 1 {
				text += (s[1] + "\n")
			}
		}
		return text
	} else {
		return ""
	}
}
func getSourceByName(g *discordgo.Guild, name string) string {
	sourceText, e := ioutil.ReadFile(sourcePath)
	if err(e, "") {
		return ""
	}
	sourceSplit := strings.Split(string(sourceText), "\n")
	for _, r := range sourceSplit {
		s := strings.Split(r, ", ")
		if name == s[0] {
			return s[1]
		}
	}
	return ""
}
func addToQueue(s *discordgo.Session, g *discordgo.Guild, user *discordgo.User, source string) {
	item.Session = s
	item.Guild = g
	item.User = user
	item.Source = source
	queue = append(queue, item)
	return
}
func nextInQueue() {
	if skip {
		if len(queue)-queueIndex >= 1 {
			q := queue[queueIndex]
			playSound(q.Session, q.Guild, q.User, q.Source)
			queueIndex++
			skip = false
			return
		}
		skip = false
		queueIndex = 0
		queue = queue[:0]
		if running {
			item.Session.UpdateStatus(0, "from Source")
			return
		}
		item.Session.UpdateStatus(0, "")
		return
	}
}
func getGuild(s *discordgo.Session, c string) *discordgo.Guild {
	channel, e := s.State.Channel(c)
	if err(e, "") {
		return nil
	}
	g, e := s.State.Guild(channel.GuildID)
	if err(e, "") {
		return nil
	}
	return g
}
func err(e error, c string) bool {
	if e != nil {
		if c != "" {
			log.Println(c)
		}
		log.Println("Error:", e)
		return true
	} else {
		return false
	}
}
func help(s *discordgo.Session, mc *discordgo.MessageCreate) {
	m, e := s.ChannelMessageSend(mc.ChannelID, ("`@" + s.State.User.Username + "#" + s.State.User.Discriminator + "` help - Shows this dialog."))
	if err(e, "Couldn't post message: "+m.Content) {
		return
	}
	m, e = s.ChannelMessageSend(mc.ChannelID, ("`@" + s.State.User.Username + "#" + s.State.User.Discriminator + "` play `sourceURL` - Streams/queues audio from specified url."))
	if err(e, "Couldn't post message: "+m.Content) {
		return
	}
	m, e = s.ChannelMessageSend(mc.ChannelID, ("`@" + s.State.User.Username + "#" + s.State.User.Discriminator + "` add `name` `sourceURL` - Saves a source for easy playback."))
	if err(e, "Couldn't post message: "+m.Content) {
		return
	}
	m, e = s.ChannelMessageSend(mc.ChannelID, ("`@" + s.State.User.Username + "#" + s.State.User.Discriminator + "` next - Skips to next in queue."))
	if err(e, "Couldn't post message: "+m.Content) {
		return
	}
	m, e = s.ChannelMessageSend(mc.ChannelID, ("`@" + s.State.User.Username + "#" + s.State.User.Discriminator + "` stop - Stops playing."))
	if err(e, "Couldn't post message: "+m.Content) {
		return
	}
}
