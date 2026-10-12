// ビューア動作確認用のダミーページ画像(SVG)を生成する。
//
// 本番では実際の書籍画像に差し替える。ファイル名の規則
// (page-01.svg ... page-NN.svg) は js/reader.js の CONFIG.pageSrc と対応。
//
// 使い方 (リポジトリのルートで実行):
//
//	go run ./tools/generate_dummy_pages
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

var outDir = filepath.Join("frontend", "public", "assets", "pages")

// ページサイズ。A5 相当の縦横比 (1:1.414)
const (
	W = 800
	H = 1131
)

const font = "'Hiragino Sans','Noto Sans JP',sans-serif"

// ページごとのアクセントカラー。めくったことが目で分かるよう少しずつ変える
var accents = []string{
	"#1f3a5f", "#2f6feb", "#0f8a7a", "#b5622a", "#7a3f9d",
	"#c0392b", "#16794c", "#8a6d1f", "#3b5bdb", "#1f3a5f",
}

var escaper = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")

// textOpt は text() の省略可能な引数。ゼロ値の項目は既定値になる
type textOpt struct {
	size   int
	color  string
	weight string
	anchor string
}

func text(x, y int, s string, o textOpt) string {
	if o.size == 0 {
		o.size = 24
	}
	if o.color == "" {
		o.color = "#1b1f24"
	}
	if o.weight == "" {
		o.weight = "400"
	}
	if o.anchor == "" {
		o.anchor = "start"
	}
	return fmt.Sprintf(
		`<text x="%d" y="%d" font-family="%s" font-size="%d" font-weight="%s" fill="%s" text-anchor="%s">%s</text>`,
		x, y, font, o.size, o.weight, o.color, o.anchor, escaper.Replace(s),
	)
}

// bodyLines は本文っぽいグレーの帯を描いて「文章が組まれている」感を出す。
// 描画後の次の y 座標も返す
func bodyLines(top, count, indent int) (string, int) {
	// 行長をばらつかせて自然に見せる
	widths := []int{640, 600, 655, 580, 630, 610, 660, 545}
	var b strings.Builder
	y := top
	for i := range count {
		fmt.Fprintf(&b, `<rect x="%d" y="%d" width="%d" height="10" rx="5" fill="#1b1f24" opacity="0.14"/>`,
			80+indent, y, widths[i%len(widths)]-indent)
		y += 30
	}
	return b.String(), y
}

// frame は全ページ共通の枠・ノンブル(ページ番号)
func frame(pageNo int, accent, inner string) string {
	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %[1]d %[2]d" width="%[1]d" height="%[2]d">
  <rect width="%[1]d" height="%[2]d" fill="#ffffff"/>
  <rect x="0" y="0" width="%[1]d" height="12" fill="%[3]s"/>
  %[4]s
  <line x1="80" y1="%[5]d" x2="%[6]d" y2="%[5]d" stroke="%[3]s" stroke-opacity="0.3" stroke-width="1"/>
  %[7]s
</svg>
`, W, H, accent, inner, H-90, W-80,
		text(W/2, H-55, fmt.Sprintf("- %d -", pageNo), textOpt{size: 20, color: "#8b95a1", anchor: "middle"}))
}

func cover(_ int, accent string) string {
	white := func(size int, weight string) textOpt {
		return textOpt{size: size, color: "#ffffff", weight: weight, anchor: "middle"}
	}
	inner := fmt.Sprintf(`
  <rect width="%[1]d" height="%[2]d" fill="%[3]s"/>
  <rect x="70" y="70" width="%[4]d" height="%[5]d" fill="none" stroke="#ffffff" stroke-opacity="0.45" stroke-width="2"/>
  %[6]s
  %[7]s
  <line x1="260" y1="560" x2="%[8]d" y2="560" stroke="#ffffff" stroke-opacity="0.6" stroke-width="2"/>
  %[9]s
  %[10]s
`, W, H, accent, W-140, H-140,
		text(W/2, 430, "Docker で作る", white(40, "300")),
		text(W/2, 510, "電子書籍サービス", white(56, "700")),
		W-260,
		text(W/2, 620, "— フロントエンド編 —", white(24, "")),
		text(W/2, H-140, "ebook.aws.furusawa.online", white(22, "")),
	)
	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %[1]d %[2]d" width="%[1]d" height="%[2]d">%[3]s</svg>
`, W, H, inner)
}

func toc(pageNo int, accent string) string {
	items := []struct{ ch, title, pg string }{
		{"第1章", "ビューアの構造", "03"},
		{"第2章", "状態管理の考え方", "04"},
		{"第3章", "ページめくりの実装", "05"},
		{"第4章", "入力ハンドリング", "06"},
		{"第5章", "画像のプリロード", "07"},
		{"第6章", "バックエンド連携", "08"},
	}
	var b strings.Builder
	b.WriteString(text(80, 160, "目 次", textOpt{size: 42, weight: "700", color: accent}))
	fmt.Fprintf(&b, `<line x1="80" y1="190" x2="%d" y2="190" stroke="%s" stroke-width="3"/>`, W-80, accent)
	y := 270
	for _, it := range items {
		b.WriteString(text(80, y, it.ch, textOpt{size: 20, color: "#8b95a1"}))
		b.WriteString(text(180, y, it.title, textOpt{size: 26}))
		b.WriteString(text(W-80, y, it.pg, textOpt{size: 22, color: "#8b95a1", anchor: "end"}))
		fmt.Fprintf(&b, `<line x1="180" y1="%[1]d" x2="%[2]d" y2="%[1]d" stroke="#1b1f24" stroke-opacity="0.12" stroke-dasharray="2 4"/>`, y+12, W-110)
		y += 80
	}
	return frame(pageNo, accent, b.String())
}

// chapterHeading は章扉ページ共通の見出し
func chapterHeading(b *strings.Builder, accent string, num int, title string) {
	b.WriteString(text(80, 150, fmt.Sprintf("CHAPTER %d", num), textOpt{size: 18, color: accent, weight: "700"}))
	b.WriteString(text(80, 210, title, textOpt{size: 34, weight: "700"}))
	fmt.Fprintf(b, `<line x1="80" y1="240" x2="240" y2="240" stroke="%s" stroke-width="4"/>`, accent)
}

func chapter(num int, title string) func(int, string) string {
	return func(pageNo int, accent string) string {
		var b strings.Builder
		chapterHeading(&b, accent, num, title)
		lines, y := bodyLines(300, 9, 0)
		b.WriteString(lines)
		// 引用ブロック
		fmt.Fprintf(&b, `<rect x="80" y="%d" width="4" height="110" fill="%s"/>`, y+30, accent)
		b.WriteString(text(110, y+70, "状態を1か所に集めておくと、", textOpt{size: 22, color: "#5b6672"}))
		b.WriteString(text(110, y+105, "描画はその写像として書ける。", textOpt{size: 22, color: "#5b6672"}))
		lines2, _ := bodyLines(y+190, 8, 0)
		b.WriteString(lines2)
		return frame(pageNo, accent, b.String())
	}
}

func figure(num int, title string) func(int, string) string {
	return func(pageNo int, accent string) string {
		var b strings.Builder
		chapterHeading(&b, accent, num, title)
		lines, y := bodyLines(300, 5, 0)
		b.WriteString(lines)
		// 図版: state -> render -> DOM
		fy := y + 40
		boxes := []struct {
			label string
			x     int
		}{{"state", 80}, {"render()", 320}, {"DOM", 560}}
		for _, bx := range boxes {
			fmt.Fprintf(&b, `<rect x="%d" y="%d" width="160" height="80" rx="8" fill="none" stroke="%s" stroke-width="2"/>`, bx.x, fy, accent)
			b.WriteString(text(bx.x+80, fy+48, bx.label, textOpt{size: 22, color: accent, anchor: "middle"}))
		}
		for _, bx := range []int{240, 480} {
			fmt.Fprintf(&b, `<line x1="%[1]d" y1="%[2]d" x2="%[3]d" y2="%[2]d" stroke="%[4]s" stroke-width="2"/>`, bx, fy+40, bx+80, accent)
			fmt.Fprintf(&b, `<path d="M%d %d l-10 -5 v10 z" fill="%s"/>`, bx+80, fy+40, accent)
		}
		b.WriteString(text(W/2, fy+130, fmt.Sprintf("図 %d-1  単方向データフロー", num), textOpt{size: 18, color: "#8b95a1", anchor: "middle"}))
		lines2, _ := bodyLines(fy+175, 7, 0)
		b.WriteString(lines2)
		return frame(pageNo, accent, b.String())
	}
}

func afterword(pageNo int, accent string) string {
	var b strings.Builder
	b.WriteString(text(80, 170, "あとがき", textOpt{size: 34, weight: "700"}))
	fmt.Fprintf(&b, `<line x1="80" y1="200" x2="240" y2="200" stroke="%s" stroke-width="4"/>`, accent)
	lines, _ := bodyLines(270, 14, 0)
	b.WriteString(lines)
	return frame(pageNo, accent, b.String())
}

func colophon(pageNo int, accent string) string {
	rows := [][2]string{
		{"書名", "Docker で作る電子書籍サービス"},
		{"発行", "2026年10月12日 初版"},
		{"著者", "Yuuki Furusawa"},
		{"発行所", "ebook.aws.furusawa.online"},
	}
	var b strings.Builder
	b.WriteString(text(W/2, 300, "奥 付", textOpt{size: 30, weight: "700", anchor: "middle", color: accent}))
	fmt.Fprintf(&b, `<line x1="280" y1="330" x2="%d" y2="330" stroke="%s" stroke-width="2"/>`, W-280, accent)
	y := 410
	for _, r := range rows {
		b.WriteString(text(200, y, r[0], textOpt{size: 20, color: "#8b95a1"}))
		b.WriteString(text(320, y, r[1], textOpt{size: 20}))
		y += 50
	}
	b.WriteString(text(W/2, H-200, "© 2026 ebook.aws.furusawa.online", textOpt{size: 18, color: "#8b95a1", anchor: "middle"}))
	return frame(pageNo, accent, b.String())
}

var pages = []func(pageNo int, accent string) string{
	cover,
	toc,
	chapter(1, "ビューアの構造"),
	figure(2, "状態管理の考え方"),
	chapter(3, "ページめくりの実装"),
	chapter(4, "入力ハンドリング"),
	figure(5, "画像のプリロード"),
	chapter(6, "バックエンド連携"),
	afterword,
	colophon,
}

func main() {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		log.Fatal(err)
	}
	for i, build := range pages {
		n := i + 1
		path := filepath.Join(outDir, fmt.Sprintf("page-%02d.svg", n))
		if err := os.WriteFile(path, []byte(build(n, accents[i])), 0o644); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("wrote %s\n", path)
	}
	fmt.Printf("\n%d pages generated.\n", len(pages))
}
