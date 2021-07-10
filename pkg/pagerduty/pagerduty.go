package pagerduty

import (
	pagerduty "github.com/PagerDuty/go-pagerduty"
	log "github.com/sirupsen/logrus"
	"github.com/skratchdot/open-golang/open"
)

// Client has the pagerduty.Client
type Client struct {
	*pagerduty.Client
}

// UserIncident a simple struct of an incident
type UserIncident struct {
	Title string
	URL   string
	ID    string
}

// Open opens the URL
func (ui *UserIncident) Open() {
	err := open.Run(ui.URL)
	if err != nil {
		log.Warnln("Error opening incident URL: ", err)
	}
}

// NewPagerdutyClient Create the pagerduty client
func NewPagerdutyClient(authToken string) *Client {
	return &Client{pagerduty.NewOAuthClient(authToken)}
}

func (pd *Client) GetCurrentUserID() string {
	var currUserOpts pagerduty.GetCurrentUserOptions
	currUser, _ := pd.GetCurrentUser(currUserOpts)
	return currUser.ID
}

// GetUserOncallStatus is the user oncall or not
func (pd *Client) GetUserOncallStatus(userID string, escalationLevel int) bool {

	var userOpts pagerduty.GetUserOptions
	if userID == "" {
		var currUserOpts pagerduty.GetCurrentUserOptions
		currUser, _ := pd.GetCurrentUser(currUserOpts)
		userID = currUser.ID
	}

	_, err := pd.GetUser(userID, userOpts)
	if err != nil {
		log.Errorln("Error from PD API:", err)
		return false
	}

	var oncalls []pagerduty.OnCall
	var opts pagerduty.ListOnCallOptions
	opts.Limit = 100
	opts.UserIDs = append(opts.UserIDs, userID)

	more := true
	for more {
		ocs, err := pd.ListOnCalls(opts)
		if err != nil {
			log.Errorln("Error from PD API:", err)
			return false
		}
		oncalls = append(oncalls, ocs.OnCalls...)
		more = ocs.APIListObject.More
		opts.Offset = opts.Offset + ocs.APIListObject.Limit
	}

	for _, oncall := range oncalls {
		if oncall.EscalationLevel <= uint(escalationLevel) {
			return true
		}
	}
	return false
}

// GetUserIncidents get a list of incidents assigned to a user
func (pd *Client) GetUserIncidents(userID string, includeLowPriority bool) []UserIncident {

	var response []UserIncident
	var userOpts pagerduty.GetUserOptions
	if userID == "" {
		return response
	}
	_, err := pd.GetUser(userID, userOpts)
	if err != nil {
		log.Errorln("Error from PD API:", err)
		return response
	}

	var incidentOpts pagerduty.ListIncidentsOptions
	var incidents []pagerduty.Incident

	incidentOpts.UserIDs = append(incidentOpts.UserIDs, userID)
	incidentOpts.Statuses = append(incidentOpts.Statuses, "triggered", "acknowledged")
	if !includeLowPriority {
		incidentOpts.Urgencies = append(incidentOpts.Urgencies, "high")
	}

	more := true
	for more {
		res, err := pd.ListIncidents(incidentOpts)
		if err != nil {
			log.Errorln("Error from PD API:", err)
			return response
		}
		incidents = append(incidents, res.Incidents...)
		more = res.APIListObject.More
		incidentOpts.Offset = incidentOpts.Offset + res.APIListObject.Limit
	}

	for _, incident := range incidents {
		response = append(response, UserIncident{
			ID:    incident.Id,
			Title: incident.Title,
			URL:   incident.HTMLURL,
		})
	}

	return response
}
