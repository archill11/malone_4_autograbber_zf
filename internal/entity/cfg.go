package entity


const (
	CHORT_LINK_URL_CfgId = "CHORT_LINK_URL"
	Auto_acc_media_gr_CfgId = "auto-acc-media-gr"
	Is_sending_now_CfgId = "is-sending-now"
)


type Cfg struct {
	Id  string `json:"id"`
	Val string `json:"val"`
}
