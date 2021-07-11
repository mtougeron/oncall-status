package startup

import (
	log "github.com/sirupsen/logrus"

	"os"
	"os/user"
	"path/filepath"
	"sync"
	"text/template"
)

func getStartupPath() string {
	u, err := user.Current()
	if err != nil {
		log.Infoln("user.Current: %v", err)
		return ""
	}
	return u.HomeDir + "/Library/LaunchAgents/com.github.mtougeron.oncall-status.plist"
}

func RunningAtStartup() bool {
	_, err := os.Stat(getStartupPath())
	return err == nil
}

func RemoveStartupItem() {
	if !RunningAtStartup() {
		return
	}
	err := os.Remove(getStartupPath())
	if err != nil {
		log.Infoln("os.Remove: %v", err)
	}
}

var launchdOnce sync.Once
var launchdTemplate *template.Template

func AddStartupItem() {
	path := getStartupPath()
	// Make sure ~/Library/LaunchAgents exists
	err := os.MkdirAll(filepath.Dir(path), 0700)
	if err != nil {
		log.Infoln("os.MkdirAll: %v", err)
		return
	}
	executable, err := os.Executable()
	if err != nil {
		log.Infoln("os.Executable: %v", err)
		return
	}
	f, err := os.Create(path)
	if err != nil {
		log.Infoln("os.Create: %v", err)
		return
	}

	launchdOnce.Do(func() {
		launchdTemplate = template.Must(template.New("launchdConfig").Parse(launchdString))
	})
	err = launchdTemplate.Execute(f,
		struct {
			Name       string
			Label      string
			Executable string
		}{
			"OncallStatus",
			"com.github.mtougeron.oncall-status",
			executable,
		})

	_ = f.Close()
	if err != nil {
		log.Infoln("template.Execute: %v", err)
		return
	}
}

var launchdString = `
<?xml version='1.0' encoding='UTF-8'?>
 <!DOCTYPE plist PUBLIC \"-//Apple Computer//DTD PLIST 1.0//EN\" \"http://www.apple.com/DTDs/PropertyList-1.0.dtd\" >
 <plist version='1.0'>
   <dict>
     <key>Label</key><string>{{.Name}}</string>
     <key>Program</key><string>{{.Executable}}</string>
     <key>StandardOutPath</key><string>/tmp/{{.Label}}-out.log</string>
     <key>StandardErrorPath</key><string>/tmp/{{.Label}}-err.log</string>
     <key>RunAtLoad</key><true/>
   </dict>
</plist>
`
