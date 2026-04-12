package theme

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/buzyka/imlate/internal/config"
	"github.com/buzyka/imlate/internal/infrastructure/imageutil"
)

const maxThemeDurationMs = 60000

type Error struct {
	Message    string
	StatusCode int
}

func (e *Error) Error() string {
	return e.Message
}

type AssetState struct {
	Slot        string     `json:"slot"`
	CurrentURL  string     `json:"current_url"`
	DefaultURL  string     `json:"default_url"`
	IsCustom    bool       `json:"is_custom"`
	ContentType string     `json:"content_type,omitempty"`
	SizeBytes   int64      `json:"size_bytes"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
}

type Settings struct {
	WelcomeDurationMs int `json:"welcome_duration_ms"`
	GoodbyeDurationMs int `json:"goodbye_duration_ms"`
}

type Response struct {
	Assets   map[string]AssetState `json:"assets"`
	Settings Settings              `json:"settings"`
}

type assetManifest struct {
	FileName    string     `json:"file_name,omitempty"`
	ContentType string     `json:"content_type,omitempty"`
	SizeBytes   int64      `json:"size_bytes,omitempty"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
}

type manifest struct {
	Assets   map[string]assetManifest `json:"assets"`
	Settings Settings                 `json:"settings"`
}

type slotSpec struct {
	DefaultURL  string
	AllowedMIME []string
	MaxInput    int64
	MaxOutput   int64
	MaxWidth    int
	MaxHeight   int
}

type Service struct {
	Config *config.Config `container:"type"`

	mu  sync.Mutex
	now func() time.Time
}

func (s *Service) GetTheme() (*Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	m, err := s.loadManifest()
	if err != nil {
		return nil, err
	}
	return s.responseFromManifest(m), nil
}

func (s *Service) UploadAsset(slot, filename string, data []byte) (*Response, error) {
	spec, ok := slotSpecs[slot]
	if !ok {
		return nil, &Error{Message: "invalid theme slot", StatusCode: 400}
	}

	processed, err := imageutil.ProcessUploadedImage(filename, data, imageutil.ProcessOptions{
		AllowedMIMEs:   spec.AllowedMIME,
		MaxInputBytes:  spec.MaxInput,
		MaxOutputBytes: spec.MaxOutput,
		MaxWidth:       spec.MaxWidth,
		MaxHeight:      spec.MaxHeight,
	})
	if err != nil {
		return nil, &Error{Message: err.Error(), StatusCode: 400}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	m, err := s.loadManifest()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(s.Config.ThemeDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create theme directory: %w", err)
	}

	current := m.Assets[slot]
	now := s.clock()
	fileName := fmt.Sprintf("%s-%d%s", slot, now.UnixNano(), processed.Extension)
	filePath := filepath.Join(s.Config.ThemeDir, fileName)
	if err := os.WriteFile(filePath, processed.Data, 0o644); err != nil {
		return nil, fmt.Errorf("failed to write theme asset: %w", err)
	}

	m.Assets[slot] = assetManifest{
		FileName:    fileName,
		ContentType: processed.ContentType,
		SizeBytes:   int64(len(processed.Data)),
		UpdatedAt:   ptrTime(now),
	}
	if err := s.writeManifest(m); err != nil {
		_ = os.Remove(filePath)
		return nil, err
	}
	if current.FileName != "" && current.FileName != fileName {
		s.removeThemeAssetFile(current.FileName)
	}

	return s.responseFromManifest(m), nil
}

func (s *Service) ResetAsset(slot string) (*Response, error) {
	if _, ok := slotSpecs[slot]; !ok {
		return nil, &Error{Message: "invalid theme slot", StatusCode: 400}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	m, err := s.loadManifest()
	if err != nil {
		return nil, err
	}

	current := m.Assets[slot]
	delete(m.Assets, slot)
	if err := s.writeManifest(m); err != nil {
		return nil, err
	}
	if current.FileName != "" {
		s.removeThemeAssetFile(current.FileName)
	}

	return s.responseFromManifest(m), nil
}

func (s *Service) UpdateSettings(settings Settings) (*Response, error) {
	if err := validateSettings(settings); err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	m, err := s.loadManifest()
	if err != nil {
		return nil, err
	}
	m.Settings = settings
	if err := s.writeManifest(m); err != nil {
		return nil, err
	}
	return s.responseFromManifest(m), nil
}

func (s *Service) GetReaderPageData() (ReaderPageData, error) {
	resp, err := s.GetTheme()
	if err != nil {
		return ReaderPageData{}, err
	}
	return ReaderPageData{
		FaviconURL:          resp.Assets["favicon"].CurrentURL,
		LogoBackgroundURL:   resp.Assets["logo_background"].CurrentURL,
		WelcomeAnimationURL: resp.Assets["welcome_animation"].CurrentURL,
		GoodbyeAnimationURL: resp.Assets["goodbye_animation"].CurrentURL,
		WelcomeDurationMs:   resp.Settings.WelcomeDurationMs,
		GoodbyeDurationMs:   resp.Settings.GoodbyeDurationMs,
	}, nil
}

func validateSettings(settings Settings) error {
	if settings.WelcomeDurationMs < MinAnimationDurationMs || settings.GoodbyeDurationMs < MinAnimationDurationMs {
		return &Error{Message: fmt.Sprintf("animation duration must be at least %d ms", MinAnimationDurationMs), StatusCode: 400}
	}
	if settings.WelcomeDurationMs > maxThemeDurationMs || settings.GoodbyeDurationMs > maxThemeDurationMs {
		return &Error{Message: fmt.Sprintf("animation duration must be less than or equal to %d ms", maxThemeDurationMs), StatusCode: 400}
	}
	return nil
}

func (s *Service) loadManifest() (*manifest, error) {
	m := &manifest{
		Assets:   map[string]assetManifest{},
		Settings: s.defaultSettings(),
	}

	path := s.manifestPath()
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return m, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read theme manifest: %w", err)
	}
	if err := json.Unmarshal(data, m); err != nil {
		return nil, fmt.Errorf("failed to decode theme manifest: %w", err)
	}
	if m.Assets == nil {
		m.Assets = map[string]assetManifest{}
	}
	if m.Settings.WelcomeDurationMs == 0 {
		m.Settings.WelcomeDurationMs = s.defaultSettings().WelcomeDurationMs
	}
	if m.Settings.GoodbyeDurationMs == 0 {
		m.Settings.GoodbyeDurationMs = s.defaultSettings().GoodbyeDurationMs
	}
	return m, nil
}

func (s *Service) writeManifest(m *manifest) error {
	if err := os.MkdirAll(s.Config.ThemeDir, 0o755); err != nil {
		return fmt.Errorf("failed to create theme directory: %w", err)
	}

	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode theme manifest: %w", err)
	}

	tmpPath := s.manifestPath() + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write theme manifest: %w", err)
	}
	if err := os.Rename(tmpPath, s.manifestPath()); err != nil {
		return fmt.Errorf("failed to replace theme manifest: %w", err)
	}
	return nil
}

func (s *Service) responseFromManifest(m *manifest) *Response {
	assets := map[string]AssetState{}
	for slot, spec := range slotSpecs {
		item, ok := m.Assets[slot]
		currentURL := spec.DefaultURL
		isCustom := false
		if ok && isPlainThemeAssetFileName(item.FileName) {
			currentURL = joinURLPath(s.Config.ThemeURLPrefix, item.FileName)
			isCustom = true
		}
		assets[slot] = AssetState{
			Slot:        slot,
			CurrentURL:  currentURL,
			DefaultURL:  spec.DefaultURL,
			IsCustom:    isCustom,
			ContentType: item.ContentType,
			SizeBytes:   item.SizeBytes,
			UpdatedAt:   item.UpdatedAt,
		}
	}

	return &Response{
		Assets:   assets,
		Settings: m.Settings,
	}
}

func (s *Service) manifestPath() string {
	return filepath.Join(s.Config.ThemeDir, "theme.json")
}

func (s *Service) defaultSettings() Settings {
	return Settings{
		WelcomeDurationMs: DefaultWelcomeDurationMs,
		GoodbyeDurationMs: DefaultGoodbyeDurationMs,
	}
}

func (s *Service) clock() time.Time {
	if s.now != nil {
		return s.now()
	}
	return time.Now().UTC()
}

func joinURLPath(prefix, fileName string) string {
	return fmt.Sprintf("%s/%s", strings.TrimRight(prefix, "/"), fileName)
}

func ptrTime(t time.Time) *time.Time {
	return &t
}

func (s *Service) removeThemeAssetFile(fileName string) {
	filePath, err := s.themeAssetPath(fileName)
	if err != nil {
		return
	}
	_ = os.Remove(filePath)
}

func (s *Service) themeAssetPath(fileName string) (string, error) {
	if !isPlainThemeAssetFileName(fileName) {
		return "", fmt.Errorf("invalid theme asset file name %q", fileName)
	}

	themeDir := filepath.Clean(s.Config.ThemeDir)
	filePath := filepath.Join(themeDir, fileName)
	relPath, err := filepath.Rel(themeDir, filePath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve theme asset path: %w", err)
	}
	if relPath == ".." || strings.HasPrefix(relPath, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("theme asset path %q escapes theme directory", fileName)
	}
	return filePath, nil
}

func isPlainThemeAssetFileName(fileName string) bool {
	if fileName == "" || fileName == "." || fileName == ".." {
		return false
	}
	return fileName == filepath.Base(fileName) && !strings.ContainsAny(fileName, `/\`)
}

var slotSpecs = map[string]slotSpec{
	"favicon": {
		DefaultURL:  DefaultFaviconURL,
		AllowedMIME: []string{"image/png"},
		MaxInput:    5 * 1024 * 1024,
		MaxOutput:   512 * 1024,
		MaxWidth:    256,
		MaxHeight:   256,
	},
	"logo_background": {
		DefaultURL:  DefaultLogoBackgroundURL,
		AllowedMIME: []string{"image/jpeg", "image/png"},
		MaxInput:    20 * 1024 * 1024,
		MaxOutput:   2 * 1024 * 1024,
		MaxWidth:    1920,
		MaxHeight:   1080,
	},
	"welcome_animation": {
		DefaultURL:  DefaultWelcomeAnimationURL,
		AllowedMIME: []string{"image/gif"},
		MaxInput:    15 * 1024 * 1024,
		MaxOutput:   15 * 1024 * 1024,
	},
	"goodbye_animation": {
		DefaultURL:  DefaultGoodbyeAnimationURL,
		AllowedMIME: []string{"image/gif"},
		MaxInput:    15 * 1024 * 1024,
		MaxOutput:   15 * 1024 * 1024,
	},
}
