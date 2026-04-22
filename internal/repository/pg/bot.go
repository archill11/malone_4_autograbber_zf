package pg

import (
	"encoding/json"
	"fmt"
	"myapp/internal/entity"
)

func (s *Database) AddNewBot(
	id int,
	username, firstname, token string,
	isDonor int,
) error {
	q := `
		INSERT INTO bots (
			id,
			username,
			first_name,
			token,
			is_donor
		)
		VALUES ($1, $2, $3, $4, $5) 
		ON CONFLICT DO NOTHING`
	_, err := s.Exec(q, id, username, firstname, token, isDonor)
	if err != nil {
		return fmt.Errorf("AddNewBot err: %v", err)
	}
	return nil
}

func (s *Database) DeleteBot(id int) error {
	q := `DELETE FROM bots WHERE id = $1`
	_, err := s.Exec(q, id)
	if err != nil {
		return fmt.Errorf("DeleteBot err: %w", err)
	}
	return nil
}

func (s *Database) GetBotByChannelId(channelId int) (entity.Bot, error) {
	q := `
		SELECT coalesce((
			SELECT to_json(c)
			FROM bots as c
			WHERE ch_id = $1 
		), '{}'::json)
	`
	var u entity.Bot
	var data []byte
	err := s.QueryRow(q, channelId).Scan(&data)
	if err != nil {
		return u, fmt.Errorf("GetBotByChannelId Scan err: %v", err)
	}
	if err := json.Unmarshal(data, &u); err != nil {
		return u, fmt.Errorf("GetBotByChannelId Unmarshal err: %v", err)
	}
	return u, nil
}

func (s *Database) GetBotByChannelLink(channelLink string) (entity.Bot, error) {
	q := `
		SELECT coalesce((
			SELECT to_json(c)
			FROM bots as c
			WHERE ch_link = $1 
		), '{}'::json)
	`
	var u entity.Bot
	var data []byte
	err := s.QueryRow(q, channelLink).Scan(&data)
	if err != nil {
		return u, fmt.Errorf("GetBotByChannelLink Scan err: %v", err)
	}
	if err := json.Unmarshal(data, &u); err != nil {
		return u, fmt.Errorf("GetBotByChannelLink Unmarshal err: %v", err)
	}
	return u, nil
}

func (s *Database) GetBotsByGrouLinkId(groupLinkId int) ([]entity.Bot, error) {
	q := `
		SELECT coalesce((
			SELECT json_agg(c)
			FROM bots as c
			WHERE group_link_id = $1 
		), '[]'::json)
	`
	u := make([]entity.Bot, 0)
	var data []byte
	err := s.QueryRow(q, groupLinkId).Scan(&data)
	if err != nil {
		return u, fmt.Errorf("GetBotsByGrouLinkId Scan err: %v", err)
	}
	if err := json.Unmarshal(data, &u); err != nil {
		return u, fmt.Errorf("GetBotsByGrouLinkId Unmarshal err: %v", err)
	}
	return u, nil
}

func (s *Database) GetAllBots() ([]entity.Bot, error) {
	q := `
		SELECT coalesce((
			SELECT json_agg(c)
			FROM bots as c
			ORDER BY min(c.created_at)
		), '[]'::json)
	`
	u := make([]entity.Bot, 0)
	var data []byte
	err := s.QueryRow(q).Scan(&data)
	if err != nil {
		return u, fmt.Errorf("GetAllBots Scan err: %v", err)
	}
	if err := json.Unmarshal(data, &u); err != nil {
		return u, fmt.Errorf("GetAllBots Unmarshal err: %v", err)
	}
	u2 := make([]entity.Bot, 0)
	for i := len(u)-1; i >= 0; i-- {
		u2 = append(u2, u[i])
	}
	return u2, nil
}

func (s *Database) GetAllVampBots() ([]entity.Bot, error) {
	q := `
		SELECT coalesce((
			SELECT json_agg(c)
			FROM bots as c
			WHERE is_donor = 0 
		), '[]'::json)
	`
	u := make([]entity.Bot, 0)
	var data []byte
	err := s.QueryRow(q).Scan(&data)
	if err != nil {
		return u, fmt.Errorf("GetAllVampBots Scan err: %v", err)
	}
	if err := json.Unmarshal(data, &u); err != nil {
		return u, fmt.Errorf("GetAllVampBots Unmarshal err: %v", err)
	}
	return u, nil
}

func (s *Database) GetAllNoChannelBots() ([]entity.Bot, error) {
	q := `
		SELECT coalesce((
			SELECT json_agg(c)
			FROM bots as c
			WHERE ch_id = 0 
		), '[]'::json)
	`
	u := make([]entity.Bot, 0)
	var data []byte
	err := s.QueryRow(q).Scan(&data)
	if err != nil {
		return u, fmt.Errorf("GetAllNoChannelBots Scan err: %v", err)
	}
	if err := json.Unmarshal(data, &u); err != nil {
		return u, fmt.Errorf("GetAllNoChannelBots Unmarshal err: %v", err)
	}
	return u, nil
}

func (s *Database) GetBotInfoById(botId int) (entity.Bot, error) {
	q := `
		SELECT coalesce((
			SELECT to_json(c)
			FROM bots as c
			WHERE id = $1 
		), '{}'::json)
	`
	var u entity.Bot
	var data []byte
	err := s.QueryRow(q, botId).Scan(&data)
	if err != nil {
		return u, fmt.Errorf("GetBotInfoById Scan err: %v", err)
	}
	if err := json.Unmarshal(data, &u); err != nil {
		return u, fmt.Errorf("GetBotInfoById Unmarshal err: %v", err)
	}
	return u, nil
}

func (s *Database) GetBotInfoByToken(token string) (entity.Bot, error) {
	q := `
		SELECT coalesce((
			SELECT to_json(c)
			FROM bots as c
			WHERE token = $1 
		), '{}'::json)
	`
	var u entity.Bot
	var data []byte
	err := s.QueryRow(q, token).Scan(&data)
	if err != nil {
		return u, fmt.Errorf("GetBotInfoByToken Scan err: %v", err)
	}
	if err := json.Unmarshal(data, &u); err != nil {
		return u, fmt.Errorf("GetBotInfoByToken Unmarshal err: %v", err)
	}
	return u, nil
}

func (s *Database) EditBotField(
	botId int,
	field string,
	content any,
) error {
	q := fmt.Sprintf(`
		UPDATE bots SET
			%s = $1
		WHERE id = $2
	`, field)
	_, err := s.Exec(q, content, botId)
	if err != nil {
		return fmt.Errorf("EditBotField: botId: %v field: %v content: %v err: %v", botId, field, content, err)
	}
	return nil
}

func (s *Database) EditBotGroupLinkIdToNull(groupLinkId int) error {
	q := `
		UPDATE bots SET 
			group_link_id = 0 
		WHERE group_link_id = $1
	`
	_, err := s.Exec(q, groupLinkId)
	if err != nil {
		return fmt.Errorf("EditBotGroupLinkIdToNull: err: %v", err)
	}
	return nil
}

func (s *Database) EditBotGroupLinkId(groupLinkId, botId int) error {
	q := `
		UPDATE bots SET
			group_link_id = $1
		WHERE id = $2
	`
	_, err := s.Exec(q, groupLinkId, botId)
	if err != nil {
		return fmt.Errorf("EditBotGroupLinkId: err: %v", err)

	}
	return nil
}

func (s *Database) EditBotPersonalLink(personal_link string, botId int) error {
	q := `
		UPDATE bots SET
			personal_link = $1
		WHERE id = $2
	`
	_, err := s.Exec(q, personal_link, botId)
	if err != nil {
		return fmt.Errorf("EditBotPersonalLink: err: %v", err)

	}
	return nil
}

func (s *Database) EditBotLichka(botId int, lichka string) error {
	q := `
		UPDATE bots SET
			lichka = $1
		WHERE id = $2
	`
	_, err := s.Exec(q, lichka, botId)
	if err != nil {
		return fmt.Errorf("EditBotLichka: err: %v", err)

	}
	return nil
}

func (s *Database) SetBotLichkaAllEmpty(lichka string) error {
	q := `
		UPDATE bots SET
			lichka = $1
		WHERE lichka = ''
	`
	_, err := s.Exec(q, lichka)
	if err != nil {
		return fmt.Errorf("SetBotLichkaAllEmpty: err: %v", err)

	}
	return nil
}

func (s *Database) EditBotUserCreator(botId, user_creator int) error {
	q := `
		UPDATE bots SET
			user_creator = $1
		WHERE id = $2
	`
	_, err := s.Exec(q, user_creator, botId)
	if err != nil {
		return fmt.Errorf("EditBotUserCreator: err: %v", err)

	}
	return nil
}

func (s *Database) EditBotChIsSkam(botId, chIsSkam int) error {
	q := `
		UPDATE bots SET
			ch_is_skam = $1
		WHERE id = $2
	`
	_, err := s.Exec(q, chIsSkam, botId)
	if err != nil {
		return fmt.Errorf("EditBotChIsSkam: err: %v", err)

	}
	return nil
}

func (s *Database) EditBotDonorChId(botId, donor_ch_id int) error {
	q := `
		UPDATE bots SET
			donor_ch_id = $1
		WHERE id = $2
	`
	_, err := s.Exec(q, donor_ch_id, botId)
	if err != nil {
		return fmt.Errorf("EditBotDonorChId: err: %v", err)

	}
	return nil
}

func (s *Database) EditBotIsErrInStat(botId, is_err_in_stat int) error {
	q := `
		UPDATE bots SET
			is_err_in_stat = $1
		WHERE id = $2
	`
	_, err := s.Exec(q, is_err_in_stat, botId)
	if err != nil {
		return fmt.Errorf("EditBotIsErrInStat: err: %v", err)
	}
	return nil
}

func (s *Database) EditBotToClickShortLink(
	botId int,
	to_click_short_link string,
) error {
	q := `
		UPDATE bots SET
			to_click_short_link = $1
		WHERE id = $2
	`
	_, err := s.Exec(q, to_click_short_link, botId)
	if err != nil {
		return fmt.Errorf("EditBotToClickShortLink: err: %v", err)

	}
	return nil
}

func (s *Database) EditBotToClickShortLinkToLichka(
	botId int,
	to_click_short_link_to_lichka string,
) error {
	q := `
		UPDATE bots SET
			to_click_short_link_to_lichka = $1
		WHERE id = $2
	`
	_, err := s.Exec(q, to_click_short_link_to_lichka, botId)
	if err != nil {
		return fmt.Errorf("EditBotToClickShortLink: err: %v", err)

	}
	return nil
}

func (s *Database) EditBotShortDomenToReplace(
	botId int,
	short_domen_to_replace string,
) error {
	q := `
		UPDATE bots SET
			short_domen_to_replace = $1
		WHERE id = $2
	`
	_, err := s.Exec(q, short_domen_to_replace, botId)
	if err != nil {
		return fmt.Errorf("EditBotShortDomenToReplace: err: %v", err)

	}
	return nil
}