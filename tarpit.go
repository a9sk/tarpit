package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
)

//go:embed data/words.txt
var wordsRaw string

//go:embed data/stopwords.txt
var stopwordsRaw string

const configPath = "config/config.json"

var (
	wordList  []string
	stopList  []string
	stopReady bool
	logger    = slog.New(slog.NewTextHandler(os.Stderr, nil))
)

func init() {
	wordList = splitLines(wordsRaw)
}

// Config holds the application configuration, mirroring the Python config.json structure.
type Config struct {
	Port                int          `json:"port"`
	RandomSeed          int64        `json:"random-seed"`
	DocumentRoot        string       `json:"document-root"`
	FakeImageDir        ListOrString `json:"fake-image-dir"`
	FakeCSSDir          ListOrString `json:"fake-css-dir"`
	SpacingChars        []string     `json:"spacing-characters"`
	UnsafeChars         []string     `json:"unsafe-characters"`
	RobotsTXT           string       `json:"robots-txt"`
	HTMLTemplates       ListOrString `json:"html-templates"`
	CSSFiles            ListOrString `json:"css-files"`
	Images              ImageConfig  `json:"images"`
	RemoveFromStopWords []string     `json:"remove-from-stop-words"`

	docRoot *url.URL
}

// ImageConfig holds image file paths per extension.
type ImageConfig struct {
	ICO ListOrString `json:"ico"`
	JPG ListOrString `json:"jpg"`
	PNG ListOrString `json:"png"`
}

// ListOrString is a config value that may be a single string, null, or a list of strings.
type ListOrString []string

// UnmarshalJSON handles string, list, or null values.
func (l *ListOrString) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*l = nil
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*l = []string{s}
		return nil
	}
	var list []string
	if err := json.Unmarshal(data, &list); err != nil {
		return err
	}
	*l = list
	return nil
}

// NameByExt returns the config value for the given extension (ico, jpg, png, css).
func (c *Config) NameByExt(ext string) ListOrString {
	switch ext {
	case "ico":
		return c.Images.ICO
	case "jpg":
		return c.Images.JPG
	case "png":
		return c.Images.PNG
	case "css":
		return c.CSSFiles
	}
	return nil
}

// DocRoot returns the parsed document root URL.
func (c *Config) DocRoot() *url.URL {
	return c.docRoot
}

// LoadConfig loads configuration from the given JSON file path.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	u, err := url.Parse(cfg.DocumentRoot)
	if err != nil {
		return nil, fmt.Errorf("parsing document-root: %w", err)
	}
	cfg.docRoot = u

	return &cfg, nil
}

// Generator produces deterministic random text and URLs based on a page path and config seed.
type Generator struct {
	rnd         *rand.Rand
	stopList    []string
	docRoot     *url.URL
	spacings    []string
	unsafeChars []string
	url         *url.URL
}

// NewGenerator creates a Generator seeded from the URL path and global config seed.
func NewGenerator(pageURL *url.URL, cfg *Config) *Generator {
	seed := hashString(pageURL.Path) + cfg.RandomSeed
	return &Generator{
		rnd:         rand.New(rand.NewSource(seed)),
		stopList:    getStopWords(cfg),
		docRoot:     cfg.DocRoot(),
		spacings:    cfg.SpacingChars,
		unsafeChars: cfg.UnsafeChars,
		url:         pageURL,
	}
}

func getStopWords(cfg *Config) []string {
	if stopReady {
		return stopList
	}
	stopList = splitLines(stopwordsRaw)
	remove := make(map[string]struct{}, len(cfg.RemoveFromStopWords))
	for _, w := range cfg.RemoveFromStopWords {
		remove[w] = struct{}{}
	}
	filtered := make([]string, 0, len(stopList))
	for _, w := range stopList {
		if _, ok := remove[w]; !ok {
			filtered = append(filtered, w)
		}
	}
	stopList = filtered
	stopReady = true
	logger.Info("prepared stop words list", "count", len(stopList))
	return stopList
}

func hashString(s string) int64 {
	var h int64
	for i := 0; i < len(s); i++ {
		h = h*31 + int64(s[i])
	}
	return h
}

func splitLines(s string) []string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func (g *Generator) getBool() bool {
	return g.rnd.Intn(2) == 0
}

func (g *Generator) getWord() string {
	if g.getBool() {
		return g.rndString(wordList)
	}
	return g.rndString(g.stopList)
}

func (g *Generator) rndString(list []string) string {
	return list[g.rnd.Intn(len(list))]
}

func (g *Generator) getSentence() string {
	n := g.rnd.Intn(18) + 3 // 3..20
	words := make([]string, n)
	for i := range words {
		words[i] = g.getWord()
	}
	return capitalizeFirst(strings.Join(words, " ")) + "."
}

func (g *Generator) getTitle() string {
	n := g.rnd.Intn(7) + 1 // 1..7
	words := make([]string, n)
	for i := range words {
		words[i] = g.getWord()
	}
	return strings.Join(words, " ")
}

func (g *Generator) getPara() string {
	n := g.rnd.Intn(6) + 15 // 15..20
	sentences := make([]string, n)
	for i := range sentences {
		sentences[i] = g.getSentence()
	}
	return strings.Join(sentences, " ")
}

func (g *Generator) getHeader() string {
	n := g.rnd.Intn(3) + 1 // 1..3
	words := make([]string, n)
	for i := range words {
		words[i] = capitalizeFirst(g.getWord())
	}
	return strings.Join(words, " ")
}

func (g *Generator) getName() string {
	return toTitle(g.getWord() + " " + g.getWord())
}

func (g *Generator) addLinksToText(txt string) string {
	words := strings.Split(txt, " ")
	n := g.rnd.Intn(4) // 0..3
	for i := 0; i < n; i++ {
		start := g.rnd.Intn(len(words))
		end := start + g.rnd.Intn(5)
		if end > len(words)-1 {
			end = len(words) - 1
		}
		if start == end {
			continue
		}
		words[start] = `<a href="` + g.GetLink() + `">` + words[start]
		words[end] = words[end] + "</a>"
	}
	return strings.Join(words, " ")
}

// GetMainHTML generates the main content block.
func (g *Generator) GetMainHTML() string {
	level := 1
	var sb strings.Builder
	n := g.rnd.Intn(23) + 3 // 3..25
	for i := 0; i < n; i++ {
		if g.getBool() {
			if g.getBool() {
				level++
			} else {
				level--
			}
			level = ((level%4)+4)%4 + 1
		}
		sb.WriteString("<h")
		sb.WriteString(itoa(level))
		sb.WriteString(">")
		sb.WriteString(g.getHeader())
		sb.WriteString("</h")
		sb.WriteString(itoa(level))
		sb.WriteString(">\n<p>")
		sb.WriteString(g.addLinksToText(g.getPara()))
		sb.WriteString("</p>\n")
	}
	return sb.String()
}

func (g *Generator) escapePageName(title string) string {
	spacer := g.spacings[g.rnd.Intn(len(g.spacings))]
	return strings.ToLower(g.removeUnsafeChars(strings.ReplaceAll(title, " ", spacer)))
}

// UnescapePageName turns the URL path back into a title.
func (g *Generator) UnescapePageName() string {
	if g.isDocRoot() {
		return "Home"
	}
	file := path.Base(g.url.Path)
	return toTitle(g.reintroduceSpacing(file))
}

func (g *Generator) reintroduceSpacing(urlPage string) string {
	title := urlPage
	for _, s := range g.spacings {
		title = strings.ReplaceAll(title, s, " ")
	}
	return title
}

func (g *Generator) removeUnsafeChars(txt string) string {
	for _, c := range g.unsafeChars {
		txt = strings.ReplaceAll(txt, c, "")
	}
	return txt
}

// GetParentPageName returns the unescaped name of the parent directory.
func (g *Generator) GetParentPageName() string {
	dirs := path.Dir(g.url.Path)
	parentFile := path.Base(dirs)
	return toTitle(g.reintroduceSpacing(parentFile))
}

func (g *Generator) getPage() string {
	return g.escapePageName(g.removeUnsafeChars(g.getTitle()))
}

func (g *Generator) getPath() string {
	return g.genURL(g.docRoot.Path)
}

func (g *Generator) genURL(parent string) string {
	newURL := parent
	n := g.rnd.Intn(4) + 1 // 1..4
	for i := 0; i < n; i++ {
		newURL = path.Join(newURL, g.getWord())
	}
	return g.removeUnsafeChars(newURL)
}

// GetLink generates a random link.
func (g *Generator) GetLink() string {
	return path.Join(g.getPath(), g.getPage())
}

// GetSubpath generates a random subpath under an optional directory.
func (g *Generator) GetSubpath(newDir *string) string {
	if newDir == nil {
		return g.genURL(g.docRoot.Path)
	}
	return g.genURL(path.Join(g.docRoot.Path, *newDir))
}

// GetLinkForTitle generates a link for a given title.
func (g *Generator) GetLinkForTitle(title string) string {
	return path.Join(g.getPath(), g.escapePageName(title))
}

func (g *Generator) isDocRoot() bool {
	return g.url.Path == g.docRoot.Path
}

// GetSiblingLink generates a link to a sibling page.
func (g *Generator) GetSiblingLink() string {
	parent := g.GetParentLink()
	return path.Join(parent, g.getPage())
}

// GetSiblingForTitle generates a sibling link for a title.
func (g *Generator) GetSiblingForTitle(title string) string {
	return path.Join(g.GetParentLink(), g.escapePageName(title))
}

// GetParentLink returns the parent URL path.
func (g *Generator) GetParentLink() string {
	if g.isDocRoot() {
		return g.url.Path
	}
	return path.Dir(g.url.Path)
}

// ChooseItem picks a random item from a list or returns a single string value.
func (g *Generator) ChooseItem(items ListOrString) *string {
	if items == nil || len(items) == 0 {
		return nil
	}
	if len(items) == 1 {
		s := items[0]
		return &s
	}
	choice := items[g.rnd.Intn(len(items))]
	return &choice
}

func capitalizeFirst(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	if len(r) > 0 && r[0] >= 'a' && r[0] <= 'z' {
		r[0] = r[0] - 'a' + 'A'
	}
	return string(r)
}

func toTitle(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	inWord := false
	for i, c := range r {
		if c >= 'a' && c <= 'z' {
			if !inWord {
				r[i] = c - 'a' + 'A'
				inWord = true
			}
		} else if c >= 'A' && c <= 'Z' {
			if inWord {
				r[i] = c - 'A' + 'a'
			}
			inWord = true
		} else {
			inWord = false
		}
	}
	return string(r)
}

func itoa(i int) string {
	switch i {
	case 1:
		return "1"
	case 2:
		return "2"
	case 3:
		return "3"
	case 4:
		return "4"
	case 5:
		return "5"
	case 6:
		return "6"
	}
	var buf [20]byte
	return string(intToBytes(buf[:], i))
}

func intToBytes(buf []byte, i int) []byte {
	if i == 0 {
		return []byte("0")
	}
	negative := i < 0
	if negative {
		i = -i
	}
	n := len(buf)
	for i > 0 {
		n--
		buf[n] = byte('0' + i%10)
		i /= 10
	}
	if negative {
		n--
		buf[n] = '-'
	}
	return buf[n:]
}

// chooseItem picks a deterministic item from a list using the URL-seeded generator.
func chooseItem(pageURL *url.URL, items ListOrString, cfg *Config) *string {
	if items == nil {
		return nil
	}
	g := NewGenerator(pageURL, cfg)
	return g.ChooseItem(items)
}

// getTemplate loads the chosen HTML template file.
func getTemplate(pageURL *url.URL, cfg *Config) (string, error) {
	templateFile := chooseItem(pageURL, cfg.HTMLTemplates, cfg)
	if templateFile == nil {
		return "", fmt.Errorf("no html templates have been defined in the configuration")
	}
	data, err := os.ReadFile(*templateFile)
	if err != nil {
		return "", fmt.Errorf("reading template: %w", err)
	}
	return string(data), nil
}

// getHTML generates the final HTML for a URL.
func getHTML(pageURL *url.URL, cfg *Config) (string, error) {
	textSource := NewGenerator(pageURL, cfg)
	res, err := getTemplate(pageURL, cfg)
	if err != nil {
		return "", err
	}

	title := textSource.UnescapePageName()

	res = strings.ReplaceAll(res, "{HOME}", cfg.DocRoot().Path)
	res = strings.ReplaceAll(res, "{TITLE}", title)
	res = strings.ReplaceAll(res, "{UPTITLE}", textSource.GetParentPageName())
	res = strings.ReplaceAll(res, "{MAIN}", textSource.GetMainHTML())
	res = strings.ReplaceAll(res, "{UP}", textSource.GetParentLink())

	cssDir := textSource.ChooseItem(cfg.FakeCSSDir)
	res = strings.ReplaceAll(res, "{CSSLINK}", textSource.GetSubpath(cssDir))

	for strings.Contains(res, "{NEWTITLE}") {
		titlePos := strings.Index(res, "{NEWTITLE}")
		titleText := textSource.getTitle()

		segment := res
		linkPos := strings.LastIndex(segment[:titlePos], "{LINK}")
		overPos := strings.LastIndex(segment[:titlePos], "{OVER}")

		if linkPos != -1 || overPos != -1 {
			if linkPos > overPos {
				link := textSource.GetLinkForTitle(titleText)
				segment = strings.Replace(segment[linkPos:titlePos], "{LINK}", link, 1)
				tagPos := linkPos
				res = res[:tagPos] + segment + res[titlePos:]
			} else {
				overLink := textSource.GetSiblingForTitle(titleText)
				segment = strings.Replace(segment[overPos:titlePos], "{OVER}", overLink, 1)
				tagPos := overPos
				res = res[:tagPos] + segment + res[titlePos:]
			}
		}
		res = strings.Replace(res, "{NEWTITLE}", titleText, 1)
	}

	for {
		replaced := false
		for _, tag := range []string{"{WORD}", "{NEWTITLE}", "{PIC}", "{SENTENCE}", "{LINK}", "{OVER}", "{NAME}"} {
			if !strings.Contains(res, tag) {
				continue
			}
			replaced = true
			switch tag {
			case "{WORD}":
				res = strings.Replace(res, tag, textSource.getWord(), 1)
			case "{NEWTITLE}":
				res = strings.Replace(res, tag, textSource.getTitle(), 1)
			case "{PIC}":
				imgDir := textSource.ChooseItem(cfg.FakeImageDir)
				res = strings.Replace(res, tag, textSource.GetSubpath(imgDir), 1)
			case "{SENTENCE}":
				res = strings.Replace(res, tag, textSource.getSentence(), 1)
			case "{LINK}":
				res = strings.Replace(res, tag, textSource.GetLink(), 1)
			case "{OVER}":
				res = strings.Replace(res, tag, textSource.GetSiblingLink(), 1)
			case "{NAME}":
				res = strings.Replace(res, tag, textSource.getName(), 1)
			}
		}
		if !replaced {
			break
		}
	}

	return res, nil
}

// Handler is the HTTP request handler.
type Handler struct {
	cfg *Config
}

// NewHandler creates a new Handler.
func NewHandler(cfg *Config) *Handler {
	return &Handler{cfg: cfg}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleGET(w, r)
	default:
		h.serve404(w)
	}
}

func (h *Handler) serve404(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNotFound)
}

func (h *Handler) handleGET(w http.ResponseWriter, r *http.Request) {
	pageURL, err := url.Parse(r.URL.String())
	if err != nil {
		logger.Error("failed to parse request URL", "url", r.URL.String(), "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	docRoot := h.cfg.DocRoot()
	if !strings.HasPrefix(pageURL.Path, docRoot.Path) && !strings.HasPrefix(r.URL.String(), docRoot.Path) {
		logger.Warn("invalid request, not in document root", "path", pageURL.Path)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if pageURL.Path == docRoot.Path {
		h.logCatch(r)
	}

	urlPath := pageURL.Path
	if len(urlPath) > 4 && urlPath[len(urlPath)-4] == '.' {
		base := path.Base(urlPath)
		if urlPath == path.Join(docRoot.Path, "robots.txt") {
			h.serveRobots(w)
			return
		}
		h.serveFile(w, r, base)
		return
	}

	h.serveHTML(w, pageURL)
}

func (h *Handler) logCatch(r *http.Request) {
	logger.Info("caught request in document root",
		"user_agent", r.UserAgent(),
		"remote_addr", r.RemoteAddr,
		"path", r.URL.Path,
	)
}

func (h *Handler) serveFile(w http.ResponseWriter, r *http.Request, base string) {
	var contentType, ext string
	switch {
	case strings.HasSuffix(base, ".ico"):
		contentType = "image/vnd.microsoft.icon"
		ext = "ico"
	case strings.HasSuffix(base, ".jpg"):
		contentType = "image/jpeg"
		ext = "jpg"
	case strings.HasSuffix(base, ".png"):
		contentType = "image/png"
		ext = "png"
	case strings.HasSuffix(base, ".css"):
		contentType = "text/css"
		ext = "css"
	default:
		logger.Warn("served 404 for unknown extension", "path", r.URL.Path)
		h.serve404(w)
		return
	}

	fileOptions := h.cfg.NameByExt(ext)
	chosen := chooseItem(r.URL, fileOptions, h.cfg)
	if chosen == nil {
		logger.Warn("client requested file but none configured", "extension", ext)
		h.serve404(w)
		return
	}

	data, err := os.ReadFile(*chosen)
	if err != nil {
		logger.Error("failed to read file", "file", *chosen, "error", err)
		h.serve404(w)
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusOK)
	w.Write(data)
	logger.Info("served file", "file", *chosen, "bytes", len(data))
}

func (h *Handler) serveRobots(w http.ResponseWriter) {
	if h.cfg.RobotsTXT == "" {
		h.serve404(w)
		return
	}
	data, err := os.ReadFile(h.cfg.RobotsTXT)
	if err != nil {
		logger.Error("failed to read robots.txt", "error", err)
		h.serve404(w)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
	logger.Info("served robots.txt", "bytes", len(data))
}

func (h *Handler) serveHTML(w http.ResponseWriter, pageURL *url.URL) {
	page, err := getHTML(pageURL, h.cfg)
	if err != nil {
		logger.Error("failed to generate HTML", "path", pageURL.Path, "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(page))
	logger.Info("served html", "path", pageURL.Path, "chars", len(page))
}

func main() {
	cfg, err := LoadConfig(configPath)
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	if cfg.RandomSeed == 0 {
		logger.Warn("random seed is set to 0; please update it in the config file")
	}

	handler := NewHandler(cfg)
	port := cfg.Port
	if envPort := os.Getenv("PORT"); envPort != "" {
		var p int
		if _, err := fmt.Sscanf(envPort, "%d", &p); err == nil {
			port = p
		}
	}
	addr := fmt.Sprintf(":%d", port)
	logger.Info("listening", "port", port)
	if err := http.ListenAndServe(addr, handler); err != nil {
		logger.Error("server error", "error", err)
		os.Exit(1)
	}
}
