package entity

type Bot struct {
	Id                       int    `json:"id"`
	Token                    string `json:"token"`
	Username                 string `json:"username"`
	Firstname                string `json:"first_name"`
	IsDonor                  int    `json:"is_donor"`
	ChId                     int    `json:"ch_id"`
	ChLink                   string `json:"ch_link"`
	GroupLinkId              int    `json:"group_link_id"`
	Lichka                   string `json:"lichka"`
	UserCreator              int    `json:"user_creator"`
	IsDisable                int    `json:"is_disable"`
	ChIsSkam                 int    `json:"ch_is_skam"`
	PersonalLink             string `json:"personal_link"`
	DonorChId                int    `json:"donor_ch_id"`
	IsErrInStat              int    `json:"is_err_in_stat"`
	ToClickShortLink         string `json:"to_click_short_link"`
	ToClickShortLinkToLichka string `json:"to_click_short_link_to_lichka"`

	AdditionalChs []byte `json:"additional_chs"`

	ShortDomenToReplace string `json:"short_domen_to_replace"`
}

type AdditionalCh struct {
	ChId   int    `json:"ch_id"`
	ChLink string `json:"ch_link"`
}
