package main

import (
	"context"
	"fmt"
	"io"
	mathrand "math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/eatmoreapple/openwechat"
	gogpt "github.com/sashabaranov/go-gpt3"
)

func main() {

	// bot := openwechat.DefaultBot()
	bot := openwechat.DefaultBot(openwechat.Desktop) // 桌面模式，上面登录不上的可以尝试切换这种模式

	// 注册消息处理函数
	bot.MessageHandler = func(msg *openwechat.Message) {

		// 回复群里/私聊消息
		if msg.IsText() && !msg.IsSendBySelf() {

			var sender *openwechat.User
			var question string
			var err error

			if msg.IsSendByGroup() && strings.HasPrefix(msg.Content, "@AI哥") {
				// 群聊@自己
				fmt.Println("--------------来自 群聊----------------------------------")
				sender, err = msg.SenderInGroup()
				if err != nil {
					fmt.Println("微信请求异常：" + err.Error())
					return
				}
				question = strings.TrimSpace(msg.Content[6:])
			} else if msg.IsSendByFriend() {
				// 私聊自己
				fmt.Println("--------------来自 私聊----------------------------------")
				sender, err = msg.Sender()
				if err != nil {
					fmt.Println("微信请求异常：" + err.Error())
					return
				}
				question = strings.TrimSpace(msg.Content)
			} else {
				return
			}

			if sender == nil {
				fmt.Println("发送者 sender 为空")
				msg.ReplyText("咱这网络不太好，可以试着重新提问")
				return
			}
			if question == "" {
				fmt.Println("问题 question 为空")
				msg.ReplyText("可以问一个正常的问题哦")
				return
			}
			fmt.Println("发送者：" + sender.String())
			fmt.Println("问题：" + question)

			if strings.HasPrefix(question, "#图片生成") {
				// 生成图片
				question = strings.TrimSpace(question[13:])
				fullPath, err := AiImage(question)
				if err != nil {
					fmt.Println("OPENAI请求异常：" + err.Error())
					msg.ReplyText("咱这网络不太好，可以试着重新提问")
					return
				}
				fmt.Println("图片路径：" + fullPath)
				image, _ := os.Open(fullPath)
				defer image.Close()
				msg.ReplyImage(image)
			} else {
				// 回复消息
				answer, err := AiChat(question)
				if err != nil {
					fmt.Println("OPENAI请求异常：" + err.Error())
					msg.ReplyText("咱这网络不太好，可以试着重新提问")
					return
				}
				answer = GetFilteredAnswer(answer)
				fmt.Println("答案：" + answer)
				msg.ReplyText(answer)
			}

		}

		// 自动添加好友
		if msg.IsFriendAdd() {
			fmt.Println("--------------添加好友----------------------------------")
			friend, err := msg.Agree()
			if err != nil {
				fmt.Println("微信请求异常：" + err.Error())
				return
			}
			fmt.Println("已添加好友：" + friend.String())
			friend.SendText("你好呀，我是AI哥，请问有什么可以帮到您？")
		}

	}

	// 注册登陆二维码回调
	bot.UUIDCallback = openwechat.PrintlnQrcodeUrl

	// 登陆
	if err := bot.Login(); err != nil {
		fmt.Println(err)
		return
	}

	// 阻塞主goroutine, 直到发生异常或者用户主动退出
	bot.Block()
}

func GetFilteredAnswer(answer string) string {
	answer = strings.TrimLeft(answer, "?")
	answer = strings.TrimLeft(answer, "？")
	answer = strings.TrimLeft(answer, ":")
	answer = strings.TrimLeft(answer, "：")
	answer = strings.TrimSpace(answer)
	answer = strings.TrimLeft(answer, "\n")
	return answer
}

var chatOnce sync.Once
var chatClient *gogpt.Client

func AiChat(question string) (string, error) {

	chatOnce.Do(func() {
		chatClient = gogpt.NewClient("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")
	})

	req := gogpt.CompletionRequest{
		Model:       gogpt.GPT3TextDavinci003,
		MaxTokens:   4000,
		Prompt:      question,
		Temperature: 0.1,
	}
	resp, err := chatClient.CreateCompletion(context.Background(), req)
	if err != nil {
		return "", err
	}

	return resp.Choices[0].Text, nil
}

var imageOnce sync.Once
var imageClient *gogpt.Client

func AiImage(question string) (string, error) {

	imageOnce.Do(func() {
		imageClient = gogpt.NewClient("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")
	})

	req := gogpt.ImageRequest{
		Prompt: question,
		N:      1,
		Size:   gogpt.CreateImageSize512x512,
	}
	resp, err := imageClient.CreateImage(context.Background(), req)
	if err != nil {
		return "", err
	}

	storagePath := "storage"
	os.MkdirAll(storagePath, 0755)

	path := storagePath + "/" + randomString(16) + ".png"
	downloadFile(resp.Data[0].URL, path)

	pwd, _ := os.Getwd()
	return pwd + "/" + path, nil
}

func randomString(length int) string {
	mathrand.Seed(time.Now().UnixNano())
	letters := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, length)
	for i := range b {
		b[i] = letters[mathrand.Intn(len(letters))]
	}
	return string(b)
}

func downloadFile(url string, path string) {
	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	// Create output file
	out, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	defer out.Close()
	// copy stream
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		panic(err)
	}
}
