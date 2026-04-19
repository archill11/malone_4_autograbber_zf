package tg_service

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"myapp/internal/entity"
	"myapp/internal/models"
	"myapp/pkg/files"
	"myapp/pkg/mycopy"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

func (srv *TgService) Donor_HandleChannelPost(m models.Update) error {
	fromId := m.ChannelPost.Chat.Id
	srv.l.Info("Donor_HandleChannelPost", zap.Any("models.Update", m))

	channel_id := m.ChannelPost.Chat.Id
	grabberBot, _ := srv.db.GetBotInfoByToken(srv.Cfg.Token)
	if channel_id != grabberBot.ChId {
		srv.l.Error("Donor_HandleChannelPost channel_id != grabberBot.ChId", zap.Any("channel_id", channel_id), zap.Any("grabberBot.ChId", grabberBot.ChId))
		return nil
	}

	if strings.HasPrefix(m.ChannelPost.Text, "Донор псевдоним") || strings.HasPrefix(m.ChannelPost.Text, "ок, начинаю рассылку") {
		return nil
	}

	err := srv.Donor_addChannelPost(m)
	if err != nil {
		srv.SendMessage(fromId, ERR_MSG)
		srv.SendMessage(fromId, err.Error())
		return err
	}
	return nil
}

func (srv *TgService) Donor_addChannelPost(m models.Update) error {
	message_id := m.ChannelPost.MessageId
	channel_id := m.ChannelPost.Chat.Id

	// Проверка что пост есть уже в базе нужна для того что бы телега не отрпавляла
	// кучу запросов повторно , тк ответ долгий из за рассылки
	post, err := srv.db.GetPostByDonorIdAndChId(message_id, channel_id)
	if err != nil {
		return fmt.Errorf("Donor_addChannelPost GetPostByDonorIdAndChId err: %v", err)
	}
	if post.PostId != 0 {
		srv.l.Info("пост уже есть в БД, валим!")
		return nil
	}
	// добавили пост в БД
	srv.db.AddNewPost(channel_id, message_id, message_id, "")

	// если Media_Group
	if m.ChannelPost.MediaGroupId != nil {
		var postType string
		if len(m.ChannelPost.Photo) > 0 {
			postType = "photo"
		} else if m.ChannelPost.Video.FileId != "" {
			postType = "video"
		} else {
			return fmt.Errorf("Media_Group без photo и video")
		}
		filePath, filePathAugmented, err := srv.downloadPostMediaV2(m, postType)
		if err != nil {
			return fmt.Errorf("Donor_addChannelPost downloadPostMedia err: %v", err)
		}
		newmedia := Media{
			Media_group_id:            *m.ChannelPost.MediaGroupId,
			Type_media:                postType,
			File_name_in_server:       filePath,
			File_name_in_server_augmented: filePathAugmented,
			Donor_message_id:          message_id,
			Reply_to_donor_message_id: 0,
			Caption:                   "",
			Caption_entities:          m.ChannelPost.CaptionEntities,
			MessageId:                 m.ChannelPost.MessageId,
			//File_id:               // нужно для подтверждения в доноре, позже в вампирах заменяем
			//Reply_to_message_id:  // нужно для подтверждения в доноре, позже в вампирах заменяем
		}
		if postType == "photo" {
			newmedia.File_id = m.ChannelPost.Photo[len(m.ChannelPost.Photo)-1].FileId
		} else if postType == "video" {
			newmedia.File_id = m.ChannelPost.Video.FileId
		}
		if m.ChannelPost.ReplyToMessage != nil {
			newmedia.Reply_to_message_id = m.ChannelPost.ReplyToMessage.MessageId
			newmedia.Reply_to_donor_message_id = m.ChannelPost.ReplyToMessage.MessageId
		}
		if m.ChannelPost.Caption != nil {
			newmedia.Caption = *m.ChannelPost.Caption
		}

		srv.MediaCh <- newmedia
		return nil
	}

	// если не Media_Group
	allVampBots, err := srv.db.GetAllVampBots()
	if err != nil {
		return fmt.Errorf("Donor_addChannelPost GetAllVampBots err: %v", err)
	}
	
	var okSend int
	var notOkSend int
	var IsDisable int
	var ChId0 int
	refkiMap := map[int]map[string]int{}
	for _, vampBot := range allVampBots {
		botRefka := vampBot.GroupLinkId
		refkiMap[botRefka] = map[string]int{
			"Успешно": 0,
			"Неуспешно": 0,
		}
	}

	srv.db.EditCfgVal("is-sending-now", "1")
	defer func() {
		srv.db.EditCfgVal("is-sending-now", "0")
	}()

	postUUID, _ := uuid.NewV7()

	errorLinks := make([]string, 0)

	for i, vampBot := range allVampBots {
		botRefka := vampBot.GroupLinkId
		if srv.Cfg.IsMultiGrabber == 1 && vampBot.DonorChId != 0 && vampBot.DonorChId != channel_id {
			continue // бот не подвязан к этому донор каналу
		}
		if vampBot.ChId == 0 {
			ChId0++
			continue
		}
		if vampBot.IsDisable == 1 {
			IsDisable++
			continue
		}

		srv.l.Info("______________________________________")
		srv.l.Info(
			"Donor_addChannelPost",
			zap.Any("bot index in arr", i),
			zap.Any("arr len", len(allVampBots)),
			zap.Any("bot ch link", vampBot.ChLink),
			zap.Any("postUUID", postUUID),
		)

		err, errLink := srv.sendChPostAsVamp(vampBot, m)
		if errLink != "" {
			errorLinks = append(errorLinks, errLink)
		}
		if err != nil {
			notOkSend++
			_, ok := refkiMap[botRefka]
			if ok {
				refkiMap[botRefka]["Неуспешно"] = refkiMap[botRefka]["Неуспешно"]+1
			}
			srv.l.Error("Donor_addChannelPost: sendChPostAsVamp err", zap.Error(err))
			if strings.Contains(err.Error(), "Bad Request: invalid file_id") {
				srv.SendMessage(channel_id, err.Error())
				srv.l.Info("Donor_addChannelPost: end ERROR")
				return nil
			}
		} else {
			okSend++
			_, ok := refkiMap[botRefka]
			if ok {
				refkiMap[botRefka]["Успешно"] = refkiMap[botRefka]["Успешно"]+1
			}
		}
		time.Sleep(time.Millisecond*1300)
	}
	srv.l.Info("Donor_addChannelPost: end")

	donorBot, _ := srv.db.GetBotInfoByToken(srv.Cfg.Token)


	var reportMess bytes.Buffer
	reportMess.WriteString(fmt.Sprintf("Отчет по посту:\n"))
	reportMess.WriteString(fmt.Sprintf("Донор псевдоним: %v\n", srv.Cfg.BotPrefix))
	reportMess.WriteString(fmt.Sprintf("Бот: %v\n", srv.AddAt(donorBot.Username)))
	reportMess.WriteString(fmt.Sprintf("Пост: https://t.me/c/%v/%v\n", srv.Delete100(channel_id), message_id))
	reportMess.WriteString(fmt.Sprintf("uuid поста в логах: %v\n", postUUID))
	reportMess.WriteString(fmt.Sprintf("Всего каналов: %v\n", len(allVampBots)))
	reportMess.WriteString(fmt.Sprintf("Успешно отправлено: %v\n", okSend))
	if notOkSend != 0 {
		reportMess.WriteString(fmt.Sprintf("Неуспешно: %v\n", notOkSend))
	}
	if ChId0 != 0 {
		reportMess.WriteString(fmt.Sprintf("Без подвяз. канала: %v\n", ChId0))
	}
	if IsDisable != 0 {
		reportMess.WriteString(fmt.Sprintf("Отключены от рассылки: %v\n", IsDisable))
	}

	var reportMessErrorLinks bytes.Buffer
	reportMessErrorLinks.WriteString(fmt.Sprintf("Донор псевдоним: %v\n", srv.Cfg.BotPrefix))
	reportMessErrorLinks.WriteString(fmt.Sprintf("uuid поста в логах: %v\n", postUUID))
	reportMessErrorLinks.WriteString(fmt.Sprintf("Список ошибок:\n"))
	if len(errorLinks) > 0 {
		for i, v := range errorLinks {
			reportMessErrorLinks.WriteString(fmt.Sprintf("%v) %v\n", i+1, v))
	
			if i%15 == 0 && i > 0 {
				sendMessageResp, err := srv.SendMessageByTokenV2(srv.Cfg.ChForStat, reportMessErrorLinks.String(), srv.Cfg.BotTokenForStat)
				if err != nil {
					srv.l.Error(fmt.Sprintf("Donor_addChannelPost SendMessageByToken err: %v", err))
				}
				if sendMessageResp.Result.MessageId != 0 {
					errLinks := fmt.Sprintf("https://t.me/c/%v/%v", srv.Delete100(srv.Cfg.ChForStat), sendMessageResp.Result.MessageId)
					reportMess.WriteString(fmt.Sprintf("Список Ошибок: %v\n", errLinks))
				}
				reportMessErrorLinks.Reset()
			}
		}
		srv.SendMessageByToken(srv.Cfg.ChForStat, reportMessErrorLinks.String(), srv.Cfg.BotTokenForStat)
	}

	srv.SendMessage(channel_id, reportMess.String())
	if srv.Cfg.BotPrefix != "_test"  { // стата в общий канал
		srv.SendMessageByToken(srv.Cfg.ChForStat, reportMess.String(), srv.Cfg.BotTokenForStat)
	}

	var reportMess2 bytes.Buffer
	if len(refkiMap) > 0 {
		reportMess2.WriteString(fmt.Sprintf("Донор псевдоним: %v\n", srv.Cfg.BotPrefix))
		reportMess2.WriteString(fmt.Sprintf("uuid поста в логах: %v\n", postUUID))
	}
	for key, val := range refkiMap {
		grLinkName, _ := srv.db.GetGroupLinkById(key)

		reportMess2.WriteString(fmt.Sprintf("Реф: %v\n", grLinkName.Title))
		reportMess2.WriteString(fmt.Sprintf("✅%v/%v❌\n", val["Успешно"], val["Неуспешно"]))
		reportMess2.WriteString("\n")
	}
	if srv.Cfg.BotPrefix != "_test"  { // стата в общий канал
		srv.SendMessageByToken(srv.Cfg.ChForStat, reportMess2.String(), srv.Cfg.BotTokenForStat)
	}

	return nil
}

func (srv *TgService) sendChPostAsVamp(vampBot entity.Bot, m models.Update) (error, string) {
	donor_ch_mes_id := m.ChannelPost.MessageId

	var errLink string

	//////////////// если кружочек
	if m.ChannelPost.VideoNote != nil {
		err := srv.sendChPostAsVamp_VideoNote(vampBot, m)
		return err, errLink
	}
	//////////////// если фото
	if len(m.ChannelPost.Photo) > 0 {
		err, errLink := srv.sendChPostAsVamp_Video_or_Photo(vampBot, m, "photo")
		return err, errLink
	}
	//////////////// если видео
	if m.ChannelPost.Video != nil {
		err, errLink := srv.sendChPostAsVamp_Video_or_Photo(vampBot, m, "video")
		return err, errLink
	}
	//////////////// если гифка
	if m.ChannelPost.Animation != nil {
		err, errLink := srv.sendChPostAsVamp_Video_or_Photo(vampBot, m, "animation")
		return err, errLink
	}
	//////////////// если голосовое
	if m.ChannelPost.Voice != nil {
		err, errLink := srv.sendChPostAsVamp_Video_or_Photo(vampBot, m, "voice")
		return err, errLink
	}

	//////////////// если просто текст
	futureMesJson := map[string]any{
		"chat_id": strconv.Itoa(vampBot.ChId),
		"disable_web_page_preview": true,
	}
	if m.ChannelPost.ReplyToMessage != nil {
		replToDonorChPostId := m.ChannelPost.ReplyToMessage.MessageId
		currPost, err := srv.db.GetPostsByDonorIdAndChId_Max(replToDonorChPostId, vampBot.ChId) // тут
		if err != nil {
			return fmt.Errorf("sendChPostAsVamp GetPostsByDonorIdAndChId_Max err: %v", err), errLink
		}
		futureMesJson["reply_to_message_id"] = currPost.PostId
	}
	if m.ChannelPost.ReplyMarkup != nil {
		var inlineKeyboardMarkup models.InlineKeyboardMarkup
		mycopy.DeepCopy(m.ChannelPost.ReplyMarkup, &inlineKeyboardMarkup)

		newInlineKeyboardMarkup, err := srv.PrepareReplyMarkup(inlineKeyboardMarkup, vampBot)
		if err != nil {
			return fmt.Errorf("sendChPostAsVamp PrepareReplyMarkup err: %v", err), errLink
		}
		futureMesJson["reply_markup"] = newInlineKeyboardMarkup
	}

	var messText string                            // строка в которую скопируем значение текста поста, тк структуры копируются по ебаной ссылке, и если срезаем часть текста то потом везде так будет
	mycopy.DeepCopy(m.ChannelPost.Text, &messText) // какого хуя в Го структуры копируются по ссылке ?
	// TODO надо посмотреть на указатели , возможно из за этого 
    // messText := m.ChannelPost.Text
    // fmt.Println(&messText)
    // fmt.Println(&m.ChannelPost.Text)

	if len(m.ChannelPost.Entities) > 0 {
		entities := make([]models.MessageEntity, 0)
		mycopy.DeepCopy(m.ChannelPost.Entities, &entities)
		sourceMessText := messText

		if srv.Cfg.IsGptText == 1 {
			messText = srv.ReplaceSymbolsOrApenAI(messText)
		}

		newEntities, newMessText, err := srv.PrepareEntities(entities, sourceMessText, messText, vampBot)
		if err != nil {
			return fmt.Errorf("sendChPostAsVamp PrepareEntities err: %v", err), errLink
		}
		messText = newMessText
		if newEntities != nil {
			futureMesJson["entities"] = newEntities
		}
	} else {
		sourceMessText := messText
		if srv.Cfg.IsGptText == 1 {
			messText = srv.ReplaceSymbolsOrApenAI(messText)
		}
		_, newMessText, err := srv.PrepareEntities(nil, sourceMessText, messText, vampBot)
		if err != nil {
			return fmt.Errorf("sendChPostAsVamp PrepareEntities 2 err: %v", err), errLink
		}
		messText = newMessText
		
	}
	futureMesJson["text"] = messText

	json_data, err := json.Marshal(futureMesJson)
	if err != nil {
		return fmt.Errorf("sendChPostAsVamp Marshal futureMesJson err: %v", err), errLink
	}
	srv.l.Info("sendChPostAsVamp -> если просто текст -> http.Post", zap.Any("futureMesJson", futureMesJson), zap.Any("string(json_data)", string(json_data)))
	sendVampPostResp, err := srv.MyHttpPost(
		fmt.Sprintf(srv.Cfg.TgEndp, vampBot.Token, "sendMessage"),
		"application/json",
		bytes.NewBuffer(json_data),
	)
	srv.l.Info("sendChPostAsVamp -> если просто текст -> http.Post after",)
	if err != nil {
		srv.l.Info("sendChPostAsVamp -> если просто текст -> http.Post after err != nil", zap.Error(err))
		dbErr := srv.db.AddNewTgError(vampBot.Id, vampBot.Token, vampBot.Username, vampBot.ChId, err.Error())
		if dbErr != nil {
			srv.l.Error("sendChPostAsVamp AddNewTgError dbErr", zap.Error(dbErr))
		}
		srv.l.Info("sendChPostAsVamp -> если просто текст -> http.Post after err != nil after AddNewTgError", zap.Error(err))

		reportMess := bytes.Buffer{}
		reportMess.WriteString(fmt.Sprintf("Донор псевдоним: %v\n", srv.Cfg.BotPrefix))
		reportMess.WriteString(fmt.Sprintf("Ошибка при попытке отправки поста в канал\n\n"))
		reportMess.WriteString(fmt.Sprintf("err: %v \n\n", err.Error()))
		reportMess.WriteString(fmt.Sprintf("bot: %v | %v\n", srv.AddAt(vampBot.Username), vampBot.Token))
		reportMess.WriteString(fmt.Sprintf("ch link: %v\n", vampBot.ChLink))
		gr, _ := srv.db.GetGroupLinkById(vampBot.GroupLinkId)
		reportMess.WriteString(fmt.Sprintf("группа-ссылка: %v - %v\n", vampBot.GroupLinkId, gr.Title))

		sendMessageResp, err2 := srv.SendMessageByTokenV2(srv.Cfg.ChForStatErrors, reportMess.String(), srv.Cfg.BotTokenForStat)
		if err2 != nil {
			srv.l.Warn("sendChPostAsVamp SendMessageByTokenV2 err2",
				zap.Error(err2),
				zap.Any("reportMess", reportMess.String()),
				zap.Any("ChForStatErrors", srv.Cfg.ChForStatErrors),
				zap.Any("BotTokenForStat", srv.Cfg.BotTokenForStat),
			)
		}
		if sendMessageResp.Result.MessageId != 0 {
			errLink = fmt.Sprintf("https://t.me/c/%v/%v", srv.Delete100(srv.Cfg.ChForStatErrors), sendMessageResp.Result.MessageId)
		}

		return fmt.Errorf("sendChPostAsVamp Post err: %v", err), errLink
	}
	defer sendVampPostResp.Body.Close()

	srv.l.Info("sendChPostAsVamp -> если просто текст -> http.Post after defer sendVampPostResp.Body.Close()",)

	var cAny struct {
		models.BotErrResp
		Result struct {
			MessageId int    `json:"message_id"`
			Caption   string `json:"caption"`
		} `json:"result"`
	}
	if err := json.NewDecoder(sendVampPostResp.Body).Decode(&cAny); err != nil {
		return fmt.Errorf("sendChPostAsVamp Decode err: %v", err), errLink
	}
	if cAny.ErrorCode != 0 {
		dbErr := srv.db.AddNewTgError(vampBot.Id, vampBot.Token, vampBot.Username, vampBot.ChId, cAny.BotErrResp.Description)
		if dbErr != nil {
			srv.l.Error("sendChPostAsVamp AddNewTgError dbErr", zap.Error(dbErr))
		}

		reportMess := bytes.Buffer{}
		reportMess.WriteString(fmt.Sprintf("Донор псевдоним: %v\n", srv.Cfg.BotPrefix))
		reportMess.WriteString(fmt.Sprintf("Ошибка при отправке поста в канал\n\n"))
		reportMess.WriteString(fmt.Sprintf("err: %v | %v\n\n", cAny.BotErrResp.ErrorCode, cAny.BotErrResp.Description))
		reportMess.WriteString(fmt.Sprintf("bot: %v | %v\n", srv.AddAt(vampBot.Username), vampBot.Token))
		reportMess.WriteString(fmt.Sprintf("ch link: %v\n", vampBot.ChLink))
		gr, _ := srv.db.GetGroupLinkById(vampBot.GroupLinkId)
		reportMess.WriteString(fmt.Sprintf("группа-ссылка: %v - %v\n", vampBot.GroupLinkId, gr.Title))

		sendMessageResp, err := srv.SendMessageByTokenV2(srv.Cfg.ChForStatErrors, reportMess.String(), srv.Cfg.BotTokenForStat)
		if err != nil {
			srv.l.Warn("sendChPostAsVamp SendMessageByTokenV2 err",
				zap.Error(err),
				zap.Any("reportMess", reportMess.String()),
				zap.Any("ChForStatErrors", srv.Cfg.ChForStatErrors),
				zap.Any("BotTokenForStat", srv.Cfg.BotTokenForStat),
			)
		}
		if sendMessageResp.Result.MessageId != 0 {
			errLink = fmt.Sprintf("https://t.me/c/%v/%v", srv.Delete100(srv.Cfg.ChForStatErrors), sendMessageResp.Result.MessageId)
		}


		if (srv.Cfg.BotPrefix == "_noviy" && vampBot.GroupLinkId == 2) || (srv.Cfg.BotPrefix == "_upravru2" && vampBot.GroupLinkId == 12) {
			if vampBot.IsErrInStat == 0 {
				err = srv.SendMessageByToken(-1002512374528, reportMess.String(), srv.Cfg.BotTokenForStat)
				if err != nil {
					srv.l.Error("sendChPostAsVamp SendMessageByToken err",
						zap.Error(err),
						zap.Any("reportMess", reportMess.String()),
					)
				}

				srv.db.EditBotIsErrInStat(vampBot.Id, 1)
			}
		}

		errMess := fmt.Errorf("sendChPostAsVamp Post ErrorResp: %+v", cAny)
		return errMess, errLink
	}
	if cAny.Result.MessageId != 0 {
		err = srv.db.AddNewPost(vampBot.ChId, cAny.Result.MessageId, donor_ch_mes_id, cAny.Result.Caption)
		if err != nil {
			return fmt.Errorf("sendChPostAsVamp AddNewPost err: %v", err), errLink
		}
	}

	return nil, errLink
}

func (srv *TgService) sendChPostAsVamp_VideoNote(vampBot entity.Bot, m models.Update) error {
	donor_ch_mes_id := m.ChannelPost.MessageId
	futureVideoNoteJson := map[string]string{
		"chat_id": strconv.Itoa(vampBot.ChId),
	}
	if m.ChannelPost.ReplyToMessage != nil {
		replToDonorChPostId := m.ChannelPost.ReplyToMessage.MessageId
		currPost, err := srv.db.GetPostByDonorIdAndChId(replToDonorChPostId, vampBot.ChId)
		if err != nil {
			return fmt.Errorf("sendChPostAsVamp_VideoNote GetPostByDonorIdAndChId err: %v", err)
		}
		futureVideoNoteJson["reply_to_message_id"] = strconv.Itoa(currPost.PostId)
	}
	var newInlineKeyboardMarkupForSupergroup models.InlineKeyboardMarkup
	if m.ChannelPost.ReplyMarkup != nil {
		var inlineKeyboardMarkup models.InlineKeyboardMarkup
		mycopy.DeepCopy(m.ChannelPost.ReplyMarkup, &inlineKeyboardMarkup)

		newInlineKeyboardMarkup, err := srv.PrepareReplyMarkup(inlineKeyboardMarkup, vampBot)
		if err != nil {
			return fmt.Errorf("sendChPostAsVamp_VideoNote PrepareReplyMarkup err: %v", err)
		}
		newInlineKeyboardMarkupForSupergroup = newInlineKeyboardMarkup
		json_data, err := json.Marshal(newInlineKeyboardMarkup)
		if err != nil {
			srv.l.Error("sendChPostAsVamp_VideoNote Marshal err", zap.Error(err), zap.Any("newInlineKeyboardMarkup", newInlineKeyboardMarkup))
		}
		futureVideoNoteJson["reply_markup"] = string(json_data)
	}

	getFileResp, err := srv.GetFile(m.ChannelPost.VideoNote.FileId)
	if err != nil {
		return fmt.Errorf("sendChPostAsVamp_VideoNote GetFile err: %v", err)
	}
	fileNameDir := strings.Split(getFileResp.Result.File_path, ".")
	fileType := "mp4"
	if len(fileNameDir) > 1 {
		fileType = fileNameDir[1]
	}
	fileNameInServer := fmt.Sprintf("./files/%s.%s", getFileResp.Result.File_unique_id, fileType)
	fileNameInServerAugmented := fmt.Sprintf("./files/%s_augmented.%s", getFileResp.Result.File_unique_id, fileType)
	srv.l.Info(fmt.Sprintf("sendChPostAsVamp_VideoNote: fileNameInServer: %s", fileNameInServer))

	_, err = os.Stat(fileNameInServer)
	if errors.Is(err, os.ErrNotExist) {
		filePath := getFileResp.Result.File_path
		filePath = strings.TrimPrefix(filePath, fmt.Sprintf("/var/lib/telegram-bot-api/%s", srv.Cfg.Token))
		tgFileUrl := fmt.Sprintf("%s/file/bot%s/%s", srv.Cfg.TgLocUrl, srv.Cfg.Token, filePath)

		err = srv.DownloadFile(fileNameInServer, tgFileUrl)
		if err != nil {
			return fmt.Errorf("sendChPostAsVamp_VideoNote DownloadFile err: %v", err)
		}
	}

	if srv.Cfg.IsUniqueVideo == 1 {
		err := UniqueProcessVideoNoteFile(fileNameInServer, fileNameInServerAugmented)
		if err != nil {
			srv.l.Error("sendChPostAsVamp_VideoNote UniqueProcessVideoNoteFile err", zap.Error(err))
		} else {
			fileNameInServer = fileNameInServerAugmented
		}
	}

	futureVideoNoteJson["video_note"] = fmt.Sprintf("@%s", fileNameInServer)
	cf, body, err := files.CreateForm(futureVideoNoteJson)
	if err != nil {
		return fmt.Errorf("sendChPostAsVamp_VideoNote CreateForm err: %v", err)
	}
	cAny2, err := srv.SendVideoNote(body, cf, vampBot.Token)
	if err != nil {
		return fmt.Errorf("sendChPostAsVamp_VideoNote SendVideoNote err: %v", err)
	}
	if cAny2.Result.MessageId != 0 {
		err = srv.db.AddNewPost(vampBot.ChId, cAny2.Result.MessageId, donor_ch_mes_id, cAny2.Result.Caption)
		if err != nil {
			return fmt.Errorf("sendChPostAsVamp_VideoNote AddNewPost err: %v", err)
		}
	}

	getChatResp, err := srv.GetChat(vampBot.ChId, vampBot.Token)
	if err != nil {
		return fmt.Errorf("sendChPostAsVamp_VideoNote GetChat err: %v", err)
	}
	if getChatResp.Result.Type != "supergroup" {
		return nil
	}
	if newInlineKeyboardMarkupForSupergroup.InlineKeyboard == nil {
		return nil
	}
	// for _, inlineKeyboard := range newInlineKeyboardMarkupForSupergroup.InlineKeyboard {
	// 	for _, v := range inlineKeyboard {
	// 		if v.Url == nil && v.Text == "" {
	// 			continue
	// 		}
	// 		err = srv.SendMessageByToken(vampBot.ChId, srv.ChInfoToLinkHTML(*v.Url, v.Text), vampBot.Token)
	// 		if err != nil {
	// 			return fmt.Errorf("sendChPostAsVamp_VideoNote SendMessageByToken for supergroup err: %v", err)
	// 		}
	// 	}
	// }
	return nil
}

func (srv *TgService) sendChPostAsVamp_Video_or_Photo(vampBot entity.Bot, m models.Update, postType string) (error, string) {
	donor_ch_mes_id := m.ChannelPost.MessageId

	var errLink string

	futureVideoJson := map[string]string{
		"chat_id": strconv.Itoa(vampBot.ChId),
	}
	if m.ChannelPost.ReplyToMessage != nil {
		replToDonorChPostId := m.ChannelPost.ReplyToMessage.MessageId
		currPost, err := srv.db.GetPostByDonorIdAndChId(replToDonorChPostId, vampBot.ChId)
		if err != nil {
			return fmt.Errorf("sendChPostAsVamp_Video_or_Photo GetPostByDonorIdAndChId err: %v", err), errLink
		}
		futureVideoJson["reply_to_message_id"] = strconv.Itoa(currPost.PostId)
	}
	if m.ChannelPost.ReplyMarkup != nil {
		var inlineKeyboardMarkup models.InlineKeyboardMarkup
		mycopy.DeepCopy(m.ChannelPost.ReplyMarkup, &inlineKeyboardMarkup)

		newInlineKeyboardMarkup, err := srv.PrepareReplyMarkup(inlineKeyboardMarkup, vampBot)
		if err != nil {
			return fmt.Errorf("sendChPostAsVamp_Video_or_Photo PrepareReplyMarkup err: %v", err), errLink
		}
		json_data, err := json.Marshal(newInlineKeyboardMarkup)
		if err != nil {
			srv.l.Error("sendChPostAsVamp_Video_or_Photo Marshal err", zap.Error(err), zap.Any("newInlineKeyboardMarkup", newInlineKeyboardMarkup))
			return fmt.Errorf("sendChPostAsVamp_Video_or_Photo Marshal err: %v", err), errLink
		}
		futureVideoJson["reply_markup"] = string(json_data)
	}

	var caption string
	if m.ChannelPost.Caption != nil {
		caption = *m.ChannelPost.Caption
		futureVideoJson["caption"] = caption
	}
	
	if len(m.ChannelPost.CaptionEntities) > 0 {
		entities := make([]models.MessageEntity, 0)
		mycopy.DeepCopy(m.ChannelPost.CaptionEntities, &entities)
		sourceCaption := caption

		if srv.Cfg.IsGptText == 1 {
            caption = srv.ReplaceSymbolsOrApenAI(caption)
        }
		newEntities, newCaption, err := srv.PrepareEntities(entities, sourceCaption, caption, vampBot)
		if err != nil {
			return fmt.Errorf("sendChPostAsVamp PrepareEntities err: %v", err), errLink
		}
		if newEntities != nil {
			j, _ := json.Marshal(newEntities)
			futureVideoJson["caption_entities"] = string(j)
		}
		futureVideoJson["caption"] = newCaption
	} else {
		if srv.Cfg.IsGptText == 1 {
            caption = srv.ReplaceSymbolsOrApenAI(caption)
			futureVideoJson["caption"] = caption
        }
	}

	if m.ChannelPost.HasMediaSpoiler {
		futureVideoJson["has_spoiler"] = "true"
	}

	fileId := ""
	if postType == "photo" && len(m.ChannelPost.Photo) > 0 {
		fileId = m.ChannelPost.Photo[len(m.ChannelPost.Photo)-1].FileId

	} else if m.ChannelPost.Video != nil {
		fileId = m.ChannelPost.Video.FileId
		futureVideoJson["width"] = strconv.Itoa(m.ChannelPost.Video.Width)
		futureVideoJson["height"] = strconv.Itoa(m.ChannelPost.Video.Height)

	} else if m.ChannelPost.Animation != nil {
		fileId = m.ChannelPost.Animation.FileId

	} else if m.ChannelPost.Voice != nil {
		fileId = m.ChannelPost.Voice.FileId
	}

	getFileResp, err := srv.GetFile(fileId)
	if err != nil {
		return fmt.Errorf("sendChPostAsVamp_Video_or_Photo GetFile fileId-%s err: %v", fileId, err), errLink
	}
	
	fileNameDir := strings.Split(getFileResp.Result.File_path, ".")
	fileType := "mp4"
	if len(fileNameDir) > 1 {
		fileType = fileNameDir[1]
	}
	fileNameInServer := fmt.Sprintf("./files/%s.%s", getFileResp.Result.File_unique_id, fileType)
	fileNameInServerAugmented := fmt.Sprintf("./files/%s_augmented.%s", getFileResp.Result.File_unique_id, fileType)
	
	srv.l.Info(fmt.Sprintf(
		"sendChPostAsVamp_Video_or_Photo: fileNameInServer: %s, fileNameInServerAugmented: %s",
		fileNameInServer,
		fileNameInServerAugmented,
	))

	_, err = os.Stat(fileNameInServer)
	if errors.Is(err, os.ErrNotExist) {
		filePath := getFileResp.Result.File_path
		filePath = strings.TrimPrefix(filePath, fmt.Sprintf("/var/lib/telegram-bot-api/%s", srv.Cfg.Token))
		tgFileUrl := fmt.Sprintf("%s/file/bot%s/%s", srv.Cfg.TgLocUrl, srv.Cfg.Token, filePath)

		err = srv.DownloadFile(fileNameInServer, tgFileUrl)
		if err != nil {
			return fmt.Errorf("sendChPostAsVamp_Video_or_Photo DownloadFile err: %v", err), errLink
		}
	}

	if postType == "video" && srv.Cfg.IsChangeMediaMetadata == 1 {
		srv.l.Debug("sendChPostAsVamp_Video_or_Photo call RandomizeMP4Metadata")
		err := RandomizeMP4Metadata(fileNameInServer, fileNameInServer)
		if err != nil {
			srv.l.Error("sendChPostAsVamp_Video_or_Photo RandomizeMP4Metadata err", zap.Error(err))
		}
	}

	if postType == "photo" && srv.Cfg.IsUniqueImage == 1 {
		srv.l.Debug("sendChPostAsVamp_Video_or_Photo call UniqueProcessImageFile")
		err := UniqueProcessImageFile(fileNameInServer, fileNameInServerAugmented)
		if err != nil {
			srv.l.Error("sendChPostAsVamp_Video_or_Photo UniqueProcessImageFile err", zap.Error(err))
		} else {
			fileNameInServer = fileNameInServerAugmented
		}
	}

	if postType == "video" && srv.Cfg.IsUniqueVideo == 1 {
		srv.l.Debug("sendChPostAsVamp_Video_or_Photo call UniqueProcessVideoFile")
		err := UniqueProcessVideoFile(fileNameInServer, fileNameInServerAugmented, false)
		if err != nil {
			srv.l.Error("sendChPostAsVamp_Video_or_Photo UniqueProcessVideoFile err", zap.Error(err))
		} else {
			fileNameInServer = fileNameInServerAugmented
		}
	}

	futureVideoJson[postType] = fmt.Sprintf("@%s", fileNameInServer)

	srv.l.Debug("sendChPostAsVamp_Video_or_Photo call fileNameInServerafter all", zap.Any("fileNameInServer", fileNameInServer))

	defer func() {
		if r := recover(); r != nil {
			srv.l.Error(fmt.Sprintf("Panic recovered: %v", r))
			// здесь можно выполнить cleanup или перезапустить сервис
		}
	}()

	formDataContentType, body, err := files.CreateForm(futureVideoJson)
	if err != nil {
		return fmt.Errorf("sendChPostAsVamp_Video_or_Photo CreateForm err: %v", err), errLink
	}

	method := "sendVideo"
	if postType == "photo" {
		method = "sendPhoto"
	} else if postType == "animation" {
		method = "sendAnimation"
	} else if postType == "voice" {
		method = "sendVoice"
	}

	url := fmt.Sprintf(srv.Cfg.TgLocEndp, vampBot.Token, method)
	methodResp, err := srv.MyHttpPost(url, formDataContentType, body)
	if err != nil {
		dbErr := srv.db.AddNewTgError(vampBot.Id, vampBot.Token, vampBot.Username, vampBot.ChId, err.Error())
		if dbErr != nil {
			srv.l.Error("sendChPostAsVamp_Video_or_Photo AddNewTgError dbErr", zap.Error(dbErr))
		}

		reportMess := bytes.Buffer{}
		reportMess.WriteString(fmt.Sprintf("Донор псевдоним: %v\n", srv.Cfg.BotPrefix))
		reportMess.WriteString(fmt.Sprintf("Ошибка при попытке отправки поста с медиа(%v) в канал\n\n", postType))
		reportMess.WriteString(fmt.Sprintf("err: %v\n\n", err.Error()))
		reportMess.WriteString(fmt.Sprintf("bot: %v | %v\n", srv.AddAt(vampBot.Username), vampBot.Token))
		reportMess.WriteString(fmt.Sprintf("ch link: %v\n", vampBot.ChLink))
		gr, _ := srv.db.GetGroupLinkById(vampBot.GroupLinkId)
		reportMess.WriteString(fmt.Sprintf("группа-ссылка: %v - %v\n", vampBot.GroupLinkId, gr.Title))

		sendMessageResp, err2 := srv.SendMessageByTokenV2(srv.Cfg.ChForStatErrors, reportMess.String(), srv.Cfg.BotTokenForStat)
		if err2 != nil {
			srv.l.Warn("sendChPostAsVamp_Video_or_Photo SendMessageByTokenV2 err",
				zap.Error(err2),
				zap.Any("reportMess", reportMess.String()),
				zap.Any("ChForStatErrors", srv.Cfg.ChForStatErrors),
				zap.Any("BotTokenForStat", srv.Cfg.BotTokenForStat),
			)
		}
		if sendMessageResp.Result.MessageId != 0 {
			errLink = fmt.Sprintf("https://t.me/c/%v/%v", srv.Delete100(srv.Cfg.ChForStatErrors), sendMessageResp.Result.MessageId)
		}

		return fmt.Errorf("sendChPostAsVamp_Video_or_Photo Post err: %v", err), errLink
	}
	defer methodResp.Body.Close()

	var sendMediaResp models.SendMediaResp
	if err := json.NewDecoder(methodResp.Body).Decode(&sendMediaResp); err != nil && err != io.EOF {
		return fmt.Errorf("sendChPostAsVamp_Video_or_Photo Decode err: %v", err), errLink
	}
	if sendMediaResp.ErrorCode != 0 {
		dbErr := srv.db.AddNewTgError(vampBot.Id, vampBot.Token, vampBot.Username, vampBot.ChId, sendMediaResp.BotErrResp.Description)
		if dbErr != nil {
			srv.l.Error("sendChPostAsVamp_Video_or_Photo AddNewTgError dbErr", zap.Error(dbErr))
		}

		reportMess := bytes.Buffer{}
		reportMess.WriteString(fmt.Sprintf("Донор псевдоним: %v\n", srv.Cfg.BotPrefix))
		reportMess.WriteString(fmt.Sprintf("Ошибка при отправке поста с медиа(%v) в канал\n\n", postType))
		reportMess.WriteString(fmt.Sprintf("err: %v | %v\n\n", sendMediaResp.BotErrResp.ErrorCode, sendMediaResp.BotErrResp.Description))
		reportMess.WriteString(fmt.Sprintf("bot: %v | %v\n", srv.AddAt(vampBot.Username), vampBot.Token))
		reportMess.WriteString(fmt.Sprintf("ch link: %v\n", vampBot.ChLink))
		gr, _ := srv.db.GetGroupLinkById(vampBot.GroupLinkId)
		reportMess.WriteString(fmt.Sprintf("группа-ссылка: %v - %v\n", vampBot.GroupLinkId, gr.Title))

		sendMessageResp, err := srv.SendMessageByTokenV2(srv.Cfg.ChForStatErrors, reportMess.String(), srv.Cfg.BotTokenForStat)
		if err != nil {
			srv.l.Warn("sendChPostAsVamp_Video_or_Photo SendMessageByTokenV2 err",
				zap.Error(err),
				zap.Any("reportMess", reportMess.String()),
				zap.Any("ChForStatErrors", srv.Cfg.ChForStatErrors),
				zap.Any("BotTokenForStat", srv.Cfg.BotTokenForStat),
			)
		}
		if sendMessageResp.Result.MessageId != 0 {
			errLink = fmt.Sprintf("https://t.me/c/%v/%v", srv.Delete100(srv.Cfg.ChForStatErrors), sendMessageResp.Result.MessageId)
		}
	}

	if sendMediaResp.Result.MessageId != 0 {
		dbErr := srv.db.AddNewPost(vampBot.ChId, sendMediaResp.Result.MessageId, donor_ch_mes_id, sendMediaResp.Result.Caption)
		if dbErr != nil {
			return fmt.Errorf("sendChPostAsVamp_Video_or_Photo AddNewPost dbErr: %v", dbErr), errLink
		}
	} else {
		srv.l.Info(fmt.Sprintf("sendChPostAsVamp_Video_or_Photo: Post resp err: %+v", sendMediaResp.BotErrResp))
		return fmt.Errorf("sendChPostAsVamp_Video_or_Photo: Post resp err: %+v", sendMediaResp.BotErrResp), errLink
	}

	return nil, errLink
}

func (srv *TgService) downloadPostMedia(m models.Update, postType string) (string, error) {
	var fileId string
	if postType == "photo" {
		fileId = m.ChannelPost.Photo[len(m.ChannelPost.Photo)-1].FileId
	} else if m.ChannelPost.Video != nil {
		fileId = m.ChannelPost.Video.FileId
	}
	srv.l.Info(fmt.Sprintf("downloadPostMedia: getting file: %s", fmt.Sprintf(srv.Cfg.TgEndp, srv.Cfg.Token, "getFile?file_id="+fileId)))
	
	GetFileResp, err := srv.GetFile(fileId)
	if err != nil {
		return "", fmt.Errorf("downloadPostMedia GetFile fileId-%s err: %v", fileId, err)
	}
	
	fileNameDir := strings.Split(GetFileResp.Result.File_path, ".")
	fileType := "mp4"
	if len(fileNameDir) > 1 {
		fileType = fileNameDir[1]
	}
	fileNameInServer := fmt.Sprintf("./files/%s.%s", GetFileResp.Result.File_unique_id, fileType)

	filePath := GetFileResp.Result.File_path
	filePath = strings.TrimPrefix(filePath, fmt.Sprintf("/var/lib/telegram-bot-api/%s", srv.Cfg.Token))
	tgFileUrl := fmt.Sprintf("%s/file/bot%s/%s", srv.Cfg.TgLocUrl, srv.Cfg.Token, filePath)

	err = srv.DownloadFile(fileNameInServer, tgFileUrl)
	if err != nil {
		return "", fmt.Errorf("downloadPostMedia DownloadFile err: %v", err)
	}

	srv.l.Info(fmt.Sprintf("downloadPostMedia done, fileNameInServer: %s", fileNameInServer))

	return fileNameInServer, nil
}

func (srv *TgService) downloadPostMediaV2(m models.Update, postType string) (string, string, error) {
	var fileId string
	if postType == "photo" {
		fileId = m.ChannelPost.Photo[len(m.ChannelPost.Photo)-1].FileId
	} else if m.ChannelPost.Video != nil {
		fileId = m.ChannelPost.Video.FileId
	}
	srv.l.Info(fmt.Sprintf("downloadPostMedia: getting file: %s", fmt.Sprintf(srv.Cfg.TgEndp, srv.Cfg.Token, "getFile?file_id="+fileId)))
	
	GetFileResp, err := srv.GetFile(fileId)
	if err != nil {
		return "", "", fmt.Errorf("downloadPostMedia GetFile fileId-%s err: %v", fileId, err)
	}
	
	fileNameDir := strings.Split(GetFileResp.Result.File_path, ".")
	fileType := "mp4"
	if len(fileNameDir) > 1 {
		fileType = fileNameDir[1]
	}
	fileNameInServer := fmt.Sprintf("./files/%s.%s", GetFileResp.Result.File_unique_id, fileType)
	fileNameInServerAugmented := fmt.Sprintf("./files/%s_augmented.%s",  GetFileResp.Result.File_unique_id, fileType)

	filePath := GetFileResp.Result.File_path
	filePath = strings.TrimPrefix(filePath, fmt.Sprintf("/var/lib/telegram-bot-api/%s", srv.Cfg.Token))
	tgFileUrl := fmt.Sprintf("%s/file/bot%s/%s", srv.Cfg.TgLocUrl, srv.Cfg.Token, filePath)

	err = srv.DownloadFile(fileNameInServer, tgFileUrl)
	if err != nil {
		return "" , "", fmt.Errorf("downloadPostMedia DownloadFile err: %v", err)
	}

	srv.l.Info(fmt.Sprintf("downloadPostMedia done, fileNameInServer: %s", fileNameInServer))

	return fileNameInServer, fileNameInServerAugmented, nil
}

func (srv *TgService) sendAndDeleteMedia(vampBot entity.Bot, fileNameInServer string, postType string) (string, int, error) {
	futureJson := map[string]string{
		"chat_id": strconv.Itoa(vampBot.ChId),
	}

	if postType == "video" && srv.Cfg.IsChangeMediaMetadata == 1 {
		err := RandomizeMP4Metadata(fileNameInServer, fileNameInServer)
		if err != nil {
			srv.l.Error("sendAndDeleteMedia RandomizeMP4Metadata err", zap.Error(err))
		}
	}

	futureJson[postType] = fmt.Sprintf("@%s", fileNameInServer)
	
	cf, body, err := files.CreateForm(futureJson)
	if err != nil {
		return "", 0, fmt.Errorf("sendAndDeleteMedia CreateForm err: %v", err)
	}
	method := "sendVideo"
	if postType == "photo" {
		method = "sendPhoto"
	}

	sendMediaResp, err := srv.MyHttpPost(
		fmt.Sprintf(srv.Cfg.TgLocEndp, vampBot.Token, method),
		cf,
		body,
	)
	if err != nil {
		return "", 0, fmt.Errorf("sendAndDeleteMedia Post err: %v", err)
	}
	defer sendMediaResp.Body.Close()
	var cAny2 models.SendMediaResp
	if err := json.NewDecoder(sendMediaResp.Body).Decode(&cAny2); err != nil && err != io.EOF {
		return "", 0, fmt.Errorf("sendAndDeleteMedia Decode err: %v", err)
	}
	if cAny2.ErrorCode != 0 {
		return "", 0, fmt.Errorf("sendAndDeleteMedia method-%s errorResp: %+v", method, cAny2)
	}

	err = srv.DeleteMessage(vampBot.ChId, cAny2.Result.MessageId, vampBot.Token)
	if err != nil {
		srv.l.Error(fmt.Sprintf("sendAndDeleteMedia DeleteMessage err: %v", err))
	}

	var fileId string
	if postType == "photo" {
		if len(cAny2.Result.Photo) > 0 {
			fileId = cAny2.Result.Photo[len(cAny2.Result.Photo)-1].FileId
		}
	} else if postType == "video" {
		if cAny2.Result.Video.FileId != "" {
			fileId = cAny2.Result.Video.FileId
		}
	} else {
		return "", 0, fmt.Errorf("sendAndDeleteMedia: no photo, no video :(")
	}
	return fileId, cAny2.Result.MessageId, nil
}

func (srv *TgService) sendAndDeleteMediaV2(
	vampBot entity.Bot,
	fileNameInServer, fileNameInServerAugmented string,
	postType string,
) (string, int, error) {
	futureJson := map[string]string{
		"chat_id": strconv.Itoa(vampBot.ChId),
	}

	if postType == "video" && srv.Cfg.IsChangeMediaMetadata == 1 {
		err := RandomizeMP4Metadata(fileNameInServer, fileNameInServer)
		if err != nil {
			srv.l.Error("sendAndDeleteMedia RandomizeMP4Metadata err", zap.Error(err))
		}
	}

	if postType == "photo" && srv.Cfg.IsUniqueImage == 1 {
		srv.l.Debug("sendAndDeleteMediaV2 call UniqueProcessImageFile")
		err := UniqueProcessImageFile(fileNameInServer, fileNameInServerAugmented)
		if err != nil {
			srv.l.Error("sendChPostAsVamp_Video_or_Photo UniqueProcessImageFile err", zap.Error(err))
		} else {
			fileNameInServer = fileNameInServerAugmented
		}
	}

	if postType == "video" && srv.Cfg.IsUniqueVideo == 1 {
		srv.l.Debug("sendAndDeleteMediaV2 call UniqueProcessVideoFile")
		err := UniqueProcessVideoFile(fileNameInServer, fileNameInServerAugmented, false)
		if err != nil {
			srv.l.Error("sendChPostAsVamp_Video_or_Photo UniqueProcessVideoFile err", zap.Error(err))
		} else {
			fileNameInServer = fileNameInServerAugmented
		}
	}

	futureJson[postType] = fmt.Sprintf("@%s", fileNameInServer)
	
	cf, body, err := files.CreateForm(futureJson)
	if err != nil {
		return "", 0, fmt.Errorf("sendAndDeleteMedia CreateForm err: %v", err)
	}
	method := "sendVideo"
	if postType == "photo" {
		method = "sendPhoto"
	}

	sendMediaResp, err := srv.MyHttpPost(
		fmt.Sprintf(srv.Cfg.TgLocEndp, vampBot.Token, method),
		cf,
		body,
	)
	if err != nil {
		return "", 0, fmt.Errorf("sendAndDeleteMedia Post err: %v", err)
	}
	defer sendMediaResp.Body.Close()
	var cAny2 models.SendMediaResp
	if err := json.NewDecoder(sendMediaResp.Body).Decode(&cAny2); err != nil && err != io.EOF {
		return "", 0, fmt.Errorf("sendAndDeleteMedia Decode err: %v", err)
	}
	if cAny2.ErrorCode != 0 {
		return "", 0, fmt.Errorf("sendAndDeleteMedia method-%s errorResp: %+v", method, cAny2)
	}

	err = srv.DeleteMessage(vampBot.ChId, cAny2.Result.MessageId, vampBot.Token)
	if err != nil {
		srv.l.Error(fmt.Sprintf("sendAndDeleteMedia DeleteMessage err: %v", err))
	}

	var fileId string
	if postType == "photo" {
		if len(cAny2.Result.Photo) > 0 {
			fileId = cAny2.Result.Photo[len(cAny2.Result.Photo)-1].FileId
		}
	} else if postType == "video" {
		if cAny2.Result.Video.FileId != "" {
			fileId = cAny2.Result.Video.FileId
		}
	} else {
		return "", 0, fmt.Errorf("sendAndDeleteMedia: no photo, no video :(")
	}
	return fileId, cAny2.Result.MessageId, nil
}

func (s *TgService) sendChPostAsVamp_Media_Group(mediaGroupId string) error {
	s.l.Info("sendChPostAsVamp_Media_Group start sending", zap.Any("len s.MediaStore.MediaGroups", len(s.MediaStore.MediaGroups)), zap.Any("s.MediaStore.MediaGroups", s.MediaStore.MediaGroups))
	mediaArr, ok := s.MediaStore.MediaGroups[mediaGroupId]
	if !ok {
		return fmt.Errorf("sendChPostAsVamp_Media_Group: not found in MediaStore")
	}
	s.l.Info("sendChPostAsVamp_Media_Group", zap.Any("len mediaArr", len(mediaArr)), zap.Any("mediaArr", mediaArr))

	allVampBots, err := s.db.GetAllVampBots()
	if err != nil {
		return fmt.Errorf("sendChPostAsVamp_Media_Group GetAllVampBots err: %v", err)
	}
	if len(allVampBots) == 0 {
		return fmt.Errorf("sendChPostAsVamp_Media_Group GetAllVampBots err: len(allVampBots) == 0")
	}

	s.db.EditCfgVal("is-sending-now", "1")
	defer func() {
		s.db.EditCfgVal("is-sending-now", "0")
	}()

	var okSend int
	var notOkSend int
	var IsDisable int
	var ChId0 int
	for _, vampBot := range allVampBots {
		if vampBot.ChId == 0 {
			ChId0++
			continue
		}
		if vampBot.IsDisable == 1 {
			IsDisable++
			continue
		}

		var mediaArrCoppy []Media
		mycopy.DeepCopy(mediaArr, &mediaArrCoppy)

		for i, media := range mediaArrCoppy {
			fileId, messageId, err := s.sendAndDeleteMediaV2(
				vampBot,
				media.File_name_in_server,
				media.File_name_in_server_augmented,
				media.Type_media,
			)
			if err != nil {
				s.l.Error(fmt.Sprintf("sendChPostAsVamp_Media_Group sendAndDeleteMedia ChLink-%s err", vampBot.ChLink), zap.Error(err))
			}
			s.l.Info(fmt.Sprintf("sendAndDeleteMedia ok messageId: %v, fileId: %v", messageId, fileId))
			mediaArrCoppy[i].File_id = fileId
			mediaArrCoppy[i].MessageId = messageId

			if media.Reply_to_donor_message_id != 0 {
				replToDonorChPostId := media.Reply_to_donor_message_id
				currPost, err := s.db.GetPostByDonorIdAndChId(replToDonorChPostId, vampBot.ChId)
				if err != nil {
					s.l.Error(fmt.Sprintf("sendChPostAsVamp_Media_Group: GetPostByDonorIdAndChId err: %v", err))
				}
				mediaArrCoppy[i].Reply_to_message_id = currPost.PostId
			}

			if len(media.Caption_entities) > 0 {
				entities := make([]models.MessageEntity, 0)
				mycopy.DeepCopy(media.Caption_entities, &entities)

				newText := media.Caption
				sourceText := media.Caption

				if s.Cfg.IsGptText == 1 {
					newText = s.ReplaceSymbolsOrApenAI(media.Caption)
				}
				newEntities, newText, err := s.PrepareEntities(entities, sourceText, newText, vampBot)
				if err != nil {
					return fmt.Errorf("sendChPostAsVamp PrepareEntities err: %v", err)
				}
				if newEntities != nil {
					mediaArrCoppy[i].Caption_entities = newEntities
				}
				if media.Caption != "" {
					mediaArrCoppy[i].Caption = newText
				}
			} else {
				if s.Cfg.IsGptText == 1 {
					caption := mediaArrCoppy[i].Caption
					caption = s.ReplaceSymbolsOrApenAI(caption)
					if media.Caption != "" {
						mediaArrCoppy[i].Caption = caption
					}
				}
			}
		}

		arrsik := make([]models.InputMedia, 0)
		sort.Slice(mediaArrCoppy, func(i, j int) (less bool) { //сортировка по MessageId
			return mediaArrCoppy[i].MessageId < mediaArrCoppy[j].MessageId
		})

		s.l.Info("sendChPostAsVamp_Media_Group mediaArrCoppy:", zap.Any("mediaArrCoppy", mediaArrCoppy))

		for _, med := range mediaArrCoppy {
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
		s.l.Info("sendChPostAsVamp_Media_Group arrsik:", zap.Any("arrsik", arrsik))

		mediaJson := map[string]any{
			"chat_id": strconv.Itoa(vampBot.ChId),
			"media":   arrsik,
		}
		if mediaArrCoppy[0].Reply_to_message_id != 0 {
			mediaJson["reply_to_message_id"] = mediaArrCoppy[0].Reply_to_message_id
		}
		media_json, err := json.Marshal(mediaJson)
		if err != nil {
			s.l.Error(fmt.Sprintf("sendChPostAsVamp_Media_Group Marshal err: %v", err))
			continue
		}

		s.l.Info("sendChPostAsVamp_Media_Group: sending media-group", zap.Any("bot ch link", vampBot.ChLink), zap.Any("media_json", mediaJson), zap.Any("bot", vampBot))
		cAny223, err := s.SendMediaGroup(media_json, vampBot.Token)
		if err != nil {
			notOkSend++
			s.l.Error(fmt.Sprintf("sendChPostAsVamp_Media_Group: SendMediaGroup err: %v", err))
		}
		s.l.Info("sendChPostAsVamp_Media_Group SendMediaGroup response", zap.Any("bot ch link", vampBot.ChLink), zap.Any("response", cAny223))
		
		for i, v := range cAny223.Result {
			if i == 0 {
				okSend++
			}
			if v.MessageId == 0 {
				continue
			}
			for _, med := range mediaArrCoppy {
				err = s.db.AddNewPost(vampBot.ChId, v.MessageId, med.Donor_message_id, v.Caption)
				if err != nil {
					s.l.Error(fmt.Sprintf("sendChPostAsVamp_Media_Group AddNewPost err: %v", err))
				}
			}
		}

		time.Sleep(time.Second*2)
	}

	delete(s.MediaStore.MediaGroups, mediaGroupId)
	s.l.Info("sendChPostAsVamp_Media_Group end sending", zap.Any("len s.MediaStore.MediaGroups", len(s.MediaStore.MediaGroups)), zap.Any("s.MediaStore.MediaGroups", s.MediaStore.MediaGroups))

	var reportMess bytes.Buffer
	reportMess.WriteString(fmt.Sprintf("Отчет по медиа-груп:\n"))
	reportMess.WriteString(fmt.Sprintf("Донор псевдоним: %s\n", s.Cfg.BotPrefix))
	reportMess.WriteString(fmt.Sprintf("Всего ботов: %d\n", len(allVampBots)))
	reportMess.WriteString(fmt.Sprintf("Успешно отправлено: %d\n", okSend))
	if notOkSend != 0 {
		reportMess.WriteString(fmt.Sprintf("Неуспешно: %d\n", notOkSend))
	}
	if ChId0 != 0 {
		reportMess.WriteString(fmt.Sprintf("Без подвяз. канала: %d\n", ChId0))
	}
	if IsDisable != 0 {
		reportMess.WriteString(fmt.Sprintf("Отключены от рассылки: %d\n", IsDisable)) 
	}
	donorBot, err := s.db.GetBotInfoByToken(s.Cfg.Token)
	if err != nil {
		s.l.Error(fmt.Errorf("sendChPostAsVamp_Media_Group GetBotInfoByToken err: %v", err).Error())
	}
	s.SendMessage(donorBot.ChId, reportMess.String())

	return nil
}
