package application

type GetLinkDefaultByShortQuery struct {
	Short     string
	Code      string
	DoCounter bool
	IP        *string
	IpClient  *string
	ISO       *string
	Country   *string
	Referer   *string
	UserAgent *string
}
