package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// Bot parameters
var (
	GuildID  = flag.String("guild", "953488559724183602", "Test guild ID")
	BotToken = flag.String("token", "", "Bot access token")
	AppID    = flag.String("app", "955836104559460362", "Application ID")
)

var s *discordgo.Session

func init() { flag.Parse() }

func init() {
	var err error
	s, err = discordgo.New("Bot " + *BotToken)
	if err != nil {
		log.Fatalf("Invalid bot parameters: %v", err)
	}
}

func main() {
	// add function handlers for go code execution
	s.AddHandler(goExecutionHandler)
	s.AddHandler(goReExec)

	err := s.Open()
	if err != nil {
		log.Fatalf("Cannot open the session: %v", err)
	}
	defer s.Close()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop
	log.Println("Graceful shutdown")
}

func goExecutionHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	// handle running code sent from user
	if c := strings.Split(m.Content, "\n"); c[0] == "run" {
		codeOuput := goExec(m.ChannelID, m.Content, m.Reference())
		sendMessageComplex(m.ChannelID, m.Reference(), string(codeOuput))
	}
}

func goReExec(s *discordgo.Session, m *discordgo.InteractionCreate) {
	if m.MessageComponentData().CustomID == "go_run" {
		msg, err := s.ChannelMessage(m.ChannelID, m.Message.MessageReference.MessageID)
		if err != nil {
			log.Fatalf("Could not get message reference: %v", err)
		}

		messageReference := m.Message.Reference()
		codeOutput := goExec(m.ChannelID, msg.Content, messageReference)
		editComplexMessage(m.Message.ID, m.ChannelID, messageReference, string(codeOutput))
	}
}

func goExec(channelID string, messageContent string, messageReference *discordgo.MessageReference) []byte {
	// check to see if we are executing go code
	// this is based on a writing standard in discord for writing code in a message
	// example message: ```go ... ```
	if c := strings.Split(messageContent, "\n"); strings.Contains(c[1], "go") {
		// add regex string replacements for content
		var r []regexp.Regexp
		runre := regexp.MustCompile("run")
		blockre := regexp.MustCompile("```.*")
		whitespacere := regexp.MustCompile("\n\n")
		r = append(r, *runre, *blockre, *whitespacere)

		// remove strings based on regex for proper code execution
		content := messageContent
		for _, regex := range r {
			content = regex.ReplaceAllString(content, "")
		}

		// create go execution file
		ioutil.WriteFile("code/code.go", []byte(content), 0644)

		// run command
		cmd := exec.Command("go", "run", "code/code.go")
		o, err := cmd.Output()

		// output error in discord if code did not successfully execute
		if err != nil {
			fmt.Println(err.Error())
			_, _ = s.ChannelMessageSendReply(channelID, err.Error(), messageReference)
			return nil
		}

		return o
	}

	return nil
}

func sendMessageComplex(channelID string, messageReference *discordgo.MessageReference, codeOutput string) {
	_, _ = s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Content:   fmt.Sprintf("Output:\n\n`%s`\n", string(codeOutput)),
		Reference: messageReference,
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "Run",
						Style:    discordgo.SuccessButton,
						CustomID: "go_run",
					},
				},
			},
		},
	})
}

func editComplexMessage(messageID string, channelID string, messageReference *discordgo.MessageReference, codeOutput string) {
	content := fmt.Sprintf("Output:\n\n`%s`\n", string(codeOutput))
	s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		ID:      messageID,
		Channel: channelID,
		Content: &content,
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "Run",
						Style:    discordgo.SuccessButton,
						CustomID: "go_run",
					},
				},
			},
		},
	})
}