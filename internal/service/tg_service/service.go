package tg_service

import (
	"myapp/internal/models"
	"myapp/internal/repository/pg"
	"time"

	"go.uber.org/zap"
)

var (
	mskLoc, _ = time.LoadLocation("Europe/Moscow")
)

type (
	UpdateConfig struct {
		Offset  int
		Timeout int
		Buffer  int
	}

	TgConfig struct {
		TgUrl           string
		TgEndp          string
		TgLocEndp       string
		TgLocUrl        string
		Token           string
		BotChId         int
		BotChLink       string
		BotPrefix       string
		DefaultLichka   string
		IsPersonalLinks int
		IsLinkedLichka  int
		ChForStat       int
		ChForStatErrors int
		BotTokenForStat string
		IsMultiGrabber  int
		IsReplaceShortLinkDomen int
		IsUseProxy  int
		ProxyStr  string
		IsGptText  int
		IsGptTextV2  int
		IsGptTextOpenAI  int
		OpenAiAPIToken  string
		IsShortLink  int
		// ShortLinkUrl  string
		IsShortLinkToClick  int
		ToClickToken  string
		IsChangeMediaMetadata int
		IsUniqueVideo int
		IsUniqueImage int
	}

	TgService struct {
		Cfg        TgConfig
		db         *pg.Database
		db2        *pg.Database
		l          *zap.Logger
		MediaCh    chan Media
		MediaStore MediaStore
	}
)

type (
	MediaStore struct {
		MediaGroups map[string][]Media
	}

	Media struct {
		Media_group_id            string
		Type_media                string
		File_name_in_server       string
		File_name_in_server_augmented string
		Donor_message_id          int
		Reply_to_donor_message_id int // реплай на сообщение в канале доноре
		Caption                   string
		Caption_entities          []models.MessageEntity
		File_id                   string
		MessageId                 int // для сортировки что бы был правильный порядок медиа
		Reply_to_message_id       int // реплай на сообщение в канале вампире
	}
)

func New(
	l *zap.Logger,
	conf TgConfig,
	db *pg.Database,
	db2 *pg.Database,
) (*TgService, error) {
	s := &TgService{
		Cfg:     conf,
		db:      db,
		db2:     db2,
		l:       l,
		MediaCh: make(chan Media, 10),
		MediaStore: MediaStore{
			MediaGroups: make(map[string][]Media),
		},
	}
	// перелив групп-ссылок из db2 в db
	// s.PerelivVampBots()

	// добавить граббера в базу при первом создании сервиса
	go s.InsertGrabberBot()
	// удаление ненужных файлов
	go s.DeleteOldFiles()
	// удаление потеряных ботов
	// go s.DeleteLostBots()
	// уведомление о метке на канале
	// go s.AlertScamBots()
	// получение tg Donor updates
	go s.GetTgBotUpdates()
	// когда MediaGroup
	go s.AcceptChPostByAdmin()


	return s, nil
}



