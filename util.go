package main

import (
	"encoding/json"
	"github.com/bwmarrin/discordgo"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

var (
	sourcePath string
	configPath string
	configText []byte = []byte("{\n\t\"CommandKey\": \"$\",\n\t\"CommandChannelID\": \"\",\n \n\t\"ListMessageID\": \"\"\n}")
)

type Configuration struct {
	CommandKey       string
	CommandChannelID string
	ListMessageID    string
}

var cfg Configuration

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
		if ar.Name == "Admin" {
			adminRoleID = ar.ID
		}
	}
	if adminRoleID != "" {
		for _, r := range user.Roles {
			if r == adminRoleID {
				return true
			}
		}
	} else {
		log.Println("No role by name of \"Admin\", Things might not go so well :/")
		return false
	}
	return false
}
func config(g *discordgo.Guild) {
	configPath = strings.ToLower(strings.Replace("radiobot/"+g.Name, " ", "", -1)) + "/config.json"
	sourcePath = strings.ToLower(strings.Replace("radiobot/"+g.Name, " ", "", -1)) + "/sources.txt"
	if _, e := os.Stat(configPath); os.IsNotExist(e) {
		os.MkdirAll(strings.ToLower(strings.Replace("radiobot/"+g.Name, " ", "", -1)), os.ModePerm)
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
func isConfigured(g *discordgo.Guild) bool {
	config(g)
	if cfg.CommandChannelID == "" || cfg.ListMessageID == "" {
		return false
	}
	return true
}
func writeConfig(g *discordgo.Guild, c string, m string) {
	newConfigText := []byte("{\n\t\"CommandKey\": \"$\",\n\t\"CommandChannelID\": \"" + c + "\",\n \n\t\"ListMessageID\": \"" + m + "\"\n}")
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
	m, e := s.ChannelMessageSend(mc.ChannelID, (cfg.CommandKey + "help - Shows this dialog."))
	if err(e, "Couldn't post message: "+m.Content) {
		return
	}
	m, e = s.ChannelMessageSend(mc.ChannelID, (cfg.CommandKey + "stop - Stops all audio."))
	if err(e, "Couldn't post message: "+m.Content) {
		return
	}
	m, e = s.ChannelMessageSend(mc.ChannelID, (cfg.CommandKey + "play `sourceURL` - Streams audio from specified url."))
	if err(e, "Couldn't post message: "+m.Content) {
		return
	}
	m, e = s.ChannelMessageSend(mc.ChannelID, (cfg.CommandKey + "add `name` `sourceURL` - Saves a source for easy playback."))
	if err(e, "Couldn't post message: "+m.Content) {
		return
	}
}
