package config

import (
	"fmt"
	"log"
	"myapp/internal/client/http"
	"myapp/internal/repository/pg"
	"myapp/internal/service/tg_service"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Tg     tg_service.TgConfig
	Server http.SerConfig
	Db     pg.DBConfig
}

func Get() *Config {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env file:", err)
	}
	var c Config

	c.Tg.TgUrl = os.Getenv("TG_URL")
	if c.Tg.TgUrl == "" {
		c.Tg.TgUrl = "https://api.telegram.org"
	}
	c.Tg.TgEndp = fmt.Sprintf("%s/bot%%s/%%s", c.Tg.TgUrl)
	c.Tg.TgLocUrl = os.Getenv("TG_LOCAL_URL")
	if c.Tg.TgLocUrl == "" {
		c.Tg.TgLocUrl = "https://api.telegram.org"
	}
	c.Tg.TgLocEndp = fmt.Sprintf("%s/bot%%s/%%s", c.Tg.TgLocUrl)
	c.Tg.Token = os.Getenv("BOT_TOKEN")
	c.Tg.BotChId, _ = strconv.Atoi(os.Getenv("BOT_CH_ID"))
	c.Tg.BotChLink = os.Getenv("BOT_CH_LINK")

	c.Tg.BotTokenForStat = os.Getenv("BOT_TOKEN_FOR_STAT")
	c.Tg.ChForStat, _ = strconv.Atoi(os.Getenv("CH_FOR_STAT"))
	c.Tg.ChForStatErrors, _ = strconv.Atoi(os.Getenv("CH_FOR_STAT_ERRORS"))

	c.Tg.BotPrefix = os.Getenv("BOT_PREFIX")
	c.Tg.DefaultLichka = os.Getenv("DEFAULT_LICHKA")
	c.Tg.IsPersonalLinks, _ = strconv.Atoi(os.Getenv("IS_PERSONAL_LINKS")) // персональные ссылки для каждого бота
	c.Tg.IsMultiGrabber, _ = strconv.Atoi(os.Getenv("IS_MULTI_GRABBER")) // возможность привязывать одного граббера к разным каналам донорам
	c.Tg.IsGptText, _ = strconv.Atoi(os.Getenv("IS_GPT_TEXT")) // создание уникальных текстов
	c.Tg.IsShortLink, _ = strconv.Atoi(os.Getenv("IS_CHORT_LINK")) // создание уникальных сокращенных ссылок
	c.Tg.ShortLinkUrl  = os.Getenv("CHORT_LINK_URL") // url создание уникальных сокращенных ссылок
	c.Tg.IsChangeMediaMetadata, _ = strconv.Atoi(os.Getenv("IS_CHANGE_MEDIA_METADATA")) // поменять метадату в медиафайлах


	c.Server.Port = os.Getenv("APP_PORT")
	c.Db.User = os.Getenv("PG_USER")
	c.Db.Password = os.Getenv("PG_PASSWORD")
	c.Db.Database = os.Getenv("PG_DATABASE")
	c.Db.Host = os.Getenv("PG_HOST")
	c.Db.Port = os.Getenv("PG_PORT")

	/////////////////////////////////////////////////////////////////
	// c.TG_ENDPOINT = "https://api.telegram.org/bot%s/%s"
	// c.TOKEN       = ""
	// c.PORT        = ""
	// c.PG_USER     = ""
	// c.PG_PASSWORD = ""
	// c.PG_DATABASE = ""
	// c.PG_HOST     = ""

	return &c
}
