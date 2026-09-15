package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
)

func mustParseURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("failed to parse URL %q: %v", raw, err)
	}
	return u
}

func testConfig(t *testing.T) *Config {
	t.Helper()
	cfg, err := LoadConfig("config/config.json")
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	return cfg
}

func TestLoadConfig(t *testing.T) {
	cfg := testConfig(t)
	if cfg.Port != 80 {
		t.Errorf("expected port 80, got %d", cfg.Port)
	}
	if cfg.RandomSeed != 12345 {
		t.Errorf("expected random seed 12345, got %d", cfg.RandomSeed)
	}
	if cfg.DocRoot().Path != "/" {
		t.Errorf("expected document root /, got %s", cfg.DocRoot().Path)
	}
}

func TestListOrStringUnmarshal(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantNil bool
		want    []string
	}{
		{"single string", `"foo"`, false, []string{"foo"}},
		{"list", `["a","b"]`, false, []string{"a", "b"}},
		{"null", `null`, true, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var l ListOrString
			if err := json.Unmarshal([]byte(tc.input), &l); err != nil {
				t.Fatalf("unmarshal failed: %v", err)
			}
			if tc.wantNil && l != nil {
				t.Fatalf("expected nil, got %v", l)
			}
			if !tc.wantNil {
				if len(l) != len(tc.want) {
					t.Fatalf("expected %v, got %v", tc.want, l)
				}
				for i := range l {
					if l[i] != tc.want[i] {
						t.Errorf("index %d: expected %q, got %q", i, tc.want[i], l[i])
					}
				}
			}
		})
	}
}

func TestNameByExt(t *testing.T) {
	cfg := testConfig(t)
	cases := []struct {
		ext  string
		want ListOrString
	}{
		{"ico", cfg.Images.ICO},
		{"jpg", cfg.Images.JPG},
		{"png", cfg.Images.PNG},
		{"css", cfg.CSSFiles},
		{"txt", nil},
	}
	for _, tc := range cases {
		got := cfg.NameByExt(tc.ext)
		if len(got) != len(tc.want) {
			t.Errorf("NameByExt(%q) = %v, want %v", tc.ext, got, tc.want)
		}
	}
}

func TestDeterministicGeneration(t *testing.T) {
	cfg := testConfig(t)
	u := mustParseURL(t, "/blog/about/once-upon-a-time")
	g1 := NewGenerator(u, cfg)
	g2 := NewGenerator(u, cfg)

	if g1.GetLink() != g2.GetLink() {
		t.Errorf("GetLink not deterministic: %q vs %q", g1.GetLink(), g2.GetLink())
	}
	if g1.getSentence() != g2.getSentence() {
		t.Errorf("getSentence not deterministic: %q vs %q", g1.getSentence(), g2.getSentence())
	}
	if g1.GetMainHTML() != g2.GetMainHTML() {
		t.Errorf("GetMainHTML not deterministic")
	}
	if g1.GetSiblingLink() != g2.GetSiblingLink() {
		t.Errorf("GetSiblingLink not deterministic")
	}
}

func TestUnescapePageName(t *testing.T) {
	cfg := testConfig(t)
	cases := []struct {
		path     string
		expected string
	}{
		{"/", "Home"},
		{"/blog/about/once-upon-a-time", "Once Upon A Time"},
		{"/blog/about/once_upon_a_time", "Once Upon A Time"},
		{"/blog/about/once%20upon%20a%20time", "Once Upon A Time"},
	}
	for _, tc := range cases {
		u := mustParseURL(t, tc.path)
		g := NewGenerator(u, cfg)
		got := g.UnescapePageName()
		if got != tc.expected {
			t.Errorf("UnescapePageName(%q) = %q, want %q", tc.path, got, tc.expected)
		}
	}
}

func TestGetParentPageName(t *testing.T) {
	cfg := testConfig(t)
	u := mustParseURL(t, "/blog/about/once-upon-a-time")
	g := NewGenerator(u, cfg)
	got := g.GetParentPageName()
	want := "About"
	if got != want {
		t.Errorf("GetParentPageName() = %q, want %q", got, want)
	}
}

func TestGetParentLink(t *testing.T) {
	cfg := testConfig(t)
	cases := []struct {
		path string
		want string
	}{
		{"/", "/"},
		{"/blog/about/page", "/blog/about"},
	}
	for _, tc := range cases {
		u := mustParseURL(t, tc.path)
		g := NewGenerator(u, cfg)
		if got := g.GetParentLink(); got != tc.want {
			t.Errorf("GetParentLink(%q) = %q, want %q", tc.path, got, tc.want)
		}
	}
}

func TestTitleCase(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"hello world", "Hello World"},
		{"ONCE UPON A TIME", "Once Upon A Time"},
		{"", ""},
		{"a", "A"},
		{"mixed-CASE text", "Mixed-Case Text"},
	}
	for _, tc := range cases {
		got := toTitle(tc.in)
		if got != tc.want {
			t.Errorf("toTitle(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestCapitalizeFirst(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"hello", "Hello"},
		{"Hello", "Hello"},
		{"", ""},
		{"123", "123"},
	}
	for _, tc := range cases {
		got := capitalizeFirst(tc.in)
		if got != tc.want {
			t.Errorf("capitalizeFirst(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestRemoveUnsafeChars(t *testing.T) {
	cfg := testConfig(t)
	g := NewGenerator(mustParseURL(t, "/"), cfg)
	got := g.removeUnsafeChars("it's a `test`")
	want := "its a test"
	if got != want {
		t.Errorf("removeUnsafeChars() = %q, want %q", got, want)
	}
}

func TestChooseItem(t *testing.T) {
	cfg := testConfig(t)
	u := mustParseURL(t, "/")
	g := NewGenerator(u, cfg)

	if g.ChooseItem(nil) != nil {
		t.Error("ChooseItem(nil) should return nil")
	}
	if g.ChooseItem(ListOrString{}) != nil {
		t.Error("ChooseItem(empty) should return nil")
	}
	if got := g.ChooseItem(ListOrString{"only"}); got == nil || *got != "only" {
		t.Errorf("ChooseItem(single) = %v, want only", got)
	}
}

func TestGetHTML(t *testing.T) {
	cfg := testConfig(t)
	u := mustParseURL(t, "/blog/about/test-page")
	page, err := getHTML(u, cfg)
	if err != nil {
		t.Fatalf("getHTML failed: %v", err)
	}
	if !strings.Contains(page, "<html>") {
		t.Error("generated HTML missing <html>")
	}
	if !strings.Contains(page, "Test Page") {
		t.Error("generated HTML missing title")
	}
	if strings.Contains(page, "{TITLE}") {
		t.Error("{TITLE} tag was not substituted")
	}
	if strings.Contains(page, "{MAIN}") {
		t.Error("{MAIN} tag was not substituted")
	}
}

func TestGetHTMLNoTemplate(t *testing.T) {
	cfg := testConfig(t)
	cfg.HTMLTemplates = nil
	u := mustParseURL(t, "/")
	if _, err := getHTML(u, cfg); err == nil {
		t.Error("expected error when no templates configured")
	}
}

func TestHandlerGET(t *testing.T) {
	cfg := testConfig(t)
	h := NewHandler(cfg)
	req := httptest.NewRequest(http.MethodGet, "/blog/test", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	ct := rr.Header().Get("Content-Type")
	if ct != "text/html" {
		t.Errorf("expected text/html, got %q", ct)
	}
	if !strings.Contains(rr.Body.String(), "<html>") {
		t.Error("response missing html")
	}
}

func TestHandlerRobots(t *testing.T) {
	cfg := testConfig(t)
	h := NewHandler(cfg)
	req := httptest.NewRequest(http.MethodGet, "/robots.txt", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	ct := rr.Header().Get("Content-Type")
	if ct != "text/plain" {
		t.Errorf("expected text/plain, got %q", ct)
	}
	if !strings.Contains(rr.Body.String(), "Disallow") {
		t.Error("robots.txt missing Disallow")
	}
}

func TestHandlerCSS(t *testing.T) {
	cfg := testConfig(t)
	h := NewHandler(cfg)
	req := httptest.NewRequest(http.MethodGet, "/css/style.css", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	ct := rr.Header().Get("Content-Type")
	if ct != "text/css" {
		t.Errorf("expected text/css, got %q", ct)
	}
	if !strings.Contains(rr.Body.String(), "body {") {
		t.Error("css missing body rule")
	}
}

func TestHandlerImage(t *testing.T) {
	cfg := testConfig(t)
	h := NewHandler(cfg)
	req := httptest.NewRequest(http.MethodGet, "/images/logo.png", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	ct := rr.Header().Get("Content-Type")
	if ct != "image/png" {
		t.Errorf("expected image/png, got %q", ct)
	}
	if rr.Body.Len() == 0 {
		t.Error("image body empty")
	}
}

func TestHandler404(t *testing.T) {
	cfg := testConfig(t)
	h := NewHandler(cfg)
	cases := []struct {
		name   string
		method string
		path   string
	}{
		{"post root", http.MethodPost, "/"},
		{"put root", http.MethodPut, "/"},
		{"delete root", http.MethodDelete, "/"},
		{"unknown extension", http.MethodGet, "/file.exe"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)
			if rr.Code != http.StatusNotFound {
				t.Errorf("expected 404, got %d", rr.Code)
			}
		})
	}
}

func TestHandlerDocumentRootValidation(t *testing.T) {
	cfg := testConfig(t)
	cfg.DocumentRoot = "/tarpit"
	u, err := url.Parse(cfg.DocumentRoot)
	if err != nil {
		t.Fatalf("failed to parse document root: %v", err)
	}
	cfg.docRoot = u

	h := NewHandler(cfg)
	req := httptest.NewRequest(http.MethodGet, "/other", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 for path outside document root, got %d", rr.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/tarpit/page", nil)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for path inside document root, got %d", rr.Code)
	}
}

func TestGetMainHTMLStructure(t *testing.T) {
	cfg := testConfig(t)
	u := mustParseURL(t, "/blog/post")
	g := NewGenerator(u, cfg)
	html := g.GetMainHTML()
	if !strings.Contains(html, "<p>") {
		t.Error("main HTML missing paragraph")
	}
	if !strings.Contains(html, "</p>") {
		t.Error("main HTML missing closing paragraph")
	}
	openH := strings.Count(html, "<h")
	closeH := strings.Count(html, "</h")
	if openH == 0 || openH != closeH {
		t.Errorf("unbalanced heading tags: open=%d close=%d", openH, closeH)
	}
}

func TestItoa(t *testing.T) {
	cases := []struct {
		in   int
		want string
	}{
		{0, "0"},
		{1, "1"},
		{4, "4"},
		{42, "42"},
		{-5, "-5"},
	}
	for _, tc := range cases {
		got := itoa(tc.in)
		if got != tc.want {
			t.Errorf("itoa(%d) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestHashString(t *testing.T) {
	// Hash should be deterministic.
	if hashString("abc") != hashString("abc") {
		t.Error("hashString not deterministic")
	}
	if hashString("abc") == hashString("abd") {
		t.Error("hashString collision")
	}
}

func TestSplitLines(t *testing.T) {
	got := splitLines("a\nb\n\n c \n")
	want := []string{"a", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("splitLines = %v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("index %d: got %q, want %q", i, got[i], want[i])
		}
	}
}

func TestBinaryImageUnchanged(t *testing.T) {
	cfg := testConfig(t)
	h := NewHandler(cfg)
	req := httptest.NewRequest(http.MethodGet, "/images/logo.png", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	original, err := os.ReadFile("assets/logo.png")
	if err != nil {
		t.Fatalf("failed to read original image: %v", err)
	}
	if !bytes.Equal(rr.Body.Bytes(), original) {
		t.Error("served image differs from original file")
	}
}
