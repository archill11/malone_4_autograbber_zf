package tg_service

import (
	"fmt"
	"myapp/internal/models"
	"time"

	"go.uber.org/zap"
)

func (srv *TgService) HandlePinnedMessage(m models.Update) error {
	var pmP *models.Message
	var fromId int
	var fromUsername string
	var pmMessageId int

	if m.Message != nil && m.Message.PinnedMessage != nil {
		pmP = m.Message.PinnedMessage
		fromId = m.Message.From.Id
		fromUsername = m.Message.From.UserName
		pmMessageId = pmP.MessageId
	} else if m.ChannelPost != nil && m.ChannelPost.PinnedMessage != nil {
		pmP = m.ChannelPost.PinnedMessage
		fromId = m.ChannelPost.From.Id
		fromUsername = m.ChannelPost.From.UserName
		pmMessageId = pmP.MessageId
	} else {
		return fmt.Errorf("HandlePinnedMessage: no pinned message found")
	}

	srv.l.Info(fmt.Sprintf("HandlePinnedMessage: fromId: %d, fromUsername: %s, pmMessageId: %d",
		fromId, fromUsername, pmMessageId),
	)

	allVampBots, err := srv.db.GetAllVampBots()
	if err != nil {
		return err
	}
	augmentedAllVampBots := srv.GetAugmentedVampBots(allVampBots)
	for i, vampBot := range augmentedAllVampBots {
		if vampBot.ChId == 0 {
			continue
		}

		currPosts, err := srv.db.GetPostsByDonorIdAndChId(pmMessageId, vampBot.ChId)
		if err != nil {
			return fmt.Errorf("editChPostAsVamp GetPostsByDonorIdAndChId err: %v", err)
		}
		for _, currPost := range currPosts {
			messageForDelete := currPost.PostId
			srv.PinChatMessage(vampBot.ChId, messageForDelete, vampBot.Token)
		}
		srv.l.Info("HandlePinnedMessage",
			zap.Any("bot index in arr", i),
			zap.Any("bot ch link", vampBot.ChLink),
			zap.Any("currPosts", currPosts),
			zap.Any("pmMessageId", pmMessageId),
		)
		time.Sleep(time.Millisecond*600)
	}

	return nil
}

