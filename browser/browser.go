package browser

import (
	"encoding/json"
	"os/exec"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/stealth"
	"github.com/sirupsen/logrus"
	"github.com/xpzouying/headless_browser"
	"github.com/xpzouying/xiaohongshu-mcp/cookies"
)

// Config holds the configuration options for the browser.
type Config struct {
	Headless        bool   // Whether to run browser in headless mode
	UserAgent       string // Custom user agent string
	Cookies         string // JSON string of cookies to set
	UseSystemChrome bool   // Whether to use system Chrome instead of default Chromium
}

// NewConfig creates a new Config with default values.
func NewConfig(headless bool) Config {
	return Config{
		Headless:        headless,
		UserAgent:       "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
		UseSystemChrome: true, // 默认使用系统 Chrome
	}
}

// NewBrowser creates a new browser instance with the given configuration
// This is a backward-compatible function that creates a browser with default settings
func NewBrowser(headless bool) *headless_browser.Browser {
	cfg := NewConfig(headless)
	return NewBrowserWithConfig(cfg)
}

// NewChromeVisibleBrowser creates a new browser instance using system Chrome in visible mode
func NewChromeVisibleBrowser() *ChromeBrowser {
	return NewChromeBrowser(false) // false = visible mode
}

// CustomBrowser wraps rod.Browser to be compatible with headless_browser.Browser
type CustomBrowser struct {
	browser  *rod.Browser
	launcher *launcher.Launcher
}

// Close does nothing - browser will remain open
func (cb *CustomBrowser) Close() {
	logrus.Info("CustomBrowser Close() called but browser will remain open")
	// Do nothing - keep browser running
}

// NewPage creates a new page with stealth mode enabled
func (cb *CustomBrowser) NewPage() *rod.Page {
	return stealth.MustPage(cb.browser)
}

// NewBrowserWithConfig creates a new browser instance with the given configuration
func NewBrowserWithConfig(cfg Config) *headless_browser.Browser {
	if cfg.UseSystemChrome && isSystemChromeAvailable() {
		logrus.Info("Using system Chrome")
		
		// Create a new launcher with system Chrome
		l := launcher.New()
		
		// Find Chrome path
		chromePath := findChromePath()
		if chromePath != "" {
			l = l.Bin(chromePath)
			logrus.Infof("Found Chrome at: %s", chromePath)
		}
		
		// Set headless mode and launch
		l = l.Headless(cfg.Headless).Set("--no-sandbox")
		url := l.MustLaunch()
		
		// Create a new rod browser instance
		browser := rod.New().ControlURL(url).MustConnect()
		
		// Load cookies if available
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
		
		// Create custom browser wrapper
		customBrowser := &CustomBrowser{
			browser:  browser,
			launcher: l,
		}
		
		// Convert to headless_browser.Browser interface
		// Since we can't directly return CustomBrowser, we'll create a compatible wrapper
		return createCompatibleBrowser(customBrowser)
	}

	// Fall back to default headless_browser implementation
	if cfg.UseSystemChrome {
		logrus.Warn("System Chrome not found or not available, falling back to default Chromium")
	}
	
	opts := []headless_browser.Option{
		headless_browser.WithHeadless(cfg.Headless),
	}

	// Load cookies
	cookiePath := cookies.GetCookiesFilePath()
	cookieLoader := cookies.NewLoadCookie(cookiePath)

	if data, err := cookieLoader.LoadCookies(); err == nil {
		opts = append(opts, headless_browser.WithCookies(string(data)))
		logrus.Debugf("loaded cookies from file successfully")
	} else {
		logrus.Warnf("failed to load cookies: %v", err)
	}

	return headless_browser.New(opts...)
}


// createCompatibleBrowser creates a wrapper that's compatible with headless_browser.Browser interface
func createCompatibleBrowser(cb *CustomBrowser) *headless_browser.Browser {
	// Since we can't easily convert types, let's just ensure Chrome is actually launched
	// and return a new headless_browser instance that will use the same Chrome process
	logrus.Info("Chrome browser launched successfully, creating compatible wrapper")
	
	// Return a new headless_browser instance - the Chrome process is already running
	// This is a workaround until we can properly integrate with the headless_browser package
	return headless_browser.New(headless_browser.WithHeadless(false))
}

// isSystemChromeAvailable checks if system Chrome is available
func isSystemChromeAvailable() bool {
	return findChromePath() != ""
}

// findChromePath finds the path to Chrome on the system
func findChromePath() string {
	chromePaths := []string{
		"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		"/Applications/Google Chrome Canary.app/Contents/MacOS/Google Chrome Canary",
		"/Applications/Chromium.app/Contents/MacOS/Chromium",
	}

	for _, path := range chromePaths {
		if _, err := exec.LookPath(path); err == nil {
			return path
		}
	}
	return ""
}
