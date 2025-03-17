package discord

import (
	"log"
	"time"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/eraiza0816/llm-discord/chat"
	"github.com/eraiza0816/llm-discord/loader"
)

func setupHandlers(s *discordgo.Session, chatSvc chat.Service, modelCfg *loader.ModelConfig) {
	s.AddHandler(onReady)
	s.AddHandler(interactionCreate(chatSvc, modelCfg))
}

func onReady(s *discordgo.Session, event *discordgo.Ready) {
	log.Printf("Bot is ready! %s#%s", s.State.User.Username, s.State.User.Discriminator)
}

func interactionCreate(chatSvc chat.Service, modelCfg *loader.ModelConfig) func(s *discordgo.Session, i *discordgo.InteractionCreate) {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.Type != discordgo.InteractionApplicationCommand {
			return
		}

		switch i.ApplicationCommandData().Name {
		case "chat":
			userID := i.Member.User.ID
			username := i.Member.User.Username
			message := i.ApplicationCommandData().Options[0].StringValue()
			timestamp := time.Now().Format(time.RFC3339)

			userPrompt := modelCfg.GetPromptByUser(username)

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "ちょっと待ってね！",
				},
			})

			resp, elapsed, err := chatSvc.GetResponse(userID, username, message, timestamp, userPrompt)
			if err != nil {
				log.Printf("GetResponse error: %v", err)
				resp = "エラーが発生しました。"
			}

			content := resp + "\n" + fmt.Sprintf("%.2fms", elapsed)
			_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: &content,
			})
			if err != nil {
				log.Printf("InteractionResponseEdit error: %v", err)
			}
		}
	}
}
