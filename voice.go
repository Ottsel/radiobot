package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"sync"

	"github.com/bwmarrin/discordgo"
	"layeh.com/gopus"
)

const (
	channels  int = 2                   // 1 for mono, 2 for stereo
	frameRate int = 48000               // audio sampling rate
	frameSize int = 960                 // uint16 size of each audio frame
	maxBytes  int = (frameSize * 2) * 2 // max size of opus data
)

var (
	speakers    map[uint32]*gopus.Decoder
	opusEncoder *gopus.Encoder
	sendpcm     bool
	recvpcm     bool
	ffmpeg      *exec.Cmd
	running     bool
	recv        chan *discordgo.Packet
	send        chan []int16
	mu          sync.Mutex
	skip        bool
)

// SendPCM will receive on the provied channel encode
// received PCM data into Opus then send that to Discordgo
func SendPCM(v *discordgo.VoiceConnection, pcm <-chan []int16) {

	// make sure this only runs one instance at a time.
	mu.Lock()
	if sendpcm || pcm == nil {
		mu.Unlock()
		return
	}
	sendpcm = true
	mu.Unlock()

	defer func() { sendpcm = false }()

	var err error

	opusEncoder, err = gopus.NewEncoder(frameRate, channels, gopus.Audio)

	if err != nil {
		fmt.Println("NewEncoder Error:", err)
		return
	}

	for {

		// read pcm from chan, exit if channel is closed.
		recv, ok := <-pcm
		if !ok {
			fmt.Println("PCM Channel closed.")
			return
		}

		// try encoding pcm frame with Opus
		opus, err := opusEncoder.Encode(recv, frameSize, maxBytes)
		if err != nil {
			fmt.Println("Encoding Error:", err)
			return
		}

		if v.Ready == false || v.OpusSend == nil {
			fmt.Printf("Discordgo not ready for opus packets. %+v : %+v", v.Ready, v.OpusSend)
			return
		}
		// send encoded opus data to the sendOpus channel
		v.OpusSend <- opus
	}
}

// ReceivePCM will receive on the the Discordgo OpusRecv channel and decode
// the opus audio into PCM then send it on the provided channel.
func ReceivePCM(v *discordgo.VoiceConnection, c chan *discordgo.Packet) {

	// make sure this only runs one instance at a time.
	mu.Lock()
	if recvpcm || c == nil {
		mu.Unlock()
		return
	}
	recvpcm = true
	mu.Unlock()

	defer func() { sendpcm = false }()
	var err error

	for {

		if v.Ready == false || v.OpusRecv == nil {
			fmt.Printf("Discordgo not ready to receive opus packets. %+v : %+v", v.Ready, v.OpusRecv)
			return
		}

		p, ok := <-v.OpusRecv
		if !ok {
			return
		}

		if speakers == nil {
			speakers = make(map[uint32]*gopus.Decoder)
		}

		_, ok = speakers[p.SSRC]
		if !ok {
			speakers[p.SSRC], err = gopus.NewDecoder(48000, 2)
			if err != nil {
				fmt.Println("error creating opus decoder:", err)
				continue
			}
		}

		p.PCM, err = speakers[p.SSRC].Decode(p.Opus, 960, false)
		if err != nil {
			fmt.Println("Error decoding opus data: ", err)
			continue
		}

		c <- p
	}
}
func PlayAudioFile(v *discordgo.VoiceConnection, source string) {

	// Create a shell command "object" to run.
	if strings.Contains(source, "youtu") {
		ytdl := exec.Command("youtube-dl", "--no-cache-dir", "-f", "bestaudio", "-o", "-", source)
		ytdlout, err := ytdl.StdoutPipe()
		if err != nil {
			fmt.Println("musicplugin: ytdl StdoutPipe err:", err)
			return
		}
		ytdlbuf := bufio.NewReaderSize(ytdlout, 16384)
		err = ytdl.Start()
		if err != nil {
			fmt.Println("Ytdl Error:", err)
			return
		}

		ffmpeg = exec.Command("ffmpeg", "-i", "pipe:0", "-af", "volume=0.3", "-f", "s16le", "-ar", strconv.Itoa(frameRate), "-ac", strconv.Itoa(channels), "pipe:1")
		ffmpeg.Stdin = ytdlbuf

	} else {
		ffmpeg = exec.Command("ffmpeg", "-i", source, "-af", "volume=0.3", "-f", "s16le", "-ar", strconv.Itoa(frameRate), "-ac", strconv.Itoa(channels), "pipe:1")
	}

	ffmpegout, err := ffmpeg.StdoutPipe()
	if err != nil {
		fmt.Println("FFmpeg StdoutPipe Error:", err)
		return
	}

	ffmpegbuf := bufio.NewReaderSize(ffmpegout, 16384)

	// Starts the ffmpeg command
	err = ffmpeg.Start()
	if err != nil {
		fmt.Println("FFmpeg Error:", err)
		return
	} else {
		running = true
	}

	// Send "speaking" packet over the voice websocket
	v.Speaking(true)

	// Send not "speaking" packet over the websocket when we finish
	defer func() {
		v.Speaking(false)
		if skip {
			running = false
			nextInQueue()
		} else {
			skip = true
		}
	}()

	// will actually only spawn one instance, a bit hacky.
	if send == nil {
		send = make(chan []int16, 2)
	}
	go SendPCM(v, send)

	for {

		// read data from ffmpeg stdout
		audiobuf := make([]int16, frameSize*channels)
		err = binary.Read(ffmpegbuf, binary.LittleEndian, &audiobuf)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return
		}
		if err != nil {
			fmt.Println("error reading from ffmpeg stdout :", err)
			return
		}

		// Send received PCM to the sendPCM channel
		send <- audiobuf
	}
}
func KillPlayer() {
	if running {
		ffmpeg.Process.Kill()
		running = false
	}
}
