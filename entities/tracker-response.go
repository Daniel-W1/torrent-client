package entities

type TrackerResponse struct {
	Interval int `bencode:"interval"`
	Peers    string `bencode:"peers"`
}