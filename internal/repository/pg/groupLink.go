package pg

import (
	"encoding/json"
	"fmt"
	"myapp/internal/entity"
)

func (s *Database) AddNewGroupLink(title, link string) error {
	q := `INSERT INTO group_link (title, link) 
			VALUES ($1, $2) 
		ON CONFLICT DO NOTHING`
	_, err := s.Exec(q, title, link)
	if err != nil {
		return fmt.Errorf("AddNewGroupLink err: %v", err)
	}
	return nil
}

func (s *Database) AddNewGroupLinkV2(title, link string, user_creator int) error {
	q := `INSERT INTO group_link (title, link, user_creator)
			VALUES ($1, $2, $3)
		ON CONFLICT DO NOTHING`
	_, err := s.Exec(q, title, link, user_creator)
	if err != nil {
		return fmt.Errorf("AddNewGroupLinkV2 err: %v", err)
	}
	return nil
}

func (s *Database) GetGroupLinkByAllFields(title, link string, user_creator int) (entity.GroupLink, error) {
	q := `
		SELECT coalesce((
			SELECT to_json(c)
	  		FROM group_link as c
	  		WHERE title = $1 AND link = $2 AND user_creator = $3
		), '{}'::json)
	`
	var u entity.GroupLink
	var data []byte
	err := s.QueryRow(q, title, link, user_creator).Scan(&data)
	if err != nil {
		return u, fmt.Errorf("GetGroupLinkByAllFields Scan err: %v", err)
	}
	if err := json.Unmarshal(data, &u); err != nil {
		return u, fmt.Errorf("GetGroupLinkByAllFields Unmarshal err: %v", err)
	}
	return u, nil
}

func (s *Database) GetGroupLinkByLink(link string) (entity.GroupLink, error) {
	q := `
		SELECT coalesce((
			SELECT to_json(c)
	  		FROM group_link as c
	  		WHERE link = $1
		), '{}'::json)
	`
	var u entity.GroupLink
	var data []byte
	err := s.QueryRow(q, link).Scan(&data)
	if err != nil {
		return u, fmt.Errorf("GetGroupLinkByLink Scan err: %v", err)
	}
	if err := json.Unmarshal(data, &u); err != nil {
		return u, fmt.Errorf("GetGroupLinkByLink Unmarshal err: %v", err)
	}
	return u, nil
}

func (s *Database) GetGroupLinkById(id int) (entity.GroupLink, error) {
	q := `
		SELECT coalesce((
			SELECT to_json(c)
	  		FROM group_link as c
	  		WHERE id = $1
		), '{}'::json)
	`
	var u entity.GroupLink
	var data []byte
	err := s.QueryRow(q, id).Scan(&data)
	if err != nil {
		return u, fmt.Errorf("GetGroupLinkById Scan err: %v", err)
	}
	if err := json.Unmarshal(data, &u); err != nil {
		return u, fmt.Errorf("GetGroupLinkById Unmarshal err: %v", err)
	}
	return u, nil
}

func (s *Database) GetAllGroupLinks() ([]entity.GroupLink, error) {
	q := `
		SELECT coalesce((
			SELECT json_agg(c)
	  		FROM group_link as c
		), '[]'::json)
	`
	u := make([]entity.GroupLink, 0)
	var data []byte
	err := s.QueryRow(q).Scan(&data)
	if err != nil {
		return u, fmt.Errorf("GetAllGroupLinks Scan err: %v", err)
	}
	if err := json.Unmarshal(data, &u); err != nil {
		return u, fmt.Errorf("GetAllGroupLinks Unmarshal err: %v", err)
	}
	return u, nil
}

func (s *Database) DeleteGroupLink(id int) error {
	q := `DELETE FROM group_link WHERE id = $1`
	_, err := s.Exec(q, id)
	if err != nil {
		return fmt.Errorf("DeleteGroupLink err: %v", err)
	}
	return nil
}

func (s *Database) UpdateGroupLink(id int, link string) error {
	q := `
		UPDATE group_link SET
			link = $1
		WHERE id = $2
	`
	_, err := s.Exec(q, link, id)
	if err != nil {
		return fmt.Errorf("UpdateGroupLink err: %v", err)
	}
	return nil
}

func (s *Database) EditGroupLinkUserCreator(grlLink string, user_creator int) error {
	q := `
		UPDATE group_link SET
			user_creator = $1
		WHERE link = $2
	`
	_, err := s.Exec(q, user_creator, grlLink)
	if err != nil {
		return fmt.Errorf("EditGroupLinkUserCreator err: %v", err)

	}
	return nil
}