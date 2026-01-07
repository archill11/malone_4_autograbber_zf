package tg_service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"myapp/internal/models"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func (srv *TgService) ChInfoToLinkHTML(link, title string) string {
	if strings.HasPrefix(link, "@") {
		link = fmt.Sprintf("https://t.me/%s", link)
	}
	return fmt.Sprintf("<a href=\"%s\">%s</a>", link, title)
}

func (srv *TgService) CreateLinkHTML(title, link string) string {
	if strings.HasPrefix(link, "@") {
		link = fmt.Sprintf("https://t.me/%s", link)
	}
	return fmt.Sprintf("<a href=\"%s\">%s</a>", link, title)
}

func (srv *TgService) DelAt(username string) string {
	usernameRunes := []rune(username)
	if len(usernameRunes) == 0 {
		return ""
	}
	if usernameRunes[0] == '@' {
		usernameRunes = usernameRunes[1:]
	}
	return string(usernameRunes)
}

func (srv *TgService) AddAt(username string) string {
	usernameRunes := []rune(username)
	if len(usernameRunes) == 0 {
		return string('@')
	}
	if usernameRunes[0] != '@' {
		usernameRunes = append([]rune{'@'}, usernameRunes...)
	}
	return string(usernameRunes)
}

func (srv *TgService) GetChidAndPostFromLink(postLink string) (int, int) {
	// "https://t.me/c/2160920084/2"
	var from_chat_id int
	var message_id int
	urlArr := strings.Split(postLink, "/")
	for i, v := range urlArr {
		if len(urlArr) < 4 {
			break
		}
		if v == "t.me" && urlArr[i+1] == "c" {
			chId := urlArr[i+2]
			postId := urlArr[i+3]
			logMes := fmt.Sprintf("ChangeLinkReferredToPost: это ссылка на канал %s и пост %s", chId, postId)
			srv.l.Info(logMes)

			chId = fmt.Sprintf("-100%s", chId)
			from_chat_id, _ = strconv.Atoi(chId)
			message_id, _ = strconv.Atoi(postId)

			return from_chat_id, message_id
		}
	}

	return from_chat_id, message_id
}

func (srv *TgService) RandRange(min, max int) int {
    return rand.Intn(max-min) + min
}

// -1002411417721 -> 2411417721
func (srv *TgService) Delete100(ch_id int) int {
	chidStr := strconv.Itoa(ch_id)
	newChidStr := strings.ReplaceAll(chidStr, "-100", "")
	newChid, _ := strconv.Atoi(newChidStr)
	return newChid
}
func (srv *TgService) Delete100Str(chidStr string) string {
	newChidStr := strings.ReplaceAll(chidStr, "-100", "")
	return newChidStr
}

// 2411417721 -> -1002411417721
func (srv *TgService) Add100(ch_id int) int {
	chidStr := strconv.Itoa(ch_id)
	newChidStr := chidStr
	if !strings.HasPrefix(chidStr, "-100") {
		newChidStr = fmt.Sprintf("-100%v", chidStr)
	}
	newChid, _ := strconv.Atoi(newChidStr)
	return newChid
}
func (srv *TgService) Add100Str(chidStr string) string {
	newChidStr := chidStr
	if !strings.HasPrefix(chidStr, "-100") {
		newChidStr = fmt.Sprintf("-100%v", chidStr)
	}
	return newChidStr
}

func (srv *TgService) CutLongMess(text string, lenth int) string {
	textRune := []rune(text)
    if len(textRune) > lenth+1 { // «Купить пачку сигарет и мочалку…»
        textRune = append(textRune[:lenth], '.', '.', '.',)
    }
	return string(textRune)
}

func (srv *TgService) TnInIntreval(tAfter, tBefore time.Time) bool {
	mskLoc, _ = time.LoadLocation("Europe/Moscow")
	tn := time.Now().In(mskLoc)
	if tn.After(tAfter) && tn.Before(tBefore) {
		return true
	}
	return false
}

// заменяет символы в тексте
func (srv *TgService) ReplaceRundomRuSymbols(mess string) string {
	if srv.Cfg.IsGptTextV2 == 1 {
		return srv.ReplaceRundomRuSymbolsV2(mess)
	}
    if mess == "" {
        return ""
    }
    aRu, aEn := "а", "a"
    oRu, oEn := "о", "o"
    // mRu, mEn := "м", "m"
    pRu, pEn := "р", "p"
    // kRu, kEn := "к", "k"
    cRu, cEn := "с", "c"
    yRu, yEn := "у", "y"
    eRu, eEn := "е", "e"

    ruEnMap := map[string]string{
        aRu: aEn,
        oRu: oEn,
        // mRu: mEn,
        // kRu: kEn,
		pRu: pEn,
        cRu: cEn,
        yRu: yEn,
        eRu: eEn,
    }
    for ru, en := range ruEnMap {
        ruCount := strings.Count(mess, ru)
        mess = strings.Replace(mess, ru, en, randRange(0, ruCount+1))
    }
    return mess
}

// заменяет символы в тексте
func (srv *TgService) ReplaceRundomRuSymbolsV2(mess string) string {
    if mess == "" {
        return ""
    }
    aRu, aEn := "а", "α"
    tRu, tEn := "т", "τ"
	eRu, eEn := "е", "e"
    // eRu, eEn := "е", "℮"
    oRu, oEn := "о", "o"
    kRu, kEn := "к", "ҡ"
    mRu, mEn := "м", "ʍ"
    pRu, pEn := "р", "ρ"
    cRu, cEn := "с", "c"
    // yRu, yEn := "у", "Ꭹ"
	yRu, yEn := "у", "y"
    xRu, xEn := "х", "᙭"

    ruEnMap := map[string]string{
        aRu: aEn,
        tRu: tEn,
        eRu: eEn,
        oRu: oEn,
        kRu: kEn,
        mRu: mEn,
		pRu: pEn,
        cRu: cEn,
        yRu: yEn,
        xRu: xEn,
    }
    for ru, en := range ruEnMap {
        ruCount := strings.Count(mess, ru)
        mess = strings.Replace(mess, ru, en, randRange(0, ruCount+1))
    }
    return mess
}

func randRange(min, max int) int {
	return rand.Intn(max-min) + min
}

func (srv *TgService) CreateShortLink(name, url string) (models.CreateShortLinkResp, error) {
	json_data, err := json.Marshal(map[string]any{
		"name": name,
		"link": url,
	})
	if err != nil {
		return models.CreateShortLinkResp{}, fmt.Errorf("CreateShortLink Marshal err: %v", err)
	}
	resp, err := http.Post(
		srv.Cfg.ShortLinkUrl,
		"application/json",
		bytes.NewBuffer(json_data),
	)
	if err != nil {
		return models.CreateShortLinkResp{}, fmt.Errorf("CreateShortLink Post err: %v", err)
	}
	defer resp.Body.Close()
	var j models.CreateShortLinkResp
	if err := json.NewDecoder(resp.Body).Decode(&j); err != nil {
		var cAny any
		if err := json.NewDecoder(resp.Body).Decode(&cAny); err != nil {
			return models.CreateShortLinkResp{}, fmt.Errorf("CreateShortLink Decode err: %v", err)
		}
		return models.CreateShortLinkResp{}, fmt.Errorf("CreateShortLink any resp: %v", cAny)
	}
	return j, nil
}

func (srv *TgService) CreateShortLinkWithWaiting(name, url string) (models.CreateShortLinkResp, error) {
	newUrlResp, err := srv.CreateShortLink(name, url)
	if err != nil || newUrlResp.Link == "" {
		time.Sleep(time.Second*7)

		newUrlResp, err := srv.CreateShortLink(name, url)
		if err != nil || newUrlResp.Link == "" {
			time.Sleep(time.Second*7)

			newUrlResp, err := srv.CreateShortLink(name, url)
			if err != nil || newUrlResp.Link == "" {
				time.Sleep(time.Second*7)

				newUrlResp, err := srv.CreateShortLink(name, url)
				if err != nil || newUrlResp.Link == "" {
					time.Sleep(time.Second*7)

					newUrlResp, err := srv.CreateShortLink(name, url)
					if err != nil || newUrlResp.Link == "" {
						time.Sleep(time.Second*7)

						newUrlResp, err := srv.CreateShortLink(name, url)
						if err != nil || newUrlResp.Link == "" {
							return newUrlResp, err
						}
					}
				}
			}
		}
	}

	return newUrlResp, nil
}