package tg_service

import (
	"fmt"
	"myapp/internal/models"
	"time"

	"go.uber.org/zap"
)

func (srv *TgService) HandlePinnedMessage(m models.Update) error {
	pm := m.Message.PinnedMessage
	pmMessageId := pm.MessageId
	fromUsername := m.Message.From.UserName
	fromId := m.Message.From.Id
	srv.l.Info(fmt.Sprintf("HandlePinnedMessage: fromId: %d, fromUsername: %s, pmMessageId: %d",
		fromId, fromUsername, pmMessageId),
	)

	allVampBots, err := srv.db.GetAllVampBots()
	if err != nil {
		return err
	}
	for i, vampBot := range allVampBots {
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

