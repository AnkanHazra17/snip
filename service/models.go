package main

type createLinkReq struct {
	URL string `json:"url"`
}

type createLinkResp struct {
	Code     string `json:"code"`
	ShortURL string `json:"short_url"`
}
