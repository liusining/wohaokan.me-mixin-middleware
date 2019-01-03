package main

import (
	"context"
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
	missParams := func(field string) {
		apiError(c, fmt.Errorf("missing params: %s", field))
	}
	assetID, ok := c.GetPostForm("asset_id")
	if !ok {
		missParams("asset_id")
		return
	}
	recipientID, ok := c.GetPostForm("endpoint")
	if !ok {
		missParams("endpoint")
		return
	}
	amount, ok := c.GetPostForm("amount")
	if !ok {
		missParams("amount")
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

func apiError(c *gin.Context, err error) {
	c.JSON(400, gin.H{
		"err": fmt.Sprintf("%s", err),
	})
	fmt.Printf("err: %s\n", err)
}
