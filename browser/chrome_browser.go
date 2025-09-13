package browser

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

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
	// Connect to existing Chrome instance - user's current Chrome with current profile
	browser, err := connectToExistingChrome()
	if err != nil {
		logrus.Fatal("Cannot connect to your current Chrome. Please enable remote debugging:\n" +
			"1. Close Chrome completely\n" +
			"2. Start Chrome with: open -a 'Google Chrome' --args --remote-debugging-port=9222\n" +
			"3. Then run this program again")
		return nil
	}

	logrus.Info("Connected to your current Chrome - will create new tab with your current profile")
	
	return &ChromeBrowser{
		browser:  browser,
		launcher: nil,
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

// connectToExistingChrome tries to connect to an existing Chrome instance
func connectToExistingChrome() (*rod.Browser, error) {
	// Try common Chrome debugging ports
	ports := []string{"9222", "9223", "9224"}

	for _, port := range ports {
		url := fmt.Sprintf("http://localhost:%s", port)
		client := &http.Client{Timeout: 2 * time.Second}

		// Check if Chrome is running on this port
		resp, err := client.Get(url + "/json/version")
		if err != nil {
			continue
		}
		resp.Body.Close()

		if resp.StatusCode == 200 {
			// Try to connect
			browser := rod.New().ControlURL(url)
			if err := browser.Connect(); err == nil {
				return browser, nil
			}
		}
	}

	return nil, fmt.Errorf("no existing Chrome instance found")
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

// getUserChromeDataDir returns the path to the user's Chrome default profile directory
func getUserChromeDataDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		logrus.Warnf("Failed to get user home directory: %v", err)
		return ""
	}

	// macOS Chrome default profile directory
	chromeProfileDir := filepath.Join(homeDir, "Library", "Application Support", "Google", "Chrome", "Default")

	logrus.Infof("Using Chrome default profile directory: %s", chromeProfileDir)
	return chromeProfileDir
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
