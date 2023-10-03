package types

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/Adeithe/go-twitch/irc"
	ratelimiting "github.com/aidenwallis/go-ratelimiting/local"
	"github.com/nicklaw5/helix/v2"
	"github.com/samber/lo"
	model "github.com/satont/twir/libs/gomodels"
)

type Channel struct {
	IsMod   bool
	Limiter ratelimiting.SlidingWindow
	ID      string
}

type ChannelsMap struct {
	sync.Mutex
	Items map[string]*Channel
}

type RateLimiters struct {
	Global   ratelimiting.SlidingWindow
	Channels ChannelsMap
}

type BotClient struct {
	Reader      *irc.Client
	Writer      *irc.Conn
	JoinLimiter ratelimiting.SlidingWindow

	RateLimiters RateLimiters
	Model        *model.Bots
	TwitchUser   *helix.User
}

func (c *BotClient) Join(channels ...string) error {
	chunks := lo.Chunk(channels, 20)

	ctx := context.Background()

	for _, ch := range chunks {
		c.JoinLimiter.Wait(ctx)
		c.Reader.Join(ch...)
		c.Writer.Join(ch...)
	}

	return nil
}

func (c *BotClient) Say(channel, text string) {
	c.Writer.Say(channel, text)
}

func (c *BotClient) Reply(channel, text, messageId string) {
	msg := fmt.Sprintf(
		"@reply-parent-msg-id=%s PRIVMSG #%s :%s",
		messageId,
		channel,
		text,
	)

	if err := c.Writer.SendRaw(msg); err != nil {
		fmt.Println("reply error", err)
	}
}

func (c *BotClient) SayWithRateLimiting(channel, text string, replyTo *string) {
	channelLimiter, ok := c.RateLimiters.Channels.Items[strings.ToLower(channel)]
	if !ok {
		return
	}

	if !c.RateLimiters.Global.TryTake() {
		return
	}

	// it should be separately
	if !channelLimiter.Limiter.TryTake() {
		return
	}

	text = strings.ReplaceAll(text, "\n", " ")

	parts := splitTextByLength(text)

	if replyTo != nil {
		for _, part := range parts {
			text = validateResponseSlashes(text)
			c.Reply(channel, part, *replyTo)
		}
	} else {
		for _, part := range parts {
			text = validateResponseSlashes(text)
			c.Say(channel, part)
		}
	}
}

func validateResponseSlashes(response string) string {
	if strings.HasPrefix(response, "/me") || strings.HasPrefix(response, "/announce") {
		return response
	} else if strings.HasPrefix(response, "/") {
		return "Slash commands except /me and /announce is disallowed. This response wont be ever sended."
	} else if strings.HasPrefix(response, ".") {
		return `Message cannot start with "." symbol.`
	} else {
		return response
	}
}

func splitTextByLength(text string) []string {
	var parts []string

	i := 500
	for utf8.RuneCountInString(text) > 0 {
		if utf8.RuneCountInString(text) < 500 {
			parts = append(parts, text)
			break
		}
		runned := []rune(text)
		parts = append(parts, string(runned[:i]))
		text = string(runned[i:])
	}

	return parts
}
