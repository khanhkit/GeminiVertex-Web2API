package browser

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

// BrowserType identifies which Chromium-based browser to use.
type BrowserType string

const (
	BrowserChrome BrowserType = "Chrome"
	BrowserEdge   BrowserType = "Edge"
)

// ChromeProfile holds profile metadata for a Chromium-based browser.
type ChromeProfile struct {
	Name        string
	DisplayName string
	Path        string
}

// getUserDataDir returns the User Data directory for the given browser.
func getUserDataDir(bt BrowserType) string {
	switch bt {
	case BrowserEdge:
		if runtime.GOOS == "windows" {
			return filepath.Join(os.Getenv("LOCALAPPDATA"), "Microsoft", "Edge", "User Data")
		} else if runtime.GOOS == "darwin" {
			return filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "Microsoft Edge")
		}
		return filepath.Join(os.Getenv("HOME"), ".config", "microsoft-edge")
	default: // BrowserChrome
		if runtime.GOOS == "windows" {
			return filepath.Join(os.Getenv("LOCALAPPDATA"), "Google", "Chrome", "User Data")
		} else if runtime.GOOS == "darwin" {
			return filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "Google", "Chrome")
		}
		return filepath.Join(os.Getenv("HOME"), ".config", "google-chrome")
	}
}

// findBrowserPath returns the path to the browser executable.
func findBrowserPath(bt BrowserType) string {
	switch bt {
	case BrowserEdge:
		if runtime.GOOS == "windows" {
			paths := []string{
				filepath.Join(os.Getenv("PROGRAMFILES"), "Microsoft", "Edge", "Application", "msedge.exe"),
				filepath.Join(os.Getenv("PROGRAMFILES(X86)"), "Microsoft", "Edge", "Application", "msedge.exe"),
				filepath.Join(os.Getenv("LOCALAPPDATA"), "Microsoft", "Edge", "Application", "msedge.exe"),
			}
			for _, p := range paths {
				if _, err := os.Stat(p); err == nil {
					return p
				}
			}
		} else if runtime.GOOS == "darwin" {
			p := "/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge"
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
		return "microsoft-edge"
	default: // BrowserChrome
		if runtime.GOOS == "windows" {
			paths := []string{
				filepath.Join(os.Getenv("PROGRAMFILES"), "Google", "Chrome", "Application", "chrome.exe"),
				filepath.Join(os.Getenv("PROGRAMFILES(X86)"), "Google", "Chrome", "Application", "chrome.exe"),
				filepath.Join(os.Getenv("LOCALAPPDATA"), "Google", "Chrome", "Application", "chrome.exe"),
			}
			for _, p := range paths {
				if _, err := os.Stat(p); err == nil {
					return p
				}
			}
		}
		return "chrome"
	}
}

// ListBrowserProfiles lists available profiles for the given browser.
func ListBrowserProfiles(bt BrowserType) ([]ChromeProfile, error) {
	userDataDir := getUserDataDir(bt)

	entries, err := os.ReadDir(userDataDir)
	if err != nil {
		return nil, fmt.Errorf("cannot read %s user data dir: %v", bt, err)
	}

	profileNames := make(map[string]string)
	localStatePath := filepath.Join(userDataDir, "Local State")
	if data, err := os.ReadFile(localStatePath); err == nil {
		var localState struct {
			Profile struct {
				InfoCache map[string]struct {
					Name string `json:"name"`
				} `json:"info_cache"`
			} `json:"profile"`
		}
		if json.Unmarshal(data, &localState) == nil {
			for k, v := range localState.Profile.InfoCache {
				profileNames[k] = v.Name
			}
		}
	}

	var profiles []ChromeProfile
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if name == "Default" || strings.HasPrefix(name, "Profile ") {
			cookiesPath := filepath.Join(userDataDir, name, "Network", "Cookies")
			if _, err := os.Stat(cookiesPath); err == nil {
				displayName := profileNames[name]
				if displayName == "" {
					displayName = name
				}
				profiles = append(profiles, ChromeProfile{
					Name:        name,
					DisplayName: displayName,
					Path:        filepath.Join(userDataDir, name),
				})
			}
		}
	}

	sort.Slice(profiles, func(i, j int) bool {
		if profiles[i].Name == "Default" {
			return true
		}
		if profiles[j].Name == "Default" {
			return false
		}
		return profiles[i].Name < profiles[j].Name
	})

	return profiles, nil
}

// ListChromeProfiles lists Chrome profiles (kept for backward compatibility).
func ListChromeProfiles() ([]ChromeProfile, error) {
	return ListBrowserProfiles(BrowserChrome)
}

// getChomeUserDataDir returns the Chrome user data directory (kept for backward compatibility).
func getChomeUserDataDir() string {
	return getUserDataDir(BrowserChrome)
}

// findChromePath returns the Chrome executable path (kept for backward compatibility).
func findChromePath() string {
	return findBrowserPath(BrowserChrome)
}

// FetchCookiesFromProfileWithBrowser fetches Google cookies from the given profile using the specified browser.
func FetchCookiesFromProfileWithBrowser(profile ChromeProfile, bt BrowserType) (map[string]string, error) {
	browserPath := findBrowserPath(bt)
	port := 20000 + time.Now().Nanosecond()%10000

	userDataDir := getUserDataDir(bt)

	cmd := exec.Command(browserPath,
		fmt.Sprintf("--remote-debugging-port=%d", port),
		fmt.Sprintf("--user-data-dir=%s", userDataDir),
		fmt.Sprintf("--profile-directory=%s", profile.Name),
		"--headless=new",
		"--disable-gpu",
		"--no-first-run",
		"--no-default-browser-check",
		"about:blank",
	)

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start %s: %v", bt, err)
	}
	defer func() {
		cmd.Process.Kill()
		cmd.Wait()
	}()

	var resp *http.Response
	var err error
	for i := 0; i < 15; i++ {
		time.Sleep(500 * time.Millisecond)
		resp, err = http.Get(fmt.Sprintf("http://127.0.0.1:%d/json", port))
		if err == nil {
			break
		}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s DevTools: %v", bt, err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var targets []struct {
		WebSocketDebuggerUrl string `json:"webSocketDebuggerUrl"`
	}
	if err := json.Unmarshal(body, &targets); err != nil || len(targets) == 0 {
		return nil, fmt.Errorf("failed to get debugger URL")
	}

	wsURL := targets[0].WebSocketDebuggerUrl
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to WebSocket: %v", err)
	}
	defer conn.Close()

	conn.WriteJSON(map[string]interface{}{"id": 1, "method": "Page.enable"})
	conn.WriteJSON(map[string]interface{}{"id": 2, "method": "Network.enable"})
	conn.WriteJSON(map[string]interface{}{
		"id":     3,
		"method": "Page.navigate",
		"params": map[string]interface{}{"url": "https://gemini.google.com"},
	})

	time.Sleep(5 * time.Second)

	conn.WriteJSON(map[string]interface{}{"id": 4, "method": "Network.getAllCookies"})

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	resultChan := make(chan map[string]string, 1)
	errChan := make(chan error, 1)

	go func() {
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				errChan <- err
				return
			}
			var result struct {
				ID     int `json:"id"`
				Result struct {
					Cookies []struct {
						Name   string `json:"name"`
						Value  string `json:"value"`
						Domain string `json:"domain"`
					} `json:"cookies"`
				} `json:"result"`
			}
			if err := json.Unmarshal(message, &result); err != nil {
				continue
			}
			if result.ID == 4 {
				cookies := make(map[string]string)
				for _, c := range result.Result.Cookies {
					if (c.Name == "__Secure-1PSID" || c.Name == "__Secure-1PSIDTS") && strings.Contains(c.Domain, "google.com") {
						cookies[c.Name] = c.Value
					}
				}
				resultChan <- cookies
				return
			}
		}
	}()

	select {
	case cookies := <-resultChan:
		if cookies["__Secure-1PSID"] == "" {
			return nil, fmt.Errorf("cookie not found, please login to Google in this profile")
		}
		return cookies, nil
	case err := <-errChan:
		return nil, err
	case <-ctx.Done():
		return nil, fmt.Errorf("timeout waiting for cookies")
	}
}

// FetchCookiesFromProfile fetches cookies using Chrome (kept for backward compatibility).
func FetchCookiesFromProfile(profile ChromeProfile) (map[string]string, error) {
	return FetchCookiesFromProfileWithBrowser(profile, BrowserChrome)
}

// chooseBrowser prompts the user to select Chrome or Edge and returns the chosen BrowserType.
func chooseBrowser(reader *bufio.Reader) BrowserType {
	fmt.Println("Select browser:")
	fmt.Println("  [1] Chrome")
	fmt.Println("  [2] Edge")
	fmt.Print("Enter choice (default: 1): ")

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "2" {
		return BrowserEdge
	}
	return BrowserChrome
}

func RunFetchCookies() error {
	reader := bufio.NewReader(os.Stdin)

	bt := chooseBrowser(reader)

	fmt.Printf("\n=== %s Cookie Fetcher ===\n", bt)
	fmt.Printf("\n[!] Please close %s browser before proceeding!\n", bt)
	fmt.Println("    (Press Enter to continue...)")
	reader.ReadString('\n')

	fmt.Printf("Scanning %s profiles...\n", bt)
	profiles, err := ListBrowserProfiles(bt)
	if err != nil {
		return err
	}

	if len(profiles) == 0 {
		return fmt.Errorf("no %s profiles found", bt)
	}

	fmt.Printf("\nAvailable %s profiles:\n", bt)
	for i, p := range profiles {
		if p.Name == "Default" {
			fmt.Printf("  [%d] %s (default account)\n", i+1, p.DisplayName)
		} else {
			fmt.Printf("  [%d] %s → __%s\n", i+1, p.DisplayName, strings.ReplaceAll(p.DisplayName, " ", "_"))
		}
	}

	fmt.Println("\nEnter profile numbers (e.g., 1,2,3) or ALL:")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	var selectedProfiles []ChromeProfile
	if strings.ToUpper(input) == "ALL" {
		selectedProfiles = profiles
	} else {
		parts := strings.Split(input, ",")
		for _, p := range parts {
			p = strings.TrimSpace(p)
			idx, err := strconv.Atoi(p)
			if err != nil || idx < 1 || idx > len(profiles) {
				fmt.Printf("Invalid selection: %s\n", p)
				continue
			}
			selectedProfiles = append(selectedProfiles, profiles[idx-1])
		}
	}

	if len(selectedProfiles) == 0 {
		return fmt.Errorf("no profiles selected")
	}

	fmt.Printf("\nFetching cookies from %d profile(s)...\n", len(selectedProfiles))
	fmt.Printf("Note: %s will start in headless mode for each profile.\n", bt)

	type result struct {
		index   int
		profile ChromeProfile
		cookies map[string]string
		err     error
	}

	results := make(chan result, len(selectedProfiles))
	for idx, profile := range selectedProfiles {
		go func(i int, p ChromeProfile) {
			var cookies map[string]string
			var err error
			for retry := 0; retry < 3; retry++ {
				cookies, err = FetchCookiesFromProfileWithBrowser(p, bt)
				if err == nil {
					break
				}
				if retry < 2 {
					time.Sleep(time.Duration(retry+1) * time.Second)
				}
			}
			results <- result{index: i, profile: p, cookies: cookies, err: err}
		}(idx, profile)
	}

	allResults := make([]result, len(selectedProfiles))
	for i := 0; i < len(selectedProfiles); i++ {
		res := <-results
		allResults[res.index] = res
	}

	allCookies := make(map[string]string)
	successCount := 0
	for _, res := range allResults {
		fmt.Printf("Processing %s... ", res.profile.DisplayName)
		if res.err != nil {
			fmt.Printf("FAILED: %v\n", res.err)
			continue
		}
		suffix := ""
		if res.profile.Name != "Default" {
			suffix = "_" + strings.ReplaceAll(res.profile.DisplayName, " ", "_")
		}
		allCookies["__Secure-1PSID"+suffix] = res.cookies["__Secure-1PSID"]
		allCookies["__Secure-1PSIDTS"+suffix] = res.cookies["__Secure-1PSIDTS"]
		successCount++
		fmt.Println("OK")
	}

	if len(allCookies) == 0 {
		return fmt.Errorf("no cookies fetched")
	}

	var orderedKeys []string
	orderedCookies := make(map[string]string)
	for _, res := range allResults {
		if res.err != nil {
			continue
		}
		suffix := ""
		if res.profile.Name != "Default" {
			suffix = "_" + strings.ReplaceAll(res.profile.DisplayName, " ", "_")
		}
		psidKey := "__Secure-1PSID" + suffix
		psidtsKey := "__Secure-1PSIDTS" + suffix

		orderedKeys = append(orderedKeys, psidKey, psidtsKey)
		orderedCookies[psidKey] = allCookies[psidKey]
		orderedCookies[psidtsKey] = allCookies[psidtsKey]
	}

	saveToEnvWithOrder(orderedKeys, orderedCookies)
	fmt.Printf("\nDone! Saved %d/%d cookie pairs to .env\n", successCount, len(selectedProfiles))
	return nil
}
