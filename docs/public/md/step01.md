# feat(frontend): 電子書籍ビューア（ページめくり）の実装

バックエンドと切り離したフロントエンド単体として、画像ベースの電子書籍ビューアを実装しました。ビルドツールなしの Vanilla JS / nginx 配信です。

## 変更ファイル

| パス | 役割 |
|---|---|
| `frontend/public/index.html` | DOM の骨組み（レイヤ2枚・ツールバー・シークバー） |
| `frontend/public/css/style.css` | 見た目と**アニメーション本体**（`transition` は CSS 側） |
| `frontend/public/js/reader.js` | 状態管理と回転角の付け外し |
| `frontend/public/assets/pages/page-01..10.svg` | 動作確認用ダミーページ10枚 |
| `tools/generate_dummy_pages/main.go` | ダミーページ生成ツール（Go。`go run ./tools/generate_dummy_pages`） |
| `frontend/Dockerfile` / `frontend/nginx.conf` | 静的配信 |
| `compose.yaml` | ローカル起動（`:8080`）。`backend` サービスの追加箇所をコメントで確保 |

公開されるのは `frontend/public/` 配下のみです。`Dockerfile` / `nginx.conf` / `tools/` / `go.mod` はその外に置いてあるため Web からは参照されません。`compose.yaml` では `public/` を**1つだけ**バインドマウントしています（サブディレクトリを個別にマウントすると Docker Desktop 側でマウントが外れ、コンテナ内が空になることがあるため）。あわせて `sendfile off`（`server` コンテキスト）で、更新後に古い内容が配信される問題を回避しています。

## ロジック

### 単方向データフロー

```
入力 ──▶ state を更新 ──▶ render() ──▶ DOM
```

状態は3つだけ。派生値は持たず都度計算するため、同期ズレが起きません。

```js
const state = {
  current:    1,      // 現在ページ (1 始まり)
  direction:  'ltr',  // 綴じ方向。実行中に切り替え可能
  isFlipping: false,  // めくり中は操作を受け付けない
};
```

DOM を直接触るのは `render()` と `goTo()` / `finishFlip()` のみ。ページ画像の差し替えだけは、アニメーションのタイミングと不可分なため `goTo()` 側に置いています。

### レイヤ構造

重ねる `<img>` は **2枚だけ**。総ページ数が増えても DOM は増えません。

```
.book
├─ #pageUnder   ← 下地。常に表示されている
└─ #sheet       ← 綴じ目を軸に回転する「紙」
   ├─ #pageFlip ← 紙の表面に貼る画像
   └─ #sheetShade
```

静止時は下地と紙が**同じページ**を表示しています。これにより、めくり終わりに紙の角度を `-180deg` → `0deg` へ戻す瞬間にも下地に正しい絵があり、チラつきが構造的に発生しません。「紙を隠してから差し替える」といった順序制御やデコード完了待ちが不要になります。

### めくりの原理

綴じ目（`transform-origin`）を軸に紙を `rotateY(0deg → -180deg)` させ、`backface-visibility: hidden` により90度を超えた紙が自動的に消えて下地が現れます。**消すタイミングを JS で判定する必要がありません。**

進む／戻るは同じ仕組みで、分岐は2点のみです。

| | 下地 `#pageUnder` | 紙 `#pageFlip` | 開始角 | 終了角 |
|---|---|---|---|---|
| **進む** | 遷移先を敷く | 現在ページ（そのまま） | `0deg` | `-180deg` |
| **戻る** | 現在ページ（そのまま） | 遷移先を貼る | `-180deg` | `0deg` |

補間は CSS の `transition` が行い、JS は角度の値を入れ替えるだけです（`requestAnimationFrame` ループなし）。

### `goTo()` の手順

1. 絵を決める（進む＝下地に遷移先／戻る＝紙に遷移先）
2. `transition` を切った状態で開始角に置く
3. `forceReflow()` でレイアウトを確定させる
4. `transition` を入れて終了角へ。`animationMs` 後に `finishFlip()`

```js
el.sheet.classList.remove('is-animating');
el.sheet.style.transformOrigin = spineOrigin();
setSheetAngle(forward ? 0 : angle);

forceReflow(el.sheet);              // ← 3

el.sheet.classList.add('is-animating');
setSheetAngle(forward ? angle : 0);
```

**手順3が必須です。** ブラウザは同一フレーム内のスタイル変更をまとめて適用するため、これがないと開始角が無かったことにされ、`transition` が再生されません。`void node.offsetWidth` の読み取りがレイアウト計算を強制します。

後始末は `transitionend` ではなく `setTimeout` で予約しています。タブが非アクティブになると `transitionend` が発火せず、`isFlipping` が `true` のまま固まって操作不能になるためです。

### 綴じ方向

左綴じ／右綴じの分岐は3関数に閉じ込めてあります。

| 関数 | ltr | rtl | 効果 |
|---|---|---|---|
| `spineOrigin()` | `left center` | `right center` | 回転軸の位置 |
| `flipAngle()` | `-180` | `180` | 倒れる向き |
| `deltaForSide()` | 右＝次へ | 左＝次へ | タップ／矢印キーの対応 |

影の向きのみ CSS 側（`[data-direction="rtl"]`）で反転させています。

### 設定

`CONFIG` を変えるだけで大半の挙動を調整できます。バックエンド接続時は `init()` で書誌情報を取得し `CONFIG` を上書きする想定です。

| キー | 既定値 | 内容 |
|---|---|---|
| `totalPages` | `10` | 総ページ数 |
| `pageSrc(n)` | `assets/pages/page-NN.svg` | ページ番号 → 画像URL |
| `direction` | `'ltr'` | 綴じ方向 |
| `animationMs` | `600` | めくり時間。起動時に CSS 変数 `--flip-duration` へ反映 |
| `preloadRange` | `2` | 前後何ページを先読みするか |
| `swipeThreshold` | `50` | スワイプ判定の最小移動量(px) |
| `storageKey` | — | しおりの `localStorage` キー |

## 操作

左右タップ / `←` `→` / `PageUp` `PageDown` / `Space` / `Home` `End` / 横スワイプ / シークバー / 全画面（`F`）

## その他の実装

- 前後2ページのプリロード（`new Image()` でキャッシュに載せる）
- `localStorage` によるしおり復帰。読み書きは `try/catch` で保護（シークレットウィンドウ対策）
- `prefers-reduced-motion: reduce` で `--flip-duration` を `1ms` にし即時切り替え
- `aria-live="polite"` でページ番号を読み上げ
- デバッグ用に `window.reader` を公開（`reader.goTo(5)` など）

## ダミーページ生成ツール（Go）

元の Python 版（`tools/generate_dummy_pages.py`）を Go に移植しました。出力される SVG は、ドメイン表記と発行日を除いて Python 版とバイト単位で同一です。

```sh
# リポジトリのルートで実行（出力先がカレントディレクトリからの相対パスのため）
go run ./tools/generate_dummy_pages
```

ページごとに「ページ番号とアクセントカラーを受け取って SVG 文字列を返す関数」を用意し、スライス `pages` に並べています。ページを足すときは関数を1つ書いて `pages` に追加し、`accents` にも色を足します。

```go
var pages = []func(pageNo int, accent string) string{
	cover,
	toc,
	chapter(1, "ビューアの構造"), // 引数を閉じ込めた関数を返す
	figure(2, "状態管理の考え方"),
	// ...
}
```

### Python から Go への置き換え

| Python | Go | ポイント |
|---|---|---|
| キーワード引数 `text(x, y, s, size=20)` | オプション構造体 `text(x, y, s, textOpt{size: 20})` | Go にはキーワード引数もデフォルト値も無い。ゼロ値（`0` / `""`）を「未指定」とみなして関数の先頭で既定値を埋める |
| f-string `f'width="{W}"'` | `fmt.Sprintf` + `%[1]d` | 長いテンプレート（`frame()` / `cover()`）では番号付き指定子で同じ値を何度も参照する |
| `lambda n, a: chapter(n, a, 1, ...)` | `chapter(1, ...)` がクロージャを返す | 章番号とタイトルを先に束縛し、シグネチャを `func(int, string) string` に揃える |
| タプルを返す `return "".join(out), y` | 多値返却 `return b.String(), y` | 使わない方は `_` で捨てる |
| `"".join(out)` | `strings.Builder` | `fmt.Fprintf(&b, ...)` で直接書き足せる |
| 自作の `esc()` | `strings.NewReplacer` | `html.EscapeString` は引用符も変換して出力が変わるため、`&` `<` `>` の3つだけを置換する |
| `for i in range(count)` | `for i := range count` | 整数に対する `range` は Go 1.22 から |
| `W // 2` | `W / 2` | 整数どうしの割り算は Go では切り捨て |
| `Path(__file__).parent.parent` | カレントディレクトリからの相対パス | `go run` は実行ファイルを一時ディレクトリに作るため、`os.Executable()` の位置は当てにならない。シンプルさを優先してルートで実行する前提にした |
| 例外で停止 | `if err != nil { log.Fatal(err) }` | エラーは戻り値で受け取り、その場で処理する |

SVG は文字列を組み立てているだけなので、Python 版との一致は `diff` で確認できます。生文字列リテラル（`` `...` ``）内の改行やインデントもそのまま出力されるため、整形時に中身の空白を変えないよう注意してください。

## 動作確認

- 全アセットが `200` で配信されること（HTML / CSS / JS / SVG / `/healthz`）
- `getElementById` 12箇所すべてが HTML 側に存在
- `CONFIG.animationMs`(600) と CSS `--flip-duration`(600ms) の一致
- 画像枚数(10) と `totalPages`(10) の一致

**未確認:** めくりアニメーションの見た目はブラウザでの目視確認ができていません。レビュー時に `docker compose up -d` → http://localhost:8080 でご確認ください。

## 今後

- `backend/` の追加と `pageSrc` の API 切り替え
- 見開き表示（レイヤ2枚構造と回転の仕組みはそのまま流用可能）
- しおりのサーバ保存（`saveProgress()` / `loadProgress()` の差し替えのみ）

🤖 Generated with [Claude Code](https://claude.com/claude-code)
