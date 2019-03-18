package fortnitego

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"log"
)

// Epic API endpoints
const (
	oauthTokenURL    = "https://account-public-service-prod03.ol.epicgames.com/account/api/oauth/token"
	oauthExchangeURL = "https://account-public-service-prod03.ol.epicgames.com/account/api/oauth/exchange"
	accountLookupURL = "https://persona-public-service-prod06.ol.epicgames.com/persona/api/public/account"
	accountInfoURL   = "https://account-public-service-prod03.ol.epicgames.com/account/api/public/account"
	killSessionURL   = "https://account-public-service-prod03.ol.epicgames.com/account/api/oauth/sessions/kill"

	serverStatusURL    = "https://lightswitch-public-service-prod06.ol.epicgames.com/lightswitch/api/service/bulk/status?serviceId=Fortnite"
	accountStatsURL    = "https://fortnite-public-service-prod11.ol.epicgames.com/fortnite/api/stats/accountId"
	accountStatsV2URL  = "https://fortnite-public-service-prod11.ol.epicgames.com/fortnite/api/statsv2/account"
	winsLeaderboardURL = "https://fortnite-public-service-prod11.ol.epicgames.com/fortnite/api/leaderboards/type/global/stat/br_placetop1_%v_m0%v/window/weekly"
)

// Platform types
const (
	PC   = "pc"
	Xbox = "xb1"
	PS4  = "ps4"
)

// tokenResponse defines the response collected by a request to the OAUTH token endpoint.
type tokenResponse struct {
	AccessToken      string `json:"access_token"`
	ExpiresIn        int    `json:"expires_in"`
	ExpiresAt        string `json:"expires_at"`
	RefreshToken     string `json:"refresh_token"`
	RefreshExpires   int    `json:"refresh_expires"`
	RefreshExpiresAt string `json:"refresh_expires_at"`
	AccountID        string `json:"account_id"`
	ClientID         string `json:"client_id"`
}

// tokenResponse defines the response collected by a request to the OAUTH exchange endpoint.
type exchangeResponse struct {
	ExpiresInSeconds int    `json:"expiresInSeconds"`
	Code             string `json:"code"`
	CreatingClientID string `json:"creatingClientId"`
}

// lookupResponse defines the response collected by a request to the persona lookup endpoint.
type lookupResponse struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
}

// statsResponse defines the response collected by a request to the battle royal stats endpoint.
type statsResponse []statsRecord

type statsResponseV2 statsRecordV2

// statsRecord defines a single entry in a statsResponse.
type statsRecord struct {
	Name      string `json:"name"`
	Value     int    `json:"value"`
	Window    string `json:"window"`
	OwnerType int    `json:"ownerType"`
}

type statsRecordV2 struct {
	StartTime			int								`json:"startTime"`
	EndTime				int								`json:"endTime"`
	Stats					map[string]int		`json:"stats"`
	AccountID			string						`json:"accountId"`
}

// Player is the hierarchical struct used to contain information regarding a player's account info and stats.
type Player struct {
	AccountInfo AccountInfo
	Stats       Stats
}

// AccountInfo contains basic information about the user.
type AccountInfo struct {
	AccountID string
	Username  string
	Platform  string
}

// Stats is the structure which holds the player's stats for the 3 different game modes offered in Battle Royal.
type Stats struct {
	Solo  statDetails
	Duo   statDetails
	Squad statDetails
}

// statDetails is the specific statistics for any given group mode.
type statDetails struct {
	Wins           int
	Top3           int `json:",omitempty"` // Squad-only
	Top5           int `json:",omitempty"` // Duo-only
	Top6           int `json:",omitempty"` // Squad-only
	Top10          int `json:",omitempty"` // Solo-only
	Top12          int `json:",omitempty"` // Duo-only
	Top25          int `json:",omitempty"` // Solo-only
	KillDeathRatio string
	WinPercentage  string
	Matches        int
	Kills          int
	MinutesPlayed  int
	KillsPerMatch  string
	KillsPerMinute string
	Score          int
}

// GlobalWinsLeaderboard contains an array of the top X players by wins on a specific platform and party mode.
type GlobalWinsLeaderboard []leaderboardEntry

// leaderboardEntry defines a single entry in a GlobalWinsLeaderboard object.
type leaderboardEntry struct {
	DisplayName string
	Rank        int
	Wins        int
}

// QueryPlayer looks up a player by their username and platform, and returns information about that player, namely, the
// statistics for the 3 different party modes.
// func (s *Session) QueryPlayer(name string, accountId string, platform string) (*Player, error) {
// 	if name == "" && accountId == "" {
// 		return nil, errors.New("no player name or id provided")
// 	}
// 	switch platform {
// 	case PC, Xbox, PS4:
// 	default:
// 		return nil, errors.New("invalid platform specified")
// 	}
//
// 	if name != "" && accountId == "" {
// 		userInfo, err := s.findUserInfo(name)
// 		if err != nil {
// 			return nil, err
// 		}
// 		accountId = userInfo.ID
// 	}
//
// 	sr, err := s.QueryPlayerById(accountId)
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	acctInfoMap, err := s.getAccountNames([]string{accountId})
// 	if err != nil {
// 		return nil, err
// 	}
// 	cleanAcctID := strings.Replace(accountId, "-", "", -1)
//
// 	return &Player{
// 		AccountInfo: AccountInfo{
// 			AccountID: accountId,
// 			Username:  acctInfoMap[cleanAcctID],
// 			Platform:  platform,
// 		},
// 		Stats: s.mapStats(sr, platform),
// 	}, nil
// }

// QueryPlayer looks up a player by their username and platform, and returns information about that player, namely, the
// statistics for the 3 different party modes.
func (s *Session) QueryPlayerV2(name string, accountId string) (*Player, error) {
	if name == "" && accountId == "" {
		return nil, errors.New("no player name or id provided")
	}

	if name != "" && accountId == "" {
		userInfo, err := s.findUserInfo(name)
		if err != nil {
			return nil, err
		}
		accountId = userInfo.ID
	}

	sr, err := s.QueryPlayerByIdV2(accountId)
	if err != nil {
		log.Println("ERR: ", err)
		return nil, err
	}

	cleanAcctID := strings.Replace(accountId, "-", "", -1)

	return &Player{
		AccountInfo: AccountInfo{
			AccountID: cleanAcctID,
		},
		Stats: s.mapStats(sr),
	}, nil
}

// func (s *Session) QueryPlayerById(accountId string) (*statsResponse, error) {
// 	u := fmt.Sprintf("%v/", accountStatsURL, accountId)
// 	req, err := s.client.NewRequest(http.MethodGet, u, nil)
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	// Set authorization to use access token.
// 	req.Header.Set("Authorization", fmt.Sprintf("%v %v", AuthBearer, s.AccessToken))
//
// 	sr := &statsResponse{}
// 	resp, err := s.client.Do(req, sr)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer resp.Body.Close()
//
// 	if len(*sr) == 0 {
// 		return nil, errors.New("no statistics found for player " + accountId)
// 	}
//
// 	return sr, nil
// }

func (s *Session) QueryPlayerByIdV2(accountId string) (*statsResponseV2, error) {
	u := fmt.Sprintf("%v/%s", accountStatsV2URL, accountId)
	req, err := s.client.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}

	// Set authorization to use access token.
	req.Header.Set("Authorization", fmt.Sprintf("%v %v", AuthBearer, s.AccessToken))

	sr := &statsResponseV2{}
	resp, err := s.client.Do(req, sr)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if sr == nil {
		return nil, errors.New("no statistics found for player " + accountId)
	}

	return sr, nil
}

// findUserInfo requests additional account information by a username.
func (s *Session) findUserInfo(username string) (*lookupResponse, error) {
	req, err := s.client.NewRequest(http.MethodGet, accountLookupURL+"/lookup?q="+url.QueryEscape(username), nil)
	if err != nil {
		return nil, err
	}

	// Set authorization to use access token.
	req.Header.Set("Authorization", fmt.Sprintf("%v %v", AuthBearer, s.AccessToken))

	ret := &lookupResponse{}
	resp, err := s.client.Do(req, ret)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if ret.ID == "" {
		return nil, errors.New("player not found")
	}

	return ret, nil
}

// Name identifiers for group type. Used in parsing URLs and responses.
const (
	Solo  = "_defaultsolo"
	Duo   = "_defaultduo"
	Squad = "_defaultsquad"
)

// getStatType is a simple helper function to return the party type if present in a given string.
func getStatType(seed string) string {
	switch {
	case strings.Contains(seed, Solo):
		return Solo
	case strings.Contains(seed, Duo):
		return Duo
	default: // _p9
		return Squad
	}
}

// mapStats takes a statsResponse object and converts it into a Stats object. It parses the JSON returned from Epic
// regarding a player's stats, and maps it accordingly based on party type, as well as calculates several useful ratios.
func (s *Session) mapStats(stats *statsResponseV2) Stats {
	// Initialize new map with stat details objects based on group type.
	groups := make(map[string]*statDetails)
	groups[Solo] = &statDetails{}
	groups[Duo] = &statDetails{}
	groups[Squad] = &statDetails{}

	// Loop through the stats for a specific user properly sorting and organizing by group type into their own objects.
	for key, record := range stats.Stats {
		switch {
		case strings.Contains(key, "placetop1_"):
			if getStatType(key) == Squad ||  getStatType(key) == Duo || getStatType(key) == Solo {
				groups[getStatType(key)].Wins = groups[getStatType(key)].Wins + record
			}
		case strings.Contains(key, "placetop3_"):
			if getStatType(key) == Squad ||  getStatType(key) == Duo || getStatType(key) == Solo {
				groups[getStatType(key)].Top3 = groups[getStatType(key)].Top3 + record
			}
		case strings.Contains(key, "placetop5_"):
			if getStatType(key) == Squad ||  getStatType(key) == Duo || getStatType(key) == Solo {
				groups[getStatType(key)].Top5 = groups[getStatType(key)].Top5 + record
			}
		case strings.Contains(key, "placetop6_"):
			if getStatType(key) == Squad ||  getStatType(key) == Duo || getStatType(key) == Solo {
				groups[getStatType(key)].Top6 = groups[getStatType(key)].Top6 + record
			}
		case strings.Contains(key, "placetop10_"):
			if getStatType(key) == Squad ||  getStatType(key) == Duo || getStatType(key) == Solo {
				groups[getStatType(key)].Top10 = groups[getStatType(key)].Top10 + record
			}
		case strings.Contains(key, "placetop12_"):
			if getStatType(key) == Squad ||  getStatType(key) == Duo || getStatType(key) == Solo {
				groups[getStatType(key)].Top12 = groups[getStatType(key)].Top12 + record
			}
		case strings.Contains(key, "placetop25_"):
			if getStatType(key) == Squad ||  getStatType(key) == Duo || getStatType(key) == Solo {
				groups[getStatType(key)].Top25 = groups[getStatType(key)].Top25 + record
			}
		case strings.Contains(key, "matchesplayed_"):
			if getStatType(key) == Squad ||  getStatType(key) == Duo || getStatType(key) == Solo {
				groups[getStatType(key)].Matches = groups[getStatType(key)].Matches + record
			}
		case strings.Contains(key, "kills_"):
			if strings.Contains(key, Squad) || strings.Contains(key, Duo) || strings.Contains(key, Solo) {
				log.Println("DEBUG: RECORD ", record)
				groups[getStatType(key)].Kills = groups[getStatType(key)].Kills + record
				log.Println("DEBUG: KILLS TOTAL ", groups[getStatType(key)].Kills)
			}
		case strings.Contains(key, "score_"):
			if getStatType(key) == Squad ||  getStatType(key) == Duo || getStatType(key) == Solo {
				groups[getStatType(key)].Score = groups[getStatType(key)].Score + record
			}
		case strings.Contains(key, "minutesplayed_"):
			if getStatType(key) == Squad ||  getStatType(key) == Duo || getStatType(key) == Solo {
				groups[getStatType(key)].MinutesPlayed = groups[getStatType(key)].MinutesPlayed + record
			}
		}
	}

	// Build new return object using the prepared map data.
	ret := Stats{
		Solo:  *groups[Solo],
		Duo:   *groups[Duo],
		Squad: *groups[Squad],
	}

	// Calculate additional information such as kill/death ratios, win percentages, etc.
	calculateStatsRatios(&ret.Solo)
	calculateStatsRatios(&ret.Duo)
	calculateStatsRatios(&ret.Squad)

	// Return built Stats object.
	return ret
}

// calculateStatsRatios takes a party-specific statDetails object and performs ratio calculations on specific data to
// provide kill death ratio, win percentage, and kills per minute/match.
func calculateStatsRatios(s *statDetails) {
	s.KillDeathRatio = strconv.FormatFloat(ratio(s.Kills, s.Matches-s.Wins), 'f', 2, 64)
	s.WinPercentage = strconv.FormatFloat(ratio(s.Wins, s.Matches)*100, 'f', 2, 64)
	s.KillsPerMinute = strconv.FormatFloat(ratio(s.Kills, s.MinutesPlayed), 'f', 2, 64)
	s.KillsPerMatch = strconv.FormatFloat(ratio(s.Kills, s.Matches), 'f', 2, 64)
}

// ratio is a helper function to perform float division without causing a division by 0 panic.
func ratio(a, b int) float64 {
	if b == 0 {
		return 0
	}

	return float64(a) / float64(b)
}

type leaderboardResponse struct {
	StatName   string `json:"statName"`
	StatWindow string `json:"statWindow"`
	Entries    []struct {
		AccountID string `json:"accountId"`
		Value     int    `json:"value"`
		Rank      int    `json:"rank"`
	} `json:"entries"`
}

// GetWinsLeaderboard returns the top 50 players and their rank position based on global wins for a specific platform,
// and party/group type.
func (s *Session) GetWinsLeaderboard(platform, groupType string) (*GlobalWinsLeaderboard, error) {
	qp := url.Values{}
	qp.Add("ownertype", "1")     // unknown
	qp.Add("pageNumber", "0")    // not implemented in-game?
	qp.Add("itemsPerPage", "50") // definable up to how many?

	// Prepare new request to obtain leaderboard information.
	u := fmt.Sprintf(winsLeaderboardURL, platform, groupType) + "?" + qp.Encode()
	req, err := s.client.NewRequest(http.MethodPost, u, nil)
	if err != nil {
		return nil, err
	}

	// Use access token.
	req.Header.Set("Authorization", fmt.Sprintf("%v %v", AuthBearer, s.AccessToken))

	// Perform request and collect response data into leaderboardResponse object.
	lr := &leaderboardResponse{}
	resp, err := s.client.Do(req, lr)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Loop through entries received building an array of account IDs.
	var accountIDs []string
	for _, item := range lr.Entries {
		accountIDs = append(accountIDs, item.AccountID)
	}

	// Send account IDs off to be queried so we can collect their human-readable display name (Epic Username).
	acctInfoMap, err := s.getAccountNames(accountIDs)
	if err != nil {
		return nil, err
	}

	// Initialize return object, and look through entries once more mapping their username as display name obtained
	// just before.
	ret := GlobalWinsLeaderboard{}
	for _, b := range lr.Entries {
		cleanAcctID := strings.Replace(b.AccountID, "-", "", -1)
		ret = append(ret, leaderboardEntry{
			DisplayName: acctInfoMap[cleanAcctID],
			Rank:        b.Rank,
			Wins:        b.Value,
		})
	}

	// Return new leaderboard object.
	return &ret, nil
}

// getAccountNames is a helper to query a bulk amount of account IDs to get additional information on them, in
// particular, their username.
func (s *Session) getAccountNames(ids []string) (map[string]string, error) {
	// Build query parameter string based on account IDs supplied.
	var p string
	for _, id := range ids {
		// Note: Epic strips the hyphens '-' in the request.
		p += "accountId=" + strings.Replace(id, "-", "", -1) + "&"
	}
	p = p[:len(p)-1] // Strip trailing '&'.

	// Prepare new request to the persona server for information about these accounts.
	req, err := s.client.NewRequest(http.MethodGet, accountInfoURL+"?"+p, nil)
	if err != nil {
		return nil, err
	}

	// Set authorization header to use our access token.
	req.Header.Set("Authorization", fmt.Sprintf("%v %v", AuthBearer, s.AccessToken))

	// Perform query and collect response into an array of lookupResponse objects.
	var data []lookupResponse
	resp, err := s.client.Do(req, &data)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Prepare return map where we will map the account ID to the newly-collected username (DisplayName).
	ret := make(map[string]string)
	for _, a := range data {
		ret[a.ID] = a.DisplayName
	}

	return ret, nil
}

// statusResponse is the expected response from a server status check.
type statusResponse []struct {
	Status         string      `json:"status"`
	Message        string      `json:"message"`
	MaintenanceURI interface{} `json:"maintenanceUri"`
}

// CheckStatus checks the status of the Fortnite game service. Will return false with error containing the status
// message from Epic.
func (s *Session) CheckStatus() (bool, error) {
	// Prepare new request.
	req, err := s.client.NewRequest(http.MethodGet, serverStatusURL, nil)
	if err != nil {
		return false, err
	}

	// Set authorization header to use access token.
	req.Header.Set("Authorization", fmt.Sprintf("%v %v", AuthBearer, s.AccessToken))

	// Perform request and decode response into a statusResponse object.
	var sr statusResponse
	resp, err := s.client.Do(req, &sr)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	// Ensure at least one value of the array has been provided to prevent panic.
	if len(sr) == 0 {
		return false, errors.New("no status response received")
	}

	// Switch between the status string to determine whether the service is up or down.
	switch sr[0].Status {
	case "UP":
		// Never return the message here since it doesn't seem to be removed when the server resume online status.
		return true, nil
	default:
		return false, errors.New("service is down: " + sr[0].Message)
	}
}
