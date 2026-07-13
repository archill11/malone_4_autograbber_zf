package tg_service

import (
	"fmt"
	"myapp/internal/entity"
	"myapp/internal/models"
	"myapp/pkg/files"
	"myapp/pkg/mycopy"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/go-co-op/gocron"
	"go.uber.org/zap"
)

func (srv *TgService) DeleteOldFiles() {
	cron := gocron.NewScheduler(mskLoc)

	cron.Every(1).Day().At("02:30").Do(func() {
		err := files.RemoveContentsFromDir("files")
		if err != nil {
			srv.l.Error(fmt.Sprintf("DeleteOldFiles .RemoveContentsFromDir('files') err: %v", err))
		}

		srv.l.Info("DeleteOldFiles At(02:30): ok")
	})

	cron.StartAsync()
}

func isValidUTF16EntityBoundary(text string, start, length int) bool {
	if start < 0 || length <= 0 {
		return false
	}

	end := start + length
	if end < start {
		return false
	}

	boundaries := make(map[int]struct{}, len(text)+1)
	totalUnits := 0
	boundaries[0] = struct{}{}

	for _, r := range text {
		if utf16.RuneLen(r) == 2 {
			totalUnits += 2
		} else {
			totalUnits++
		}
		boundaries[totalUnits] = struct{}{}
	}

	if end > totalUnits {
		return false
	}

	_, startOk := boundaries[start]
	_, endOk := boundaries[end]
	return startOk && endOk
}

func (srv *TgService) sanitizeEntitiesForText(entities []models.MessageEntity, text string) []models.MessageEntity {
	if len(entities) == 0 {
		return entities
	}

	validEntities := make([]models.MessageEntity, 0, len(entities))
	for _, entity := range entities {
		if isValidUTF16EntityBoundary(text, entity.Offset, entity.Length) {
			validEntities = append(validEntities, entity)
		}
	}

	if len(validEntities) != len(entities) {
		srv.l.Warn("PrepareEntities: dropped invalid entities after text transform",
			zap.Int("all_entities", len(entities)),
			zap.Int("valid_entities", len(validEntities)),
		)
	}

	return validEntities
}

func utf16UnitsByRuneIndex(text string) []int {
	runes := []rune(text)
	units := make([]int, len(runes)+1)
	for i, r := range runes {
		step := 1
		if utf16.RuneLen(r) == 2 {
			step = 2
		}
		units[i+1] = units[i] + step
	}
	return units
}

func byteOffsetsByRuneIndex(text string) []int {
	runes := []rune(text)
	offsets := make([]int, len(runes)+1)
	bytePos := 0
	for i, r := range runes {
		offsets[i] = bytePos
		bytePos += utf8.RuneLen(r)
	}
	offsets[len(runes)] = len(text)
	return offsets
}

func runeIndexByByteOffset(offsets []int, byteOffset int) (int, bool) {
	i := sort.SearchInts(offsets, byteOffset)
	if i >= len(offsets) || offsets[i] != byteOffset {
		return 0, false
	}
	return i, true
}

func extractEntityTextByUTF16(text string, start, length int) (string, bool) {
	if !isValidUTF16EntityBoundary(text, start, length) {
		return "", false
	}

	units := utf16UnitsByRuneIndex(text)
	startRune := sort.SearchInts(units, start)
	endRune := sort.SearchInts(units, start+length)
	if startRune >= len(units) || endRune >= len(units) || units[startRune] != start || units[endRune] != start+length || endRune <= startRune {
		return "", false
	}

	runes := []rune(text)
	return string(runes[startRune:endRune]), true
}

func normalizeForSearch(v string) string {
	v = strings.ToLower(v)
	v = strings.ReplaceAll(v, "ё", "е")
	v = strings.TrimSpace(v)
	return v
}

func firstNonEmptyLineSpan(text string) (string, int, int, bool) {
	lines := strings.Split(text, "\n")
	bytePos := 0
	for _, line := range lines {
		lineNoCR := strings.TrimSuffix(line, "\r")
		trimmed := strings.TrimSpace(lineNoCR)
		if trimmed != "" {
			startInLine := strings.Index(lineNoCR, trimmed)
			if startInLine < 0 {
				startInLine = 0
			}
			startByte := bytePos + startInLine
			endByte := startByte + len(trimmed)
			return trimmed, startByte, endByte, true
		}
		bytePos += len(line) + 1
	}
	return "", 0, 0, false
}

func remapEntityToNewText(entity models.MessageEntity, oldText, newText string, minRuneStart int) (models.MessageEntity, int, bool) {
	oldPart, ok := extractEntityTextByUTF16(oldText, entity.Offset, entity.Length)
	if !ok {
		return models.MessageEntity{}, 0, false
	}

	newRunes := []rune(newText)
	newRuneToByte := byteOffsetsByRuneIndex(newText)
	newUTF16Units := utf16UnitsByRuneIndex(newText)

	// 1) Exact text search in new text, preserving entity order.
	if minRuneStart < 0 {
		minRuneStart = 0
	}
	if minRuneStart > len(newRunes) {
		minRuneStart = len(newRunes)
	}
	startByte := newRuneToByte[minRuneStart]
	relByte := strings.Index(newText[startByte:], oldPart)
	if relByte >= 0 {
		foundStartByte := startByte + relByte
		foundEndByte := foundStartByte + len(oldPart)
		startRune, okStart := runeIndexByByteOffset(newRuneToByte, foundStartByte)
		endRune, okEnd := runeIndexByByteOffset(newRuneToByte, foundEndByte)
		if okStart && okEnd && endRune > startRune {
			entity.Offset = newUTF16Units[startRune]
			entity.Length = newUTF16Units[endRune] - newUTF16Units[startRune]
			return entity, endRune, true
		}
	}

	// 2) Mention fallback: try to find the same mention token ignoring case.
	if entity.Type == "mention" {
		oldNorm := normalizeForSearch(oldPart)
		if oldNorm != "" {
			searchArea := strings.ToLower(newText[startByte:])
			idx := strings.Index(searchArea, oldNorm)
			if idx >= 0 {
				foundStartByte := startByte + idx
				foundEndByte := foundStartByte + len(oldNorm)
				startRune, okStart := runeIndexByByteOffset(newRuneToByte, foundStartByte)
				endRune, okEnd := runeIndexByByteOffset(newRuneToByte, foundEndByte)
				if okStart && okEnd && endRune > startRune {
					entity.Offset = newUTF16Units[startRune]
					entity.Length = newUTF16Units[endRune] - newUTF16Units[startRune]
					return entity, endRune, true
				}
			}
		}
	}

	// 3) text_link fallback: attach link to the first non-empty line in the new text.
	if entity.Type == "text_link" {
		_, lineStartByte, lineEndByte, ok := firstNonEmptyLineSpan(newText)
		if ok {
			startRune, okStart := runeIndexByByteOffset(newRuneToByte, lineStartByte)
			endRune, okEnd := runeIndexByByteOffset(newRuneToByte, lineEndByte)
			if okStart && okEnd && endRune > startRune {
				entity.Offset = newUTF16Units[startRune]
				entity.Length = newUTF16Units[endRune] - newUTF16Units[startRune]
				return entity, endRune, true
			}
		}
	}

	return models.MessageEntity{}, 0, false
}

func (srv *TgService) rebaseEntitiesToNewText(entities []models.MessageEntity, oldText, newText string) []models.MessageEntity {
	if len(entities) == 0 {
		return entities
	}
	if oldText == newText {
		return entities
	}

	rebased := make([]models.MessageEntity, 0, len(entities))
	nextRuneStart := 0
	for _, entity := range entities {
		newEntity, endRune, ok := remapEntityToNewText(entity, oldText, newText, nextRuneStart)
		if !ok {
			continue
		}
		rebased = append(rebased, newEntity)
		nextRuneStart = endRune
	}

	if len(rebased) != len(entities) {
		srv.l.Warn("PrepareEntities: some entities were not remapped to new text",
			zap.Int("all_entities", len(entities)),
			zap.Int("rebased_entities", len(rebased)),
		)
	}

	return rebased
}

// метод заменяет ссылку на канал и пост такого вида https://t.me/c/1949679854/4333, под конкретного vampBota
func (srv *TgService) ChangeLinkReferredToPost(originalLink string, vampBot entity.Bot) (string, error) {
	urlArr := strings.Split(originalLink, "/")
	for i, v := range urlArr {
		if len(urlArr) < 4 {
			break
		}
		if v == "t.me" && urlArr[i+1] == "c" {
			chId := urlArr[i+2]
			postId := urlArr[i+3]
			logMes := fmt.Sprintf("ChangeLinkReferredToPost: это ссылка на канал %s и пост %s", chId, postId)
			srv.l.Info(logMes)

			refToDonorChPostId, err := strconv.Atoi(postId)
			if err != nil {
				return "", fmt.Errorf("ChangeLinkToPost Atoi err: %v", err)
			}
			currPost, err := srv.db.GetPostByDonorIdAndChId(refToDonorChPostId, vampBot.ChId)
			if err != nil {
				return "", fmt.Errorf("ChangeLinkToPost GetPostByDonorIdAndChId err: %v", err)
			}
			if vampBot.ChId < 0 {
				urlArr[i+2] = strconv.Itoa(-vampBot.ChId)
			} else {
				urlArr[i+2] = strconv.Itoa(vampBot.ChId)
			}
			if urlArr[i+2][0] == '1' && urlArr[i+2][1] == '0' && urlArr[i+2][2] == '0' {
				urlArr[i+2] = urlArr[i+2][3:]
			}
			urlArr[i+3] = strconv.Itoa(currPost.PostId)

			newLink := strings.Join(urlArr, "/")
			return newLink, nil
		}
	}
	// https://t.me/lichka
	if isLink(originalLink, fakeLichkaPrefixes) {
		lichka := vampBot.Lichka
		if lichka != "" {
			newLink := fmt.Sprintf("https://t.me/%s", srv.DelAt(lichka))
			return newLink, nil
		}
	}
	return "", nil
}

func isLink(link string, prefixSlice []string) bool {
	for _, v := range prefixSlice {
		if strings.HasPrefix(link, v) {
			return true
		}
	}
	return false
}

var (
	fakeLichkaPrefixes = []string{
		"https://t.me/lichka",
		"https://lichka",
		"https://fake-lichka",
		"https://t.me/fake-lichka",
		"http://t.me/lichka",
		"t.me/lichka",
	}

	fakeLinkedLichkaPrefixes = []string{
		"https://t.me/fake-linked_lichka",
		"https://t.me/fake-linked-lichka",
		"https://t.me/fake_linked_lichka",
		"http://t.me/fake-linked_lichka",
		"http://t.me/fake-linked-lichka",
		"http://t.me/fake_linked_lichka",
	}
	
	fakeLinkPrefixes = []string{
		"http://fake-link",
		"fake-link",
		"https://fake-link",
	}

	cutLinkPrefixes = []string{
		"http://cut-link",
		"cut-link",
		"https://cut-link",
	}
)

// метод заменяет fake-link на нужную группу-ссылку vampBota
// и вырезает все ссылки и Entities если группа-ссылка - cut-link
func (srv *TgService) PrepareEntities(
	entities []models.MessageEntity,
	sourceText, messText string,
	vampBot entity.Bot,
) ([]models.MessageEntity, string, error) {
	srv.l.Info("PrepareEntities", zap.Any("vampBot", vampBot))

	cutEntities := false

	entities = srv.rebaseEntitiesToNewText(entities, sourceText, messText)

	for i, v := range entities {
		// если fake-link
		if isLink(v.Url, fakeLichkaPrefixes) {
			lichka := srv.DelAt(vampBot.Lichka)
			urlLichka := fmt.Sprintf("https://t.me/%v", lichka)

			if srv.Cfg.IsShortLink == 1 && srv.Cfg.IsShortLinkToClick == 0 {
				newUrlResp, err := srv.CreateShortLinkWithWaiting(urlLichka)
				srv.l.Debug(
					"МЕТОД PrepareEntities go CreateShortLinkWithWaiting",
					zap.Any("urlLichka", urlLichka),
					zap.Any("newUrlResp", newUrlResp),
					zap.Any("vampBot", vampBot),
					zap.Any("entities[i]", entities[i]),
					zap.Any("entities", entities),
				)
				if err != nil || newUrlResp.Link == "" {
					err := fmt.Errorf("PrepareEntities CreateShortLinkWithWaiting err: %v, newUrlResp: %+v, url: %v", err, newUrlResp, urlLichka)
					srv.l.Error(err.Error())

					reportMessage := "Ошибка при создании короткой уникальной ссылки"
					srv.SendErrorReportToErrorStatCh(vampBot, err.Error(), reportMessage)
				}
				if newUrlResp.Link != "" {
					urlLichka = newUrlResp.Link
				}
				srv.l.Debug(
					"МЕТОД PrepareEntities go CreateShortLinkWithWaiting",
					zap.Any("urlLichka", urlLichka),
					zap.Any("newUrlResp", newUrlResp),
					zap.Any("entities[i]", entities[i]),
					zap.Any("entities", entities),
				)
			} else if srv.Cfg.IsShortLinkToClick == 1 {
				botInfo, _ := srv.db.GetBotInfoById(vampBot.Id)
				if botInfo.ToClickShortLinkToLichka != "" {
					urlLichka = botInfo.ToClickShortLink
				} else {
					newUrlResp, _ := srv.CreateShortLinkWithWaiting(urlLichka)
					if newUrlResp.Link != "" {
						urlLichka = newUrlResp.Link
						srv.db.EditBotToClickShortLinkToLichka(vampBot.Id, urlLichka)
					}
				}
			}

			entities[i].Url = urlLichka
			continue
		}

		if isLink(v.Url, fakeLinkedLichkaPrefixes) {
			if vampBot.LinkedLichka != "" {
				entities[i].Url = vampBot.LinkedLichka
			}
			continue
		}

		if isLink(v.Url, fakeLinkPrefixes) {
			groupLink, err := srv.db.GetGroupLinkById(vampBot.GroupLinkId)
			if err != nil {
				return nil, messText, err
			}
			srv.l.Debug("PrepareEntities:", zap.Any("vampBot", vampBot), zap.Any("groupLink", groupLink))
			if groupLink.Link == "" {
				continue
			}
			// если cut-link
			if isLink(groupLink.Link, cutLinkPrefixes) {
				messText = strings.Replace(messText, "Переходим по ссылке - ССЫЛКА", "", -1)
				messText = strings.Replace(messText, "👉 РЕГИСТРАЦИЯ ТУТ 👈", "", -1)
				messText = strings.Replace(messText, "🔖 Написать мне 🔖", "", -1)
				cutEntities = true
				break
			}
			refLink := groupLink.Link

			if srv.Cfg.IsShortLink == 1 && srv.Cfg.IsShortLinkToClick == 0 {
				newUrlResp, err := srv.CreateShortLinkWithWaiting(refLink)
				if err != nil || newUrlResp.Link == "" {
					err := fmt.Errorf("PrepareEntities CreateShortLinkWithWaiting err: %v, newUrlResp: %+v, url: %v", err, newUrlResp, refLink)
					srv.l.Error(err.Error())

					reportMessage := "Ошибка при создании короткой уникальной ссылки"
					srv.SendErrorReportToErrorStatCh(vampBot, err.Error(), reportMessage)
				}
				if newUrlResp.Link != "" {
					refLink = newUrlResp.Link
				}

			} else if srv.Cfg.IsShortLinkToClick == 1 {
				botInfo, _ := srv.db.GetBotInfoById(vampBot.Id)
				if botInfo.ToClickShortLink != "" {
					refLink = botInfo.ToClickShortLink
				} else {
					newUrlResp, err := srv.CreateShortLinkWithWaiting(refLink)
					if err != nil {
						srv.l.Error("CreateShortLinkWithWaiting 22_33 err",
							zap.Any("newUrlResp", newUrlResp),
							zap.Any("err", err),
						)
					}
					if newUrlResp.Link != "" {
						refLink = newUrlResp.Link
						srv.db.EditBotToClickShortLink(vampBot.Id, refLink)
					}
				}
			}

			if srv.Cfg.IsReplaceShortLinkDomen == 1 {
				if vampBot.ShortDomenToReplace != "" {
					parsedURL, err := url.Parse(refLink)
					if err == nil {
						host := parsedURL.Hostname()
						refLink = strings.ReplaceAll(refLink, host, vampBot.ShortDomenToReplace)
					}
				}
			}

			if srv.Cfg.IsPersonalLinks == 1 {
				if vampBot.PersonalLink != "" {
					refLink = vampBot.PersonalLink
				}
			}

			entities[i].Url = refLink
			continue
		}
		// если Tg ссылка
		newUrl, err := srv.ChangeLinkReferredToPost(v.Url, vampBot)
		if err != nil {
			return nil, messText, fmt.Errorf("PrepareEntities ChangeLinkReferredToPost err: %v", err)
		}
		if newUrl != "" {
			entities[i].Url = newUrl
		}
		srv.l.Debug("PrepareEntities", zap.Any("newUrl", newUrl), zap.Any("vampBot", vampBot))
	}

	lichka := srv.AddAt(vampBot.Lichka)
	srv.l.Debug(
		"PrepareEntities Replace 1 @lichka",
		zap.Any("lichka", lichka),
		zap.Any("old messText", messText),
		zap.Any("vampBot", vampBot),
	)
	if srv.DelAt(vampBot.Lichka) != "" {
		messText = strings.Replace(messText, "@lichka", lichka, -1)
	}
	if srv.DelAt(vampBot.Lichka) == "" && vampBot.LinkedLichka != "" {
		txtMeText := "Написать мне"
		messText = strings.Replace(messText, "@lichka", txtMeText, -1)

		// txtMeIdx := strings.Index([]rune(messText), []rune(txtMeText))
		// txtMeIdx := strings.Index(string([]rune(messText)), string([]rune(txtMeText)))

		var txtMeIdx int
		for i := 0; i < len([]rune(messText)); i++ {
			if string([]rune(messText)[i:i+len([]rune(txtMeText))]) == txtMeText {
				txtMeIdx = i
				break
			}
		}
		txtMeOffset := len([]rune(messText)) - txtMeIdx
		txtMeLength := len([]rune(txtMeText))

		entities = append(entities, models.MessageEntity{
			Type: "text_link",
			Url:  vampBot.LinkedLichka,
			Offset: txtMeOffset-1,
			Length: txtMeLength,
		})

		srv.l.Debug(
			"PrepareEntities Replace @lichka to LinkedLichka",
			zap.Any("txtMeText", txtMeText),
			zap.Any("txtMeIdx", txtMeIdx),
			zap.Any("len(messText)", len([]rune(messText))),
			zap.Any("txtMeOffset", txtMeOffset),
			zap.Any("txtMeLength", txtMeLength),
			zap.Any("messText", messText),
			zap.Any("entities", entities),
		)
	}
	srv.l.Debug(
		"PrepareEntities Replace 2 @lichka",
		zap.Any("lichka", lichka),
		zap.Any("new messText", messText),
		zap.Any("vampBot", vampBot),
		zap.Any("entities", entities),
	)
	if !cutEntities {
		return entities, messText, nil
	}
	return nil, messText, nil
}

func (srv *TgService) PrepareReplyMarkup(entities models.InlineKeyboardMarkup, vampBot entity.Bot) (models.InlineKeyboardMarkup, error) {
	for i, v := range entities.InlineKeyboard {
		for ii, vv := range v {
			if vv.Url == nil {
				continue
			}
			// если fake-link
			if isLink(*vv.Url, fakeLinkPrefixes) {
				groupLink, err := srv.db.GetGroupLinkById(vampBot.GroupLinkId)
				if err != nil {
					return models.InlineKeyboardMarkup{}, err
				}
				srv.l.Info("PrepareEntities:", zap.Any("vampBot", vampBot), zap.Any("groupLink", groupLink))
				if groupLink.Link == "" {
					continue
				}
				entities.InlineKeyboard[i][ii].Url = &groupLink.Link
				continue
			}
			// если Tg ссылка
			newUrl, err := srv.ChangeLinkReferredToPost(*vv.Url, vampBot)
			if err != nil {
				return models.InlineKeyboardMarkup{}, fmt.Errorf("PrepareReplyMarkup ChangeLinkReferredToPost err: %v", err)
			}
			if newUrl != "" {
				entities.InlineKeyboard[i][ii].Url = &newUrl
			}
		}
	}
	return entities, nil
}

func (srv *TgService) GetPostAndChFromLink(link string) (string, string, error) {
	urlArr := strings.Split(link, "/")
	if len(urlArr) != 6 {
		return "", "", fmt.Errorf("GetPostAndChFromLink err: не правилная ссылка %s", link)
	}
	for i, v := range urlArr {
		if v == "t.me" && urlArr[i+1] == "c" {
			chId := urlArr[i+2]
			postId := urlArr[i+3]
			logMes := fmt.Sprintf("GetPostAndChFromLink: это ссылка на канал %s и пост %s", chId, postId)
			srv.l.Info(logMes)
			return chId, postId, nil
		}
	}
	return "", "", nil
}

func (srv *TgService) GetAugmentedVampBots(allVampBots []entity.Bot) ([]entity.Bot) {
	augmentedAllVampBots := make([]entity.Bot, 0)
	for _, vampBot := range allVampBots {
		augmentedAllVampBots = append(augmentedAllVampBots, vampBot)

		// var additionalChs []entity.AdditionalCh
		// err = json.Unmarshal(vampBot.AdditionalChs, &additionalChs)
		// if err != nil {
		// 	return fmt.Errorf("Donor_addChannelPost json.Unmarshal err: %v", err)
		// }
		for _, additionalCh := range vampBot.AdditionalChs {
			if additionalCh.ChId == 0 {
				continue
			}
			var botWithOtherCh entity.Bot
			mycopy.DeepCopy(vampBot, &botWithOtherCh)
			botWithOtherCh.ChId = additionalCh.ChId
			botWithOtherCh.ChLink = additionalCh.ChLink
			augmentedAllVampBots = append(augmentedAllVampBots, botWithOtherCh)
		}
	}

	return augmentedAllVampBots
}