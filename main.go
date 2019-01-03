package main

import (
	"context"
	"encoding/base64"
	"fmt"

	mixin "github.com/MixinNetwork/bot-api-go-client"
	"github.com/MixinNetwork/go-number"
	"github.com/gin-gonic/gin"
	"github.com/satori/go.uuid"
	"github.com/spf13/viper"
)

func init() {
	viper.SetConfigType("yaml")
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("loading config file: %s", err))
	}
	fmt.Println("Config loaded")
}

func main() {
	r := gin.Default()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(func(c *gin.Context) {
		requestID := func() string {
			if id, ok := c.GetPostForm("request_id"); ok {
				return id
			} else if id, ok = c.GetQuery("request_id"); ok {
				return id
			} else {
				req := struct {
					RequestID string `json:"request_id"`
				}{}
				c.BindJSON(&req)
				return req.RequestID
			}
		}()
		c.Set("RequestID", requestID)
		fmt.Printf("RequestID: %s\n", requestID)
		c.Next()
	})
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"msg": "pong",
		})
	})
	r.POST("/auth_info", authInfo)
	r.POST("/deliver_money", deliverMoney)
	r.POST("/deliver_contact", deliverContact)
	r.Run(viper.GetString("service.bind_ip"))
}

func authInfo(c *gin.Context) {
	ctx := context.Background()
	fmt.Println("Applying for an access token")
	authCode, _ := c.GetPostForm("auth_code")
	accessToken, scope, err := mixin.OAuthGetAccessToken(ctx,
		viper.GetString("mixin.client_id"),
		viper.GetString("mixin.client_secret"),
		authCode, "")
	fmt.Printf("\nDone\n")
	if err != nil {
		apiError(c, err)
		return
	}
	user, err := mixin.UserMe(ctx, accessToken)
	if err != nil {
		apiError(c, err)
		return
	}
	c.JSON(200, gin.H{
		"access_token":    accessToken,
		"scope":           scope,
		"user_id":         user.UserId,
		"session_id":      user.SessionId,
		"pin_token":       user.PinToken,
		"identity_number": user.IdentityNumber,
		"full_name":       user.FullName,
		"avatar_url":      user.AvatarURL,
	})
}

func deliverMoney(c *gin.Context) {
	ctx := context.Background()
	assetID, ok := c.GetPostForm("asset_id")
	if !ok {
		missParams(c, "asset_id")
		return
	}
	recipientID, ok := c.GetPostForm("endpoint")
	if !ok {
		missParams(c, "endpoint")
		return
	}
	amount, ok := c.GetPostForm("amount")
	if !ok {
		missParams(c, "amount")
		return
	}
	traceID := uuid.Must(uuid.NewV1()).String()
	memo, _ := c.GetPostForm("memo")
	tran := mixin.TransferInput{
		AssetId:     assetID,
		RecipientId: recipientID,
		Amount:      number.FromString(amount),
		TraceId:     traceID,
		Memo:        memo,
	}
	err := mixin.CreateTransfer(ctx, &tran,
		viper.GetString("mixin.client_id"),
		viper.GetString("mixin.session_id"),
		viper.GetString("mixin.private_key"),
		viper.GetString("mixin.pin"),
		viper.GetString("mixin.pin_token"))
	if err != nil {
		apiError(c, err)
		return
	}
	c.JSON(200, gin.H{
		"trace_id": traceID,
	})
}

func deliverContact(c *gin.Context) {
	ctx := context.Background()
	mixinUID, ok := c.GetPostForm("mixin_uid")
	if !ok {
		missParams(c, "mixin_uid")
		return
	}
	contactUID, ok := c.GetPostForm("contact_uid")
	if !ok {
		missParams(c, "contact_uid")
		return
	}
	conversationID := mixin.UniqueConversationId(
		viper.GetString("mixin.client_id"), mixinUID)
	participants := []mixin.Participant{mixin.Participant{UserId: mixinUID, Role: ""}}
	_, err := mixin.CreateConversation(ctx, "CONTACT",
		conversationID, participants, viper.GetString("mixin.client_id"),
		viper.GetString("mixin.session_id"), viper.GetString("mixin.private_key"))
	if err != nil {
		apiError(c, err)
		return
	}
	contact := fmt.Sprintf("{\"user_id\":\"%s\"}", contactUID)
	msgData := base64.StdEncoding.EncodeToString([]byte(contact))
	err = mixin.PostMessage(ctx, conversationID,
		mixinUID, mixin.UuidNewV4().String(),
		"PLAIN_CONTACT", msgData, viper.GetString("mixin.client_id"),
		viper.GetString("mixin.session_id"), viper.GetString("mixin.private_key"))
	if err != nil {
		apiError(c, err)
		return
	}
	c.JSON(200, gin.H{
		"conversation_id": conversationID,
	})
}

func apiError(c *gin.Context, err error) {
	c.JSON(400, gin.H{
		"err": fmt.Sprintf("%s", err),
	})
	fmt.Printf("err: %s\n", err)
}

func missParams(c *gin.Context, field string) {
	apiError(c, fmt.Errorf("missing params: %s", field))
}