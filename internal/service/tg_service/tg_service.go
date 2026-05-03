package tg_service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"myapp/internal/entity"
	"myapp/internal/models"
	"sort"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
)

func (srv *TgService) PerelivVampBots() {
	allVampBots, err := srv.db2.GetAllVampBots()
	if err != nil {
		fmt.Println(fmt.Errorf("PerelivVampBots GetAllVampBots err: %v", err).Error())
	}
	for _, vampBot := range allVampBots {
		// grLink, err := srv.db2.GetGroupLinkById(vampBot.GroupLinkId)
		// if err != nil {
		// 	fmt.Println(fmt.Errorf("PerelivVampBots GetGroupLinkById err: %v", err).Error())
		// }
		// if grLink.Id == 0 {
		// 	fmt.Println(fmt.Errorf("PerelivVampBots GetGroupLinkById err: grLink.Id == 0").Error())
		// 	continue
		// }

		botInfo, err := srv.db.GetBotInfoByToken(vampBot.Token)
		if err != nil {
			fmt.Println(fmt.Errorf("PerelivVampBots GetBotInfoByToken err: %v", err).Error())
		}
		if botInfo.Id != 0 {
			fmt.Println(fmt.Errorf("PerelivVampBots GetBotInfoByToken err: botInfo.Id != 0").Error())
			continue
		}

		// grLink2, err := srv.db.GetGroupLinkByLink(grLink.Link)
		// if err != nil {
		// 	fmt.Println(fmt.Errorf("PerelivVampBots GetGroupLinkByLink err: %v", err).Error())
		// }
		// if grLink2.Id == 0 {
		// 	err = srv.db.AddNewGroupLinkV2(grLink.Title, grLink.Link, grLink.UserCreator)
		// 	if err != nil {
		// 		fmt.Println(fmt.Errorf("PerelivVampBots AddNewGroupLinkV2 err: %v", err).Error())
		// 	}
		// }

		// grLink3, err := srv.db.GetGroupLinkByAllFields(grLink.Title, grLink.Link, grLink.UserCreator)
		// if err != nil {
		// 	fmt.Println(fmt.Errorf("PerelivVampBots GetGroupLinkByAllFields err: %v", err).Error())
		// }
		// if grLink3.Id == 0 {
		// 	fmt.Println(fmt.Errorf("PerelivVampBots GetGroupLinkByAllFields err: grLink3.Id == 0").Error())
		// 	continue
		// }

		err = srv.db.AddNewBot(vampBot.Id, vampBot.Username, vampBot.Firstname, vampBot.Token, 0)
		if err != nil {
			fmt.Println(fmt.Errorf("PerelivVampBots AddNewBot err: %v", err).Error())
		}
		err = srv.db.EditBotGroupLinkId(vampBot.GroupLinkId, vampBot.Id)
		if err != nil {
			fmt.Println(fmt.Errorf("PerelivVampBots EditBotGroupLinkId err: %v", err).Error())
		}

		err = srv.db.EditBotChId(vampBot.ChId, vampBot.Id)
		if err != nil {
			fmt.Println(fmt.Errorf("PerelivVampBots EditBotChId err: %v", err).Error())
		}
		err = srv.db.EditBotChLink(vampBot.ChLink, vampBot.Id)
		if err != nil {
			fmt.Println(fmt.Errorf("PerelivVampBots EditBotChLink err: %v", err).Error())
		}

		err = srv.db.EditBotUserCreator(vampBot.Id, vampBot.UserCreator)
		if err != nil {
			fmt.Println(fmt.Errorf("PerelivVampBots EditBotUserCreator err: %v", err).Error())
		}
		err = srv.db.EditBotLichka(vampBot.Id, vampBot.Lichka)
		if err != nil {
			fmt.Println(fmt.Errorf("PerelivVampBots EditBotLichka err: %v", err).Error())
		}
		err = srv.db.EditBotPersonalLink(vampBot.PersonalLink, vampBot.Id)
		if err != nil {
			fmt.Println(fmt.Errorf("PerelivVampBots EditBotPersonalLink err: %v", err).Error())
		}

		err = srv.db.EditBotDonorChId(vampBot.Id, vampBot.DonorChId)
		if err != nil {
			fmt.Println(fmt.Errorf("PerelivVampBots EditBotDonorChId err: %v", err).Error())
		}
		err = srv.db.EditBotIsErrInStat(vampBot.Id, vampBot.IsErrInStat)
		if err != nil {
			fmt.Println(fmt.Errorf("PerelivVampBots EditBotIsErrInStat err: %v", err).Error())
		}
		// err = srv.db.EditBotToClickShortLink(vampBot.Id, vampBot.ToClickShortLink)
		// if err != nil {
		// 	fmt.Println(fmt.Errorf("PerelivVampBots EditBotToClickShortLink err: %v", err).Error())
		// }
		// err = srv.db.EditBotToClickShortLinkToLichka(vampBot.Id, vampBot.ToClickShortLinkToLichka)
		// if err != nil {
		// 	fmt.Println(fmt.Errorf("PerelivVampBots EditBotToClickShortLinkToLichka err: %v", err).Error())
		// }
		err = srv.db.EditBotShortDomenToReplace(vampBot.Id, vampBot.ShortDomenToReplace)
		if err != nil {
			fmt.Println(fmt.Errorf("PerelivVampBots EditBotShortDomenToReplace err: %v", err).Error())
		}

		time.Sleep(time.Millisecond*100)
	}
}

func (srv *TgService) GetTgBotUpdates() {
	updConf := UpdateConfig{
		Offset:  0,
		Timeout: 30,
		Buffer:  1000,
	}

	updates := srv.GetUpdatesChan(&updConf, srv.Cfg.Token)
	for update := range updates {
		srv.bot_Update(update)
	}
}

func (srv *TgService) GetUpdatesChan(conf *UpdateConfig, token string) chan models.Update {
	UpdCh := make(chan models.Update, conf.Buffer)

	go func() {
		for {
			logMess := fmt.Sprintf(srv.Cfg.TgEndp, token, "getUpdates")
			fmt.Println(logMess)

			updates, err := srv.GetUpdates(conf.Offset, conf.Timeout, token)
			if err != nil {
				fmt.Println("Failed to get updates, retrying in 4 seconds...")
				time.Sleep(time.Second * 4)
				continue
			}

			for _, update := range updates {
				if update.UpdateId >= conf.Offset {
					conf.Offset = update.UpdateId + 1
					UpdCh <- update
				}
			}
		}
	}()

	return UpdCh
}

func (srv *TgService) bot_Update(m models.Update) error {
	srv.l.Info("	NEW TG Update")

	if m.ChannelPost != nil { // on Channel_Post
		go func() {
			err := srv.Donor_HandleChannelPost(m)
			if err != nil {
				srv.l.Error("Donor_HandleChannelPost err", zap.Error(err))
			}
		}()
		return nil
	}

	if m.EditedChannelPost != nil { // on Edited_Channel_Post
		go func() {
			err := srv.Donor_HandleEditedChannelPost(m)
			if err != nil {
				srv.l.Error("Donor_HandleEditedChannelPost err", zap.Error(err))
			}
		}()
		return nil
	}

	if m.CallbackQuery != nil { // on Callback_Query
		go func() {
			err := srv.HandleCallbackQuery(m)
			if err != nil {
				srv.l.Error("HandleCallbackQuery err", zap.Error(err))
			}
		}()
		return nil
	}

	if m.Message != nil && m.Message.ReplyToMessage != nil { // on Reply_To_Message
		go func() {
			err := srv.HandleReplyToMessage(m)
			if err != nil {
				srv.l.Error("HandleReplyToMessage err", zap.Error(err))
			}
		}()
		return nil
	}

	if m.Message != nil && m.Message.Chat != nil { // on Message
		go func() {
			err := srv.HandleMessage(m)
			if err != nil {
				srv.l.Error("HandleMessage err", zap.Error(err))
			}
		}()
		return nil
	}

	return nil
}

func MediaInSlice(s []models.InputMedia, m models.InputMedia) bool {
	for _, v := range s {
		if v.Media == m.Media {
			return true
		}
	}
	return false
}

func MediaInSlice2(s []Media, m Media) bool {
	for _, v := range s {
		if v.File_name_in_server == m.File_name_in_server {
			return true
		}
	}
	return false
}

func (srv *TgService) DeleteLostBots() {
	for {
		time.Sleep(time.Hour * 2)

		donorBot, err := srv.db.GetBotInfoByToken(srv.Cfg.Token)
		if err != nil {
			errMess := fmt.Sprintf("DeleteLostBots: GetBotInfoByToken err: %v", err)
			srv.l.Error(errMess)
			srv.SendMessage(donorBot.ChId, errMess)
		}
		if donorBot.Id == 0 {
			errMess := fmt.Sprintf("DeleteLostBots: GetBotInfoByToken err: donorBot.Id == 0")
			srv.l.Error(errMess)
			srv.SendMessage(donorBot.ChId, errMess)
		}

		allBots, err := srv.db.GetAllBots()
		if err != nil {
			errMess := fmt.Sprintf("DeleteLostBots: GetAllBots err: %v", err)
			srv.l.Error(errMess)
			srv.SendMessage(donorBot.ChId, errMess)
		}
		if len(allBots) == 0 {
			errMess := fmt.Sprintf("DeleteLostBots: GetAllBots err: len(allBots) == 0")
			srv.l.Error(errMess)
			srv.SendMessage(donorBot.ChId, errMess)
		}

		for _, bot := range allBots {
			if bot.IsDonor == 1 {
				continue
			}
			resp, err := srv.GetMe(bot.Token)
			if err != nil {
				errMess := fmt.Sprintf("DeleteLostBots: getBotByToken token-%s err: %v", bot.Token, err)
				srv.l.Error(errMess)
				srv.SendMessage(donorBot.ChId, errMess)
			}
			if resp.ErrorCode == 401 && resp.Description == "Unauthorized" {
				srv.db.DeleteBot(bot.Id)

				var mess bytes.Buffer
				mess.WriteString("удален бот без доступа\n")
				mess.WriteString(fmt.Sprintf("бот: @%s | %s\n", bot.Username, bot.Token))
				mess.WriteString(fmt.Sprintf("канал: %d | %s\n", bot.ChId, bot.ChLink))
				logMess := mess.String()

				srv.SendMessage(donorBot.ChId, logMess)
				time.Sleep(time.Second)
			}
		}
	}
}

func (srv *TgService) InsertGrabberBot() {
	time.Sleep(time.Second * 4)

	bots, err := srv.db.GetAllBots()
	if err != nil {
		err = fmt.Errorf("InsertGrabberBot GetAllBots err: %v", err)
		srv.l.Error(err.Error())
		return
	}
	if len(bots) > 0 {
		return
	}

	grabberBot, err := srv.GetMe(srv.Cfg.Token)
	if err != nil {
		err = fmt.Errorf("InsertGrabberBot GetMe err: %v", err)
		srv.l.Error(err.Error())
		return
	}
	err = srv.db.AddNewBot(
		grabberBot.Result.Id,
		grabberBot.Result.UserName,
		grabberBot.Result.FirstName,
		srv.Cfg.Token,
		1,
	)
	if err != nil {
		err = fmt.Errorf("InsertGrabberBot AddNewBot err: %v", err)
		srv.l.Error(err.Error())
		return
	}
	donorBotInfo, err := srv.db.GetBotInfoById(grabberBot.Result.Id)
	if err != nil {
		err = fmt.Errorf("InsertGrabberBot GetBotInfoById err: %v", err)
		srv.l.Error(err.Error())
		return
	}
	if donorBotInfo.ChId == 0 {
		err = srv.db.EditBotChId(srv.Cfg.BotChId, grabberBot.Result.Id)
		if err != nil {
			err = fmt.Errorf("InsertGrabberBot EditBotField err: %v", err)
			srv.l.Error(err.Error())
			return
		}
	}
	if donorBotInfo.ChLink == "" {
		err = srv.db.EditBotChLink(srv.Cfg.BotChLink, grabberBot.Result.Id)
		if err != nil {
			err = fmt.Errorf("InsertGrabberBot EditBotField err: %v", err)
			srv.l.Error(err.Error())
			return
		}
	}
}

func (srv *TgService) AlertScamBots() {
	for {
		time.Sleep(time.Hour * 6)

		donorBot, err := srv.db.GetBotInfoByToken(srv.Cfg.Token)
		if err != nil {
			errMess := fmt.Sprintf("AlertScamBots: GetBotInfoByToken err: %v", err)
			srv.l.Error(errMess)
			srv.SendMessage(donorBot.ChId, errMess)
		}
		if donorBot.Id == 0 {
			errMess := fmt.Sprintf("AlertScamBots: GetBotInfoByToken err: donorBot.Id == 0")
			srv.l.Error(errMess)
			srv.SendMessage(donorBot.ChId, errMess)
		}

		allBots, err := srv.db.GetAllBots()
		if err != nil {
			errMess := fmt.Sprintf("AlertScamBots: GetAllBots err: %v", err)
			srv.l.Error(errMess)
			srv.SendMessage(donorBot.ChId, errMess)
		}
		if len(allBots) == 0 {
			errMess := fmt.Sprintf("AlertScamBots: GetAllBots err: len(allBots) == 0")
			srv.l.Error(errMess)
			srv.SendMessage(donorBot.ChId, errMess)
		}

		for _, bot := range allBots {
			if bot.IsDonor == 1 || bot.ChIsSkam == 1 {
				continue
			}
			resp, err := srv.GetChat(bot.ChId, bot.Token)
			if err != nil {
				errMess := fmt.Sprintf("AlertScamBots: GetChat token-%s err: %v", bot.Token, err)
				srv.l.Error(errMess)
				srv.SendMessage(donorBot.ChId, errMess)
				var logBotMess bytes.Buffer
				logBotMess.WriteString("удален бот\n")
				logBotMess.WriteString(fmt.Sprintf("Донор псевдоним: %v\n", srv.CreateCodeFmt(srv.Cfg.BotPrefix)))
				logBotMess.WriteString(fmt.Sprintf("%v\n", srv.AddAt(bot.Username)))
				logBotMess.WriteString(fmt.Sprintf("%v\n", bot.Token))
				logBotMess.WriteString(fmt.Sprintf("%v\n", bot.ChLink))
				logBotMess.WriteString(fmt.Sprintf("%d\n", bot.ChId))
				grLink, _ := srv.db.GetGroupLinkById(bot.GroupLinkId)
				logBotMess.WriteString(fmt.Sprintf("group_link: %d, %v - %v\n", bot.GroupLinkId, grLink.Title, grLink.Link))
				srv.SendMessage(donorBot.ChId, logBotMess.String())
				if srv.Cfg.BotPrefix != "_test" { // стата в общий канал
					srv.SendMessageByToken(srv.Cfg.ChForStat, logBotMess.String(), srv.Cfg.BotTokenForStat)
				}
				// srv.db.DeleteBot(bot.Id)
			}
			if strings.Contains(resp.Result.Description, "this account as a scam or a fake") {
				var mess bytes.Buffer
				mess.WriteString("обнаружен скам на канале\n")
				mess.WriteString(fmt.Sprintf("Донор псевдоним: %v\n", srv.CreateCodeFmt(srv.Cfg.BotPrefix)))
				mess.WriteString(fmt.Sprintf("бот: @%v | %v\n", bot.Username, bot.Token))
				mess.WriteString(fmt.Sprintf("канал: %v | %v\n", bot.ChLink, bot.ChId))
				logMess := mess.String()

				srv.SendMessage(donorBot.ChId, logMess)
				srv.db.EditBotChIsSkam(bot.Id, 1)
				if srv.Cfg.BotPrefix != "_test" { // стата в общий канал
					srv.SendMessageByToken(srv.Cfg.ChForStat, mess.String(), srv.Cfg.BotTokenForStat)
				}

				time.Sleep(time.Second)
			}
		}
	}
}

func (srv *TgService) AcceptChPostByAdmin() {
	mediaArr := make([]Media, 0)
	for {
		select {
		case x, ok := <-srv.MediaCh:
			if ok {
				okk := MediaInSlice2(mediaArr, x)
				if !okk {
					mediaArr = append(mediaArr, x)
				}
			} else {
				srv.l.Error("AcceptChPostByAdmin closed!")
				return
			}
		case <-time.After(time.Second * 15):
			if len(mediaArr) == 0 {
				continue
			}
			if len(mediaArr) == 1 {
				srv.l.Error("AcceptChPostByAdmin len(mediaArr) == 1")
				continue
			}

			sort.Slice(mediaArr, func(i, j int) (less bool) { //сортировка по MessageId
				return mediaArr[i].MessageId < mediaArr[j].MessageId
			})

			mediaGroupId := mediaArr[0].Media_group_id
			srv.MediaStore.MediaGroups[mediaGroupId] = mediaArr

			arrsik := make([]models.InputMedia, 0)
			for _, med := range mediaArr {
				nwmd := models.InputMedia{
					Type:            med.Type_media,
					Media:           med.File_id,
					Caption:         med.Caption,
					CaptionEntities: med.Caption_entities,
				}
				ok := MediaInSlice(arrsik, nwmd)
				if !ok {
					arrsik = append(arrsik, nwmd)
				}
			}

			donorBot, err := srv.db.GetBotInfoByToken(srv.Cfg.Token)
			if err != nil {
				srv.l.Error(fmt.Sprintf("AcceptChPostByAdmin: GetBotInfoByToken token-%s err: %v", srv.Cfg.Token, err))
			}

			acceptMess := map[string]any{
				"chat_id": strconv.Itoa(donorBot.ChId),
				"media":   arrsik,
			}
			if mediaArr[0].Reply_to_message_id != 0 {
				acceptMess["reply_to_message_id"] = mediaArr[0].Reply_to_message_id
			}
			media_json, err := json.Marshal(acceptMess)
			if err != nil {
				srv.l.Error(fmt.Sprintf("AcceptChPostByAdmin: json.Marshal(acceptMess) err: %v", err))
			}
			err = srv.sendData(media_json, "sendMediaGroup")
			if err != nil {
				srv.l.Error(fmt.Sprintf("AcceptChPostByAdmin: sendData(sendMediaGroup) err: %v", err))
			}

			cfgVal, _ := srv.db.GetCfgValById(entity.Auto_acc_media_gr_CfgId)
			if cfgVal.Val == "1" {
				m := models.Update{
					CallbackQuery: &models.CallbackQuery{
						From: models.User{Id: 0, UserName: "auto"},
					},
				}
				srv.CQ_accept_ch_post_by_admin(m, mediaGroupId)
			} else {
				media_json, err = json.Marshal(map[string]any{
					"chat_id":      strconv.Itoa(donorBot.ChId),
					"text":         "подтвердите сообщение сверху",
					"reply_markup": fmt.Sprintf(`{ "inline_keyboard" : [[{ "text": "разослать по каналам", "callback_data": "accept_ch_post_%s_by_admin" }]] }`, mediaGroupId),
				})
				if err != nil {
					srv.l.Error(fmt.Sprintf("AcceptChPostByAdmin: Marshal media_json err: %v", err))
				}
				err = srv.sendData(media_json, "sendMessage")
				if err != nil {
					srv.l.Error(fmt.Sprintf("AcceptChPostByAdmin: sendData(sendMessage) err: %v", err))
				}
			}

			mediaArr = mediaArr[0:0]
		}
	}
}