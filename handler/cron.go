package handler

import (
	"context"
	"net/http"
	"strings"
	"time"

	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"
	"github.com/tdaira/hayaoki_bot/sheets"
	"github.com/nlopes/slack"
)

// SlashHandler handles slash message.
type CronHandler struct {
	Ctx            context.Context
	BotClient      *slack.Client
	PostPram       slack.PostMessageParameters
	HayaokiChannel string
	SpreadSheet    *sheets.SpreadSheet
}

// NewSlashHandler create SlashHandler instance.
func NewCronHandler(channelID string) *CronHandler {
	return &CronHandler{HayaokiChannel: channelID}
}

// Run runs SlashHandler.
func (s *CronHandler) Run() {
	http.HandleFunc("/cron", s.handler)
}

func (s *CronHandler) handler(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	s.Ctx = ctx
	log.Infof(ctx, "Receive message.")

	// New spread sheet instance.
	var err error
	s.SpreadSheet, err = sheets.NewSpreadSheet(ctx)
	if err != nil {
		log.Errorf(ctx, err.Error())
		return
	}

	// Get bot token.
	key := datastore.NewKey(ctx, "slack", "bot_token", 0, nil)
	slackToken := new(SlackToken)
	if err := datastore.Get(ctx, key, slackToken); err != nil {
		log.Errorf(ctx, err.Error())
		return
	}

	// New RTM instance for the reply.
	slack.SetHTTPClient(urlfetch.Client(ctx))
	s.BotClient = slack.New(slackToken.Value)
	s.PostPram = slack.PostMessageParameters{
		Username:    "hayaoki_bot",
		AsUser:      true,
		IconURL:     "https://avatars.slack-edge.com/2017-10-07/252871472931_f189dd4ee78316f6cd13_72.png",
	}

	hayaokiMap, isToday, err := s.SpreadSheet.Hayaoki.GetLastResult()
	if err != nil {
		log.Errorf(ctx, err.Error())
		return
	}
	if !isToday {
		log.Infof(ctx, "LatestResult is not today.")
		return
	}
	log.Infof(ctx, "[HayaokiMap]")
	for key, val := range hayaokiMap {
		log.Infof(ctx, key + ": " + val)
	}
	kikenMap, err := s.SpreadSheet.Kiken.GetKikenList()
	if err != nil {
		log.Errorf(ctx, err.Error())
		return
	}
	log.Infof(ctx, "[KikenMap]")
	for key, val := range kikenMap {
		log.Infof(ctx, key + ": " + val)
	}

	// Create penalty string.
	failedUsers := []string{}
	successUsers := []string{}
	for key, val := range hayaokiMap {
		if val == "" {
			kikenStr := ""
			if val, ok := kikenMap[key]; ok {
				kikenStr = val
			}
			contains, err := s.containsToday(kikenStr)
			if err != nil {
				log.Errorf(ctx, err.Error())
				return
			}
			if !contains {
				failedUsers = append(failedUsers, key)
			}
		} else {
			successUsers = append(successUsers, key)
		}
	}

	// Add san.
	for i, user := range failedUsers {
		failedUsers[i] = "<@" + user + ">さん"
	}
	for i, user := range successUsers {
		successUsers[i] = "<@" +  user + ">さん"
	}

	resultStr := strings.Join(failedUsers, ",") + "は" + strings.Join(successUsers, ",") + "に飲み物を提供してください。"
	if len(failedUsers) == 0 || len(successUsers) == 0 {
		resultStr = "本日の飲み物提供はありません。"
	}
	if _, _, err := s.BotClient.PostMessage(
		s.HayaokiChannel,
		resultStr,
		s.PostPram); err != nil {
		log.Errorf(ctx, err.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *CronHandler) containsToday(dateStr string) (bool, error) {
	return (&sheets.KikenSheet{}).ContainsDate(dateStr, time.Now())
}
