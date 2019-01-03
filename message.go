package main

import (
	"context"
	"encoding/base64"

	mixin "github.com/MixinNetwork/bot-api-go-client"
	"github.com/spf13/viper"
)

func deliverMessage(ctx context.Context, recipientID, category, msgData string) (conversationID, messageID string, err error) {
	conversationID = mixin.UniqueConversationId(
		viper.GetString("mixin.client_id"), recipientID)
	participants := []mixin.Participant{mixin.Participant{UserId: recipientID, Role: ""}}
	_, err = mixin.CreateConversation(ctx, "CONTACT",
		conversationID, participants, viper.GetString("mixin.client_id"),
		viper.GetString("mixin.session_id"), viper.GetString("mixin.private_key"))
	if err != nil {
		return "", "", err
	}
	// contact := fmt.Sprintf("{\"user_id\":\"%s\"}", contactUID)
	msgB64 := base64.StdEncoding.EncodeToString([]byte(msgData))
	msgID := mixin.UuidNewV4().String()
	err = mixin.PostMessage(ctx, conversationID,
		recipientID, msgID,
		"PLAIN_CONTACT", msgB64, viper.GetString("mixin.client_id"),
		viper.GetString("mixin.session_id"), viper.GetString("mixin.private_key"))
	if err != nil {
		return "", "", err
	}
	return conversationID, msgID, nil
}
