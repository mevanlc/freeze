package main

import (
	"bytes"
	"errors"
	"fmt"
	imagepng "image/png"
	"math"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/beevik/etree"
)

const chromiumRenderTimeout = time.Minute

func chromiumConvert(doc *etree.Document, width, height float64, output string) error {
	browser, err := findChromium()
	if err != nil {
		return err
	}
	return chromiumConvertWithExecutable(browser, doc, width, height, output)
}

func chromiumConvertWithExecutable(browser string, doc *etree.Document, width, height float64, output string) error {
	tempDir, err := os.MkdirTemp("", "freeze-chromium-*")
	if err != nil {
		return fmt.Errorf("create temporary directory: %w", err)
	}
	defer os.RemoveAll(tempDir) //nolint:errcheck

	svgPath := filepath.Join(tempDir, "freeze.svg")
	if err := doc.WriteToFile(svgPath); err != nil {
		return fmt.Errorf("write temporary SVG: %w", err)
	}
	outputPath, err := filepath.Abs(output)
	if err != nil {
		return fmt.Errorf("resolve output path: %w", err)
	}
	screenshotPath := filepath.Join(tempDir, "screenshot.png")

	urlPath := filepath.ToSlash(svgPath)
	if runtime.GOOS == "windows" && !strings.HasPrefix(urlPath, "/") {
		urlPath = "/" + urlPath
	}
	fileURL := (&url.URL{Scheme: "file", Path: urlPath}).String()
	windowSize := strconv.Itoa(int(math.Ceil(width))) + "," + strconv.Itoa(int(math.Ceil(height)))
	cmd := exec.Command(browser, //nolint:gosec
		"--headless=new",
		"--disable-gpu",
		"--hide-scrollbars",
		"--force-device-scale-factor=1",
		"--default-background-color=00000000",
		"--user-data-dir="+filepath.Join(tempDir, "profile"),
		"--window-size="+windowSize,
		"--screenshot="+screenshotPath,
		fileURL,
	)
	configureChromiumProcess(cmd)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start %s: %w", filepath.Base(browser), err)
	}
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	timer := time.NewTimer(chromiumRenderTimeout)
	defer timer.Stop()

	for {
		select {
		case runErr := <-done:
			if !isCompletePNG(screenshotPath) {
				return chromiumRunError(browser, runErr, stderr.String())
			}
			return copyChromiumScreenshot(screenshotPath, outputPath)
		case <-ticker.C:
			if !isCompletePNG(screenshotPath) {
				continue
			}
			stopChromiumProcess(cmd, done)
			return copyChromiumScreenshot(screenshotPath, outputPath)
		case <-timer.C:
			if isCompletePNG(screenshotPath) {
				stopChromiumProcess(cmd, done)
				return copyChromiumScreenshot(screenshotPath, outputPath)
			}
			stopChromiumProcess(cmd, done)
			return fmt.Errorf("run %s: timed out after %s", filepath.Base(browser), chromiumRenderTimeout)
		}
	}
}

func isCompletePNG(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close() //nolint:errcheck
	_, err = imagepng.Decode(f)
	return err == nil
}

func copyChromiumScreenshot(source, output string) error {
	data, err := os.ReadFile(source)
	if err != nil {
		return fmt.Errorf("read Chromium screenshot: %w", err)
	}
	if err := os.WriteFile(output, data, 0o600); err != nil { //nolint:gosec // The CLI output path is intentionally user-selected.
		return fmt.Errorf("write Chromium screenshot: %w", err)
	}
	return nil
}

func chromiumRunError(browser string, runErr error, stderr string) error {
	if runErr == nil {
		runErr = errors.New("browser exited without writing a PNG")
	}
	if message := strings.TrimSpace(stderr); message != "" {
		return fmt.Errorf("run %s: %w: %s", filepath.Base(browser), runErr, message)
	}
	return fmt.Errorf("run %s: %w", filepath.Base(browser), runErr)
}

func stopChromiumProcess(cmd *exec.Cmd, done <-chan error) {
	_ = signalChromiumProcess(cmd, false)
	select {
	case <-done:
		return
	case <-time.After(2 * time.Second):
		_ = signalChromiumProcess(cmd, true)
		<-done
	}
}

func findChromium() (string, error) {
	for _, candidate := range chromiumCandidates() {
		path, err := exec.LookPath(candidate)
		if err == nil {
			return path, nil
		}
	}
	return "", errors.New("no supported Chromium browser found (tried Google Chrome, Google Chrome Beta, Microsoft Edge, and Chromium)")
}

func chromiumCandidates() []string {
	switch runtime.GOOS {
	case "darwin":
		var candidates []string
		home, homeErr := os.UserHomeDir()
		addApplication := func(name, executable string) {
			relativePath := filepath.Join(name+".app", "Contents", "MacOS", executable)
			candidates = append(candidates, filepath.Join("/Applications", relativePath))
			if homeErr == nil {
				candidates = append(candidates, filepath.Join(home, "Applications", relativePath))
			}
		}
		addApplication("Google Chrome", "Google Chrome")
		addApplication("Google Chrome Beta", "Google Chrome Beta")
		addApplication("Microsoft Edge", "Microsoft Edge")
		addApplication("Chromium", "Chromium")
		return append(candidates,
			"google-chrome",
			"google-chrome-beta",
			"microsoft-edge",
			"chromium",
		)
	case "windows":
		var candidates []string
		addInstallation := func(root, relativePath string) {
			if root != "" {
				candidates = append(candidates, filepath.Join(root, relativePath))
			}
		}
		localAppData := os.Getenv("LOCALAPPDATA")
		programFiles := os.Getenv("ProgramFiles")
		programFilesX86 := os.Getenv("ProgramFiles(x86)")
		addInstallation(localAppData, `Google\Chrome\Application\chrome.exe`)
		addInstallation(programFiles, `Google\Chrome\Application\chrome.exe`)
		addInstallation(programFilesX86, `Google\Chrome\Application\chrome.exe`)
		addInstallation(localAppData, `Google\Chrome Beta\Application\chrome.exe`)
		addInstallation(programFiles, `Google\Chrome Beta\Application\chrome.exe`)
		addInstallation(programFilesX86, `Google\Chrome Beta\Application\chrome.exe`)
		addInstallation(programFiles, `Microsoft\Edge\Application\msedge.exe`)
		addInstallation(programFilesX86, `Microsoft\Edge\Application\msedge.exe`)
		addInstallation(localAppData, `Chromium\Application\chrome.exe`)
		return append(candidates, "chrome.exe", "msedge.exe", "chromium.exe")
	default:
		return []string{
			"google-chrome",
			"google-chrome-stable",
			"google-chrome-beta",
			"microsoft-edge",
			"microsoft-edge-stable",
			"chromium",
			"chromium-browser",
		}
	}
}
