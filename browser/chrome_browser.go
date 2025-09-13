package browser

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/stealth"
	"github.com/sirupsen/logrus"
	"github.com/xpzouying/xiaohongshu-mcp/cookies"
)

// ChromeBrowser is a direct Chrome browser implementation
type ChromeBrowser struct {
	browser  *rod.Browser
	launcher *launcher.Launcher
}

// NewChromeBrowser creates a new Chrome browser instance
func NewChromeBrowser(headless bool) *ChromeBrowser {
	// Find Chrome path
	chromePath := findChromePath()
	if chromePath == "" {
		logrus.Fatal("System Chrome not found")
		return nil
	}

	logrus.Infof("Starting Chrome from: %s", chromePath)

	// Create launcher with system Chrome using automation profile to avoid conflicts
	userDataDir := getAutomationChromeDataDir()
	
	// Copy user cookies to automation profile if they exist
	copyUserCookiesToAutomation(userDataDir)
	
	l := launcher.New().
		Bin(chromePath).
		Headless(headless).
		UserDataDir(userDataDir).
		Set("--no-sandbox").
		Set("--disable-features", "VizDisplayCompositor").
		Set("--remote-debugging-port", "0")

	// Launch Chrome
	url := l.MustLaunch()
	logrus.Infof("Chrome launched at: %s", url)

	// Connect to Chrome
	browser := rod.New().ControlURL(url).MustConnect()

	// Load cookies if available
	loadCookiesForBrowser(browser)

	return &ChromeBrowser{
		browser:  browser,
		launcher: l,
	}
}

// Close does nothing - browser will remain open
func (cb *ChromeBrowser) Close() {
	logrus.Info("Close() called but browser will remain open")
	// Do nothing - keep browser running
}

// NewPage creates a new page with stealth mode enabled
func (cb *ChromeBrowser) NewPage() *rod.Page {
	return stealth.MustPage(cb.browser)
}

// loadCookiesForBrowser loads cookies for the browser
func loadCookiesForBrowser(browser *rod.Browser) {
	cookiePath := cookies.GetCookiesFilePath()
	cookieLoader := cookies.NewLoadCookie(cookiePath)

	if data, err := cookieLoader.LoadCookies(); err == nil {
		var cookieData []*proto.NetworkCookie
		if err := json.Unmarshal(data, &cookieData); err == nil {
			browser.MustSetCookies(cookieData...)
			logrus.Debugf("loaded %d cookies from file successfully", len(cookieData))
		}
	} else {
		logrus.Warnf("failed to load cookies: %v", err)
	}
}

// getUserChromeDataDir returns the path to the user's Chrome data directory
func getUserChromeDataDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		logrus.Warnf("Failed to get user home directory: %v", err)
		return ""
	}
	
	// macOS Chrome user data directory
	chromeDataDir := filepath.Join(homeDir, "Library", "Application Support", "Google", "Chrome")
	
	logrus.Infof("Using Chrome user data directory: %s", chromeDataDir)
	return chromeDataDir
}

// getAutomationChromeDataDir returns a separate Chrome data directory for automation
func getAutomationChromeDataDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		logrus.Warnf("Failed to get user home directory: %v", err)
		return ""
	}
	
	// Create a separate Chrome profile for automation to avoid conflicts
	chromeDataDir := filepath.Join(homeDir, "Library", "Application Support", "Google", "Chrome-Automation")
	
	// Create directory if it doesn't exist
	if err := os.MkdirAll(chromeDataDir, 0755); err != nil {
		logrus.Warnf("Failed to create Chrome automation directory: %v", err)
	}
	
	logrus.Infof("Using Chrome automation data directory: %s", chromeDataDir)
	return chromeDataDir
}

// copyUserCookiesToAutomation copies user's Chrome cookies to automation profile
func copyUserCookiesToAutomation(automationDir string) {
	userChromeDir := getUserChromeDataDir()
	userCookiesPath := filepath.Join(userChromeDir, "Default", "Cookies")
	automationCookiesDir := filepath.Join(automationDir, "Default")
	automationCookiesPath := filepath.Join(automationCookiesDir, "Cookies")
	
	// Create Default directory in automation profile
	if err := os.MkdirAll(automationCookiesDir, 0755); err != nil {
		logrus.Warnf("Failed to create automation Default directory: %v", err)
		return
	}
	
	// Check if user cookies file exists
	if _, err := os.Stat(userCookiesPath); os.IsNotExist(err) {
		logrus.Info("No user cookies file found to copy")
		return
	}
	
	// Copy cookies file
	if err := copyFile(userCookiesPath, automationCookiesPath); err != nil {
		logrus.Warnf("Failed to copy user cookies: %v", err)
	} else {
		logrus.Info("Successfully copied user cookies to automation profile")
	}
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()
	
	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()
	
	_, err = destFile.ReadFrom(sourceFile)
	return err
}
