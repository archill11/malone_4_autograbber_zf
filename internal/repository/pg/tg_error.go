package pg

import (
	"fmt"
)

func (s *Database) AddNewTgError(
	bot_id int,
	bot_token, bot_username string,
	bot_ch_id int,
	err_description string,
) error {
	q := `
		INSERT INTO tg_errors (
			bot_id,
			bot_token,
			bot_username,
			bot_ch_id,
			err_description
		)
		VALUES
			($1, $2, $3, $4, $5)
		ON CONFLICT (bot_id, err_description)
		DO UPDATE SET err_count = tg_errors.err_count + 1
	`
	_, err := s.Exec(q,
		bot_id,
		bot_token,
		bot_username,
		bot_ch_id,
		err_description,
	)
	if err != nil {
		return fmt.Errorf("db: AddNewTgError: %w", err)
	}

	return nil
}

