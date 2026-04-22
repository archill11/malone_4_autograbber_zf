package tg_service

import (
	"bytes"
	"fmt"
	"myapp/internal/entity"
	"myapp/internal/models"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

func (srv *TgService) SendErrorReportToErrorStatCh(
	vampBot entity.Bot,
	errStr string,
	reportMessage string,
) models.SendMessageResp {
	gr, _ := srv.db.GetGroupLinkById(vampBot.GroupLinkId)

	reportMess := bytes.Buffer{}
	reportMess.WriteString(fmt.Sprintf("Донор псевдоним: %v\n", srv.CreateCodeFmt(srv.Cfg.BotPrefix)))
	reportMess.WriteString(fmt.Sprintf("%v\n\n", reportMessage))
	reportMess.WriteString(fmt.Sprintf("err: %v\n\n", errStr))
	reportMess.WriteString(fmt.Sprintf("bot: %v | %v\n", srv.AddAt(vampBot.Username), vampBot.Token))
	reportMess.WriteString(fmt.Sprintf("ch link: %v\n", vampBot.ChLink))
	reportMess.WriteString(fmt.Sprintf("группа-ссылка: %v - %v\n", vampBot.GroupLinkId, gr.Title))
	
	chId := srv.Cfg.ChForStatErrors
	botToken := srv.Cfg.BotTokenForStat
	
	sendMessageResp, err := srv.SendMessageByTokenV2(chId, reportMess.String(), botToken)
	if err != nil {
		srv.l.Warn("SendErrorToErrorStatCh_v2 SendMessageByTokenV2 err",
			zap.Error(err),
			zap.Any("reportMess", reportMess.String()),
			zap.Any("ChForStatErrors", chId),
			zap.Any("BotTokenForStat", botToken),
		)
	}

	return sendMessageResp
}

func (srv *TgService) SendFinalReportByRefs(
	refkiMap map[int]map[string]int,
	postUUID uuid.UUID,
) {
	if len(refkiMap) == 0 {
		return
	}

	var reportMess2 bytes.Buffer

	if len(refkiMap) > 0 {
		reportMess2.WriteString(fmt.Sprintf("Донор псевдоним: %v\n", srv.CreateCodeFmt(srv.Cfg.BotPrefix)))
		reportMess2.WriteString(fmt.Sprintf("uuid поста в логах: %v\n", srv.CreateCodeFmt(postUUID.String())))
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
}

func (srv *TgService) SendFinalReport(
	channel_id int,
	message_id int,	
	postUUID uuid.UUID,
	allVampBots []entity.Bot,
	okSend int,
	notOkSend int,
	ChId0 int,
	IsDisable int,
	errorLinks []string,
) {
	donorBot, _ := srv.db.GetBotInfoByToken(srv.Cfg.Token)

	var reportMess bytes.Buffer
	reportMess.WriteString(fmt.Sprintf("Отчет по посту:\n"))
	reportMess.WriteString(fmt.Sprintf("Донор псевдоним: %v\n", srv.CreateCodeFmt(srv.Cfg.BotPrefix)))
	reportMess.WriteString(fmt.Sprintf("Бот: %v\n", srv.AddAt(donorBot.Username)))
	reportMess.WriteString(fmt.Sprintf("Пост: %v\n", srv.CreateChPostLink(channel_id, message_id)))
	reportMess.WriteString(fmt.Sprintf("uuid поста в логах: %v\n", srv.CreateCodeFmt(postUUID.String())))
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
	reportMessErrorLinks.WriteString(fmt.Sprintf("Донор псевдоним: %v\n", srv.CreateCodeFmt(srv.Cfg.BotPrefix)))
	reportMessErrorLinks.WriteString(fmt.Sprintf("uuid поста в логах: %v\n", srv.CreateCodeFmt(postUUID.String())))
	reportMessErrorLinks.WriteString(fmt.Sprintf("Список ошибок:\n"))
	if len(errorLinks) > 0 {
		for i, errLink := range errorLinks {
			reportMessErrorLinks.WriteString(fmt.Sprintf("%v) %v\n", i+1, errLink))
	
			if i%20 == 0 && i > 0 {
				sendMessageResp, err := srv.SendMessageByTokenV2(srv.Cfg.ChForStat, reportMessErrorLinks.String(), srv.Cfg.BotTokenForStat)
				if err != nil {
					srv.l.Error(fmt.Sprintf("Donor_addChannelPost SendMessageByToken err: %v", err))
				}
				if sendMessageResp.Result.MessageId != 0 {
					errLinks := srv.CreateChPostLink(srv.Cfg.ChForStat, sendMessageResp.Result.MessageId)
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
}

func (s *TgService) SendReportAboutSendingMediaGroup(
	allVampBotsLen int,
	okSend int,
	notOkSend int,
	ChId0 int,
	IsDisable int,
) {
	reportMess := bytes.Buffer{}
	reportMess.WriteString(fmt.Sprintf("Отчет по медиа-груп:\n"))
	reportMess.WriteString(fmt.Sprintf("Донор псевдоним: %s\n", s.CreateCodeFmt(s.Cfg.BotPrefix)))
	reportMess.WriteString(fmt.Sprintf("Всего ботов: %d\n", allVampBotsLen))
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
}