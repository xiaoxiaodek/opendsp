package rtb

type BidRequest struct {
	ID      string        `json:"id"`
	Imp     []Impression  `json:"imp"`
	Device  *Device       `json:"device,omitempty"`
	User    *User         `json:"user,omitempty"`
	Site    *Site         `json:"site,omitempty"`
	App     *App          `json:"app,omitempty"`
	IsTest  int           `json:"is_test,omitempty"`
}

type Impression struct {
	ID                string   `json:"id"`
	Banner            *Banner  `json:"banner,omitempty"`
	Video             *Video   `json:"video,omitempty"`
	Native            *Native  `json:"native,omitempty"`
	BidFloor          float64  `json:"bidfloor,omitempty"`
	BidFloorCur       string   `json:"bidfloorcur,omitempty"`
	Secure            int      `json:"secure,omitempty"`
	BlockedCategories []string `json:"bcat,omitempty"`
}

type Banner struct {
	W    int32 `json:"w,omitempty"`
	H    int32 `json:"h,omitempty"`
	Pos  int32 `json:"pos,omitempty"`
}

type Video struct {
	W           int32    `json:"w,omitempty"`
	H           int32    `json:"h,omitempty"`
	MinDuration int32    `json:"minduration,omitempty"`
	MaxDuration int32    `json:"maxduration,omitempty"`
	MIMEs       []string `json:"mimes,omitempty"`
	Protocols   []int32  `json:"protocols,omitempty"`
	Linearity   int32    `json:"linearity,omitempty"`
}

type Native struct {
	Request string `json:"request"`
	Ver     string `json:"ver,omitempty"`
}

type Device struct {
	UA         string `json:"ua,omitempty"`
	IP         string `json:"ip,omitempty"`
	OS         string `json:"os,omitempty"`
	OSVersion  string `json:"osv,omitempty"`
	DeviceType int32  `json:"devicetype,omitempty"`
	Make       string `json:"make,omitempty"`
	Model      string `json:"model,omitempty"`
	IFA        string `json:"ifa,omitempty"`
	Carrier    string `json:"carrier,omitempty"`
	Geo        *Geo   `json:"geo,omitempty"`
	W          int32  `json:"w,omitempty"`
	H          int32  `json:"h,omitempty"`
}

type Geo struct {
	Country string  `json:"country,omitempty"`
	City    string  `json:"city,omitempty"`
	Lat     float64 `json:"lat,omitempty"`
	Lon     float64 `json:"lon,omitempty"`
}

type User struct {
	ID       string `json:"id,omitempty"`
	BuyerUID string `json:"buyeruid,omitempty"`
	Gender   string `json:"gender,omitempty"`
	YOB      int32  `json:"yob,omitempty"`
}

type Site struct {
	ID      string   `json:"id,omitempty"`
	Domain  string   `json:"domain,omitempty"`
	Name    string   `json:"name,omitempty"`
	Content *Content `json:"content,omitempty"`
}

type App struct {
	ID      string   `json:"id,omitempty"`
	Name    string   `json:"name,omitempty"`
	Bundle  string   `json:"bundle,omitempty"`
	Content *Content `json:"content,omitempty"`
}

type Content struct {
	ID       string   `json:"id,omitempty"`
	Title    string   `json:"title,omitempty"`
	Cat      []string `json:"cat,omitempty"`
	Keywords string   `json:"keywords,omitempty"`
	URL      string   `json:"url,omitempty"`
}

type BidResponse struct {
	ID         string     `json:"id"`
	SeatBid    []SeatBid  `json:"seatbid,omitempty"`
	BidID      string     `json:"bidid,omitempty"`
	NBR        int        `json:"nbr,omitempty"`
}

type SeatBid struct {
	Bid  []Bid  `json:"bid"`
}

type Bid struct {
	ID      string  `json:"id"`
	ImpID   string  `json:"impid"`
	Price   float64 `json:"price"`
	Adm     string  `json:"adm,omitempty"`
	CrID    string  `json:"crid,omitempty"`
	NURL    string  `json:"nurl,omitempty"`
	Adomain []string `json:"adomain,omitempty"`
}
