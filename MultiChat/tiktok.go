package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	golive "github.com/PirateTok/live-go"
	"github.com/PirateTok/live-go/events"
	pb "github.com/PirateTok/live-go/proto"
)

// TikTok LIVE Webcast is an unofficial, public-only protocol.
// No session cookies, passwords or paid signing services are used.
func parseTikTokChat(msg *pb.WebcastChatMessage) (Message, bool) {
	if msg == nil || strings.TrimSpace(msg.Content) == "" {
		return Message{}, false
	}
	var user, id string
	if msg.User != nil {
		user = msg.User.Nickname
		if user == "" {
			user = msg.User.UniqueId
		}
	}
	if msg.Common != nil && msg.Common.MsgId > 0 {
		id = fmt.Sprint(msg.Common.MsgId)
	}
	return Message{ID: id, Platform: "tiktok", User: user, Text: msg.Content, Kind: "chat"}, true
}
func tiktokLoop(ctx context.Context, c TikTokConfig, h *Hub, logf func(string, ...any)) {
	delay := 5 * time.Second
	for ctx.Err() == nil {
		feed, err := golive.NewClient(c.Username).MaxRetries(10).Connect(ctx)
		if err != nil {
			logf("[tiktok] offline, blocked or unavailable; retrying without credentials")
			if !sleepContext(ctx, delay) {
				return
			}
			if delay < time.Minute {
				delay *= 2
			}
			continue
		}
		delay = 5 * time.Second
		logf("[tiktok] connected to public LIVE chat")
		running := true
		for running && ctx.Err() == nil {
			select {
			case <-ctx.Done():
				return
			case evt, ok := <-feed:
				if !ok {
					running = false
					break
				}
				switch evt.Type {
				case events.EventChat:
					chat, ok := evt.Data.(*pb.WebcastChatMessage)
					if ok {
						if m, valid := parseTikTokChat(chat); valid {
							h.publish(m)
						}
					}
				case events.EventLiveEnded:
					running = false
				}
			}
		}
		if !sleepContext(ctx, delay) {
			return
		}
	}
}
