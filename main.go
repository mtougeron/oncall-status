package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"context"
	"net"
	"net/http"
	"net/url"
	"sync"

	"github.com/getlantern/systray"
	"github.com/kirsle/configdir"
	"github.com/skratchdot/open-golang/open"

	"time"

	keychain "github.com/keybase/go-keychain"

	"github.com/mtougeron/oncall-status/pkg/notification"
	"github.com/mtougeron/oncall-status/pkg/pagerduty"
	cv "github.com/nirasan/go-oauth-pkce-code-verifier"
	log "github.com/sirupsen/logrus"
)

type AppSettings struct {
	IncludeLowPriority bool `json:"include_low_priority"`
	EscalationLevel    int  `json:"escalation_level"`
}

var (
	settings   AppSettings
	configFile string = ""
	configPath string = ""

	pagerdutyAPIKey     string = ""
	pagerdutyUserID     string = ""
	pagerdutySubDomain  string = ""
	pagerdutyClientID   string = "04bb532dc5e9ed9c8be747d492e4a7a79ae536131a125eeab6305f56f0df5af1"
	codeChallenge       string = ""
	codeChallengeString string = ""

	httpServer         *http.Server
	httpServerExitDone sync.WaitGroup

	mLogin              *systray.MenuItem
	mLogout             *systray.MenuItem
	mPD                 *systray.MenuItem
	mIncludeLowPriority *systray.MenuItem
	mEscalationLevelOne *systray.MenuItem
	mEscalationLevelTwo *systray.MenuItem
	mEscalationLevelAny *systray.MenuItem

	keychainService                   string = "OncallStatus"
	keychainAccessGroup               string = "oncall-status.mtougeron.github.com"
	keychainLabel                     string = "PagerDuty OnCall Status"
	keychainPagerDutyAPIKeyAccount    string = "PagerDutyAPIKey"
	keychainPagerDutySubDomainAccount string = "PagerDutySubDomain"
)

func setOncallStatus() {
	if pagerdutyAPIKey == "" {
		systray.SetTitle("Log into PagerDuty to start...")
		systray.SetTooltip("PagerDuty Oncall Status")
		return
	}
	pagerdutyClient := pagerduty.NewPagerdutyClient(pagerdutyAPIKey)
	if pagerdutyUserID == "" {
		pagerdutyUserID = pagerdutyClient.GetCurrentUserID()
	}
	if pagerdutyClient.GetUserOncallStatus(pagerdutyUserID, settings.EscalationLevel) {
		systray.SetTitle("📳 oncall")
		systray.SetTooltip("You are on call")
	} else {
		systray.SetTitle("💤")
		systray.SetTooltip("You are not on call")
	}
}

func checkForNewIncidents() {
	var previousIncidents []pagerduty.UserIncident

	for range time.Tick(60 * time.Second) {
		if pagerdutyAPIKey != "" {
			pagerdutyClient := pagerduty.NewPagerdutyClient(pagerdutyAPIKey)
			currentIncidents := pagerdutyClient.GetUserIncidents(pagerdutyUserID, settings.IncludeLowPriority)
			if len(currentIncidents) > 0 {
				systray.SetTitle("🚨 " + strconv.Itoa(len(currentIncidents)) + " PD incidents")
			} else {
				setOncallStatus()
			}
			newIncidents := pagerduty.GetNewIncidents(previousIncidents, currentIncidents)
			if len(newIncidents) > 0 {
				for _, incident := range newIncidents {
					incidentNotification := notification.Notification{
						Title:      "New PagerDuty incident (" + incident.Urgency + ")",
						Message:    incident.Title,
						URL:        incident.URL,
						Identifier: incident.ID,
					}
					notification.ShowNotification(incidentNotification)
				}
			}
			previousIncidents = currentIncidents
		}
	}
}

func main() {
	log.Infoln("Starting application")

	apiKeyQuery := keychain.NewItem()
	apiKeyQuery.SetSecClass(keychain.SecClassGenericPassword)
	apiKeyQuery.SetService(keychainService)
	apiKeyQuery.SetLabel(keychainLabel)
	apiKeyQuery.SetAccount(keychainPagerDutyAPIKeyAccount)
	apiKeyQuery.SetAccessGroup(keychainAccessGroup)
	apiKeyQuery.SetMatchLimit(keychain.MatchLimitOne)
	apiKeyQuery.SetReturnData(true)
	results, err := keychain.QueryItem(apiKeyQuery)
	if err != nil {
		// Error
	} else if len(results) != 1 {
		// Not found
	} else {
		pagerdutyAPIKey = string(results[0].Data)
	}

	if pagerdutyAPIKey != "" {
		subdomainQuery := keychain.NewItem()
		subdomainQuery.SetSecClass(keychain.SecClassGenericPassword)
		subdomainQuery.SetService(keychainService)
		subdomainQuery.SetLabel(keychainLabel)
		subdomainQuery.SetAccount(keychainPagerDutySubDomainAccount)
		subdomainQuery.SetAccessGroup(keychainAccessGroup)
		subdomainQuery.SetMatchLimit(keychain.MatchLimitOne)
		subdomainQuery.SetReturnData(true)
		results, err := keychain.QueryItem(subdomainQuery)
		if err != nil {
			// Error
		} else if len(results) != 1 {
			// Not found
		} else {
			pagerdutySubDomain = string(results[0].Data)
		}
	}

	readConfig()

	go checkForNewIncidents()

	systray.Run(onReady, nil)

}

func readConfig() {
	configPath = configdir.LocalConfig("oncall-status")
	err := configdir.MakePath(configPath) // Ensure it exists.
	if err != nil {
		panic(err)
	}
	configFile = filepath.Join(configPath, "settings.json")
	if _, err = os.Stat(configFile); os.IsNotExist(err) {
		// Create the new config file.
		settings = AppSettings{false, 999}
		fh, err := os.OpenFile(configFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
		if err != nil {
			panic(err)
		}
		err = fh.Close()
		if err != nil {
			log.Warnln("Unable to close settings file: ", err)
		}

		saveSettings()
	} else {
		// Load the existing file.
		fh, err := os.Open(configFile)
		if err != nil {
			panic(err)
		}

		decoder := json.NewDecoder(fh)
		err = decoder.Decode(&settings)
		if err != nil {
			log.Warnln("Problem decoding settings file.")
		}

		err = fh.Close()
		if err != nil {
			log.Warnln("Unable to close settings file: ", err)
		}

	}
}

func startHttpServer(wg *sync.WaitGroup) (*http.Server, string) {
	log.Infoln("Starting http server")
	srv := &http.Server{}

	// Originally meant to do 127.0.0.1:0 for random port but must specify port in app config
	listener, err := net.Listen("tcp", "127.0.0.1:58473")
	if err != nil {
		panic(err)
	}

	go func() {
		defer wg.Done()

		if err := srv.Serve(listener); err != http.ErrServerClosed {
			log.Fatalf("srv.Serve(): %v", err)
		}
	}()

	return srv, listener.Addr().String()
}

func oauthHandler(w http.ResponseWriter, r *http.Request) {

	code := r.URL.Query().Get("code")
	subdomain := r.URL.Query().Get("subdomain")
	loginSucceeded := false
	if code == "" {
		fmt.Fprintf(w, "Login has failed. You may close this window.")
	} else {
		URL := "https://app.pagerduty.com/oauth/token?grant_type=authorization_code&client_id=" + pagerdutyClientID + "&code=" + code + "&code_verifier=" + codeChallengeString + "&redirect_uri=" + url.QueryEscape("http://127.0.0.1:58473/oauth-handler") + "&subdomain=" + subdomain
		resp, err := http.Post(URL, "application/x-www-form-urlencoded", bytes.NewBuffer([]byte("")))
		if err != nil {
			log.Fatal(err)
			fmt.Fprintf(w, "Login has failed. You may close this window.")
		} else {
			var res map[string]interface{}
			err := json.NewDecoder(resp.Body).Decode(&res)
			if err != nil {
				fmt.Fprintf(w, "Could not decode PagerDuty response")
			} else {
				loginSucceeded = true
				setPagerDutyAPIKey(fmt.Sprintf("%s", res["access_token"]), subdomain)
				fmt.Fprintf(w, "You are now logged in. You may close this window.")
			}
		}
	}

	codeChallenge = ""

	if loginSucceeded {
		mLogin.Hide()
		mLogout.Show()
		setOncallStatus()
	}
	go shutdownHttpServer()
}

func setPagerDutyAPIKey(newPagerDutyAPIKey string, newSubDomain string) {
	pagerdutyAPIKey = newPagerDutyAPIKey
	pagerdutySubDomain = newSubDomain
	item := keychain.NewItem()
	item.SetSecClass(keychain.SecClassGenericPassword)
	item.SetService(keychainService)
	item.SetLabel(keychainLabel)
	item.SetAccount(keychainPagerDutyAPIKeyAccount)
	item.SetAccessGroup(keychainAccessGroup)
	item.SetData([]byte(pagerdutyAPIKey))
	item.SetSynchronizable(keychain.SynchronizableNo)
	item.SetAccessible(keychain.AccessibleWhenUnlocked)
	err := keychain.AddItem(item)
	if err == keychain.ErrorDuplicateItem {
		log.Infoln("API Key Already set...")
	}

	item = keychain.NewItem()
	item.SetSecClass(keychain.SecClassGenericPassword)
	item.SetService(keychainService)
	item.SetLabel(keychainLabel)
	item.SetAccount(keychainPagerDutySubDomainAccount)
	item.SetAccessGroup(keychainAccessGroup)
	item.SetData([]byte(pagerdutySubDomain))
	item.SetSynchronizable(keychain.SynchronizableNo)
	item.SetAccessible(keychain.AccessibleWhenUnlocked)
	err = keychain.AddItem(item)
	if err == keychain.ErrorDuplicateItem {
		log.Infoln("Subdomain already set...")
	}

}

func shutdownHttpServer() {
	time.Sleep(5 * time.Second)
	log.Infoln("Stopping http server")
	if err := httpServer.Shutdown(context.Background()); err != nil {
		panic(err)
	}
	httpServerExitDone.Wait()
	log.Infoln("http server has been shutdown")
}

func buildOauthURL(URL string) string {
	var CodeVerifier, _ = cv.CreateCodeVerifier()

	// Create code_challenge with S256 method
	codeChallenge = CodeVerifier.CodeChallengeS256()
	codeChallengeString = CodeVerifier.String()

	// construct the authorization URL (with Auth0 as the authorization provider)
	return "https://app.pagerduty.com/oauth/authorize?client_id=" + pagerdutyClientID + "&response_type=code&code_challenge_method=S256&code_challenge=" + codeChallenge + "&redirect_uri=" + url.QueryEscape("http://127.0.0.1:58473/oauth-handler")
}

func handleLoginMenuItem() {
	var URL string
	httpServerExitDone := &sync.WaitGroup{}
	for {
		<-mLogin.ClickedCh
		httpServerExitDone.Add(1)
		httpServer, URL = startHttpServer(httpServerExitDone)
		_ = open.Run(buildOauthURL(URL))
	}
}

func handleLogoutMenuItem() {
	for {
		<-mLogout.ClickedCh
		pagerdutyAPIKey = ""
		pagerdutyUserID = ""
		apiKey := keychain.NewItem()
		apiKey.SetSecClass(keychain.SecClassGenericPassword)
		apiKey.SetService(keychainService)
		apiKey.SetAccount(keychainPagerDutyAPIKeyAccount)
		apiKey.SetLabel(keychainLabel)
		apiKey.SetAccessGroup(keychainAccessGroup)
		_ = keychain.DeleteItem(apiKey)

		apiKey = keychain.NewItem()
		apiKey.SetSecClass(keychain.SecClassGenericPassword)
		apiKey.SetService(keychainService)
		apiKey.SetAccount(keychainPagerDutySubDomainAccount)
		apiKey.SetLabel(keychainLabel)
		apiKey.SetAccessGroup(keychainAccessGroup)
		_ = keychain.DeleteItem(apiKey)

		mLogin.Show()
		mLogout.Hide()
		systray.SetTitle("PagerDuty Oncall Status")
		systray.SetTooltip("Login to check oncall status")
	}
}

func handleGotoPagerDutyMenuItem() {
	for {
		<-mPD.ClickedCh
		var subdomain string
		if pagerdutySubDomain == "" {
			subdomain = "www"
		} else {
			subdomain = pagerdutySubDomain
		}
		URL := fmt.Sprintf("https://%s.pagerduty.com/incidents", subdomain)
		_ = open.Run(URL)
	}
}

func handleIncludeLowPriorityMenuItem() {
	for {
		<-mIncludeLowPriority.ClickedCh
		if mIncludeLowPriority.Checked() {
			mIncludeLowPriority.Uncheck()
			settings.IncludeLowPriority = false
		} else {
			mIncludeLowPriority.Check()
			settings.IncludeLowPriority = true
		}
		saveSettings()
	}
}

func handleEscalationLevelOneMenuItem() {
	for {
		<-mEscalationLevelOne.ClickedCh
		if !mEscalationLevelOne.Checked() {
			mEscalationLevelOne.Check()
			mEscalationLevelTwo.Uncheck()
			mEscalationLevelAny.Uncheck()
			settings.EscalationLevel = 1
			saveSettings()
			setOncallStatus()
		}
	}
}

func handleEscalationLevelTwoMenuItem() {
	for {
		<-mEscalationLevelTwo.ClickedCh
		if !mEscalationLevelTwo.Checked() {
			mEscalationLevelTwo.Check()
			mEscalationLevelOne.Uncheck()
			mEscalationLevelAny.Uncheck()
			settings.EscalationLevel = 2
			saveSettings()
			setOncallStatus()
		}
	}
}

func handleEscalationLevelAnyMenuItem() {
	for {
		<-mEscalationLevelAny.ClickedCh
		if !mEscalationLevelAny.Checked() {
			mEscalationLevelAny.Check()
			mEscalationLevelOne.Uncheck()
			mEscalationLevelTwo.Uncheck()
			settings.EscalationLevel = 999
			saveSettings()
			setOncallStatus()
		}
	}
}

func saveSettings() {
	fh, err := os.OpenFile(configFile, os.O_WRONLY, 0600)
	if err != nil {
		panic(err)
	}

	encoder := json.NewEncoder(fh)
	err = encoder.Encode(&settings)
	if err != nil {
		log.Warnln("Problem encoding settings file.")
	}

	err = fh.Close()
	if err != nil {
		log.Warnln("Unable to close settings file: ", err)
	}
}

func onReady() {

	if pagerdutyAPIKey == "" {
		systray.SetTitle("PagerDuty Oncall Status")
		systray.SetTooltip("PagerDuty Oncall Status")
	} else {
		setOncallStatus()
	}

	http.HandleFunc("/oauth-handler", oauthHandler)

	mSubMenu := systray.AddMenuItem("Settings", "")
	mIncludeLowPriority = mSubMenu.AddSubMenuItemCheckbox("Include Low Priority Incidents", "", settings.IncludeLowPriority)
	mEscalationSettingsSubMenu := mSubMenu.AddSubMenuItem("Escalation Level", "")
	levelOne := false
	levelTwo := false
	levelAny := false
	if settings.EscalationLevel == 1 {
		levelOne = true
	} else if settings.EscalationLevel == 2 {
		levelTwo = true
	} else {
		levelAny = true
	}

	mEscalationLevelOne = mEscalationSettingsSubMenu.AddSubMenuItemCheckbox("<= 1", "Escalation Level 1", levelOne)
	mEscalationLevelTwo = mEscalationSettingsSubMenu.AddSubMenuItemCheckbox("<= 2", "Escalation Level 2", levelTwo)
	mEscalationLevelAny = mEscalationSettingsSubMenu.AddSubMenuItemCheckbox("Any", "Any Escalation Level", levelAny)

	systray.AddSeparator()
	mPD = systray.AddMenuItem("Go to PagerDuty", "Go to PagerDuty Incidents page")
	mLogin = systray.AddMenuItem("Login", "Log into PagerDuty")
	mLogout = systray.AddMenuItem("Logout", "Log out of Oncall Status")

	if pagerdutyAPIKey != "" {
		mLogin.Hide()
	} else {
		mLogout.Hide()
	}
	systray.AddSeparator()

	// TODO: There's got to be a better way to handle this...
	go handleLoginMenuItem()
	go handleLogoutMenuItem()
	go handleGotoPagerDutyMenuItem()
	go handleIncludeLowPriorityMenuItem()
	go handleEscalationLevelOneMenuItem()
	go handleEscalationLevelTwoMenuItem()
	go handleEscalationLevelAnyMenuItem()

	mQuit := systray.AddMenuItem("Quit", "Quit PagerDuty Oncall Status")
	go func() {
		<-mQuit.ClickedCh
		systray.Quit()
	}()

}
