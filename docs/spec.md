# hostu 規格

照 terminu design（tdp v0.9.3）用 terminu-go v0.4.4 做。畫面、按鍵、外框、顏色、popup 的開關這些 tdp 已經規定的不寫在這裡，
照 tdp 做。

## 一句話

**在終端機裡用 profile 切換 hosts 檔。**

hostu 把幾份 hosts 設定存成 profile（例：公司、家裡、客戶現場），一次只有一份生效。選一份按 `Enter`，hostu 把它寫成 hosts 檔；
profile 裡一個 domain 一筆，每筆可以有好幾個位址，一筆一筆加、改、刪，不用自己編 hosts 檔。第一次打開時，hostu 把現在的 hosts 檔
讀成一份 `default` profile。

## 需求（user 給的）

- 管理本機的 hosts 檔；master-detail：側邊是存著的 profile 清單，旁邊是那個 profile 的內容。
- 一次只有一個生效，用 icon 標出來；可以新增、編輯、切換生效的那一個。
- profile 存成 hostu 自己的格式（`profiles.yaml`）；啟用一個 profile 時，才把它匯出、覆蓋 hosts 檔（設定是一回事，匯出是另一回事）。
- 能把現在的 hosts 檔解析成 profile，用來建「預設」profile。
- 在 TUI 裡一個 domain 一個 domain 加，或一個 domain 加好幾個位址；不讓使用者直接編 hosts 檔。
- 支援 Windows、macOS、Linux。參考舊的 CLI `~/Documents/sideproj/hostfile`。

## 版面

一個畫面、兩個 panel，最上面一列 statusbar。

```
hosts: /etc/hosts (in sync)     active: office
┏[1] PROFILES━━━━━━━━━━━━━┓┌[2] DOMAINS: office─────────────────────────────────┐
┃   NAME         DOMAINS  ┃│ DOMAIN              ADDRESSES                      │
┃▓󰗠 office▓▓▓▓▓▓▓▓▓▓▓▓3▓▓┃│ localhost           127.0.0.1, ::1                 │
┃   home               2  ┃│ broadcasthost       255.255.255.255                │
┃   default            2  ┃│ git.corp.local      10.1.2.3                       │
┃                         ┃│                                                    │
┗━ Enter:activate ━━━━━━━━┛└─ Enter:edit n:new ─────────────────────────────────┘
Space:menu ?:help !:log Tab|1–2:panels q:quit
```

- **[1] PROFILES**（一般清單）：每個 profile 一列，`NAME`（文字）與 `DOMAINS`（幾個 domain，數字靠右）。生效的那一個名稱前面畫
  `nf-md-check_circle`（U+F05E0，Green），其他的空兩格，名稱對齊。順序是建立的順序（資料本身有順序，tdp layout/panel）。
- **[2] DOMAINS: <profile>**（明細，跟著 [1] 的 cursor 即時換，label 帶 profile 的名字）：每個 domain 一列，`DOMAIN` 與 `ADDRESSES`
  （位址用 `, ` 接起來，放不下從尾端截）。順序是加進來的順序。欄放不下時收掉 `ADDRESSES`，`DOMAIN` 永遠留著。
- **statusbar**：`hosts: <hosts 檔的路徑> <狀態>  active: <生效的 profile>`。狀態是 hosts 檔現在的內容跟生效的 profile 比的結果：
  hosts 檔讀不到是 Red 的 `(can't read)`；對不上（改了還沒寫、或 hosts 檔被別的程式改過）是 Peach 的 `(differs)`；對得上是
  `(in sync)`；沒有生效的 profile 時空著。狀態補空白到最寬的 `(can't read)`，生效的 profile 放最後，欄位不跟著動（tdp L2）。
  `active` 的值 Lavender（你在哪裡），沒有時寫 `none`；其他 Overlay0。
- **最低尺寸 80×24**（tdp L1 可以調低）：兩個 panel 與所有 popup 在 24 列都放得下；常用的終端機（80×24）就能用。

### [1] PROFILES 的動作

| 範圍 | 動作 | 做什麼 |
|---|---|---|
| item | `[Enter] Activate` | 把這個 profile 寫成 hosts 檔（見「啟用」） |
| item | `[p]review` | 用 viewer 看它會寫成的 hosts 檔全文 |
| item | `[r]ename` | 改名字（text input，舊值當提議） |
| item | `[c]opy` | 複製成一個新的 profile（text input，提議 `<名字> copy`） |
| item | `[D]elete` | 刪掉，先 confirm；生效中的那一個停用（tdp M6），要刪先啟用別的 |
| panel | `[n]ew` | 新增一個空的 profile（text input 打名字） |
| panel | `[i]mport` | 把 hosts 檔現在的內容讀成一個新的 profile（text input，提議 `imported`） |

新增、複製、匯入之後 cursor 移到新的那一個（tdp K3）。名字不能空、不能跟別的重複，在 text input 裡就擋（錯誤列）。

### [2] DOMAINS 的動作

| 範圍 | 動作 | 做什麼 |
|---|---|---|
| item | `[Enter] Edit` | 打開表單 `EDIT: <domain>`，改 domain 與位址 |
| item | `[D]elete` | 從這個 profile 刪掉這個 domain，先 confirm |
| panel | `[n]ew` | 打開表單 `NEW DOMAIN`，加一個 domain 與它的位址 |

**表單**（新增與編輯同一張，左右式兩欄，按鈕 `Add`／`Save`）：

```
┏━NEW DOMAIN━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃                                                ┃
┃ Domain     git.corp.local                      ┃
┃ Addresses  10.1.2.3 fd00::3                    ┃
┃                                                ┃
┃                                                ┃   ← 錯誤列
┃ ────────────────────────────────────────────── ┃
┃                                           Add  ┃
┃                                                ┃
┗━ Enter:edit Tab:actions Esc:cancel ━━━━━━━━━━━━┛
```

- **Domain**（text input）：主機名稱的格式（英數、`-`、`_`、`.`，每段 1–63 字、全長 253 字以內），不合格在 input popup 裡擋。
  `_` 不是主機名稱的字元，但 hosts 檔收、內部的機器名常有（例：`my_host`）；不收的話，解析 hosts 檔時有它的行會被跳過。
- **Addresses**（text input）：一個或好幾個 IPv4／IPv6 位址（可以帶 zone，例：`fe80::1%lo0`），用空白隔開；每一個都要是位址，在
  input popup 裡擋，錯誤寫出哪一個不是。一個 domain 好幾個位址就在這一欄一起打。
- 送出時檢查：兩欄都要有；domain 不能跟這個 profile 裡別的 domain 重複（不分大小寫）。
- 選 text input 而不是一個位址一列：位址通常一兩個（IPv4 與 IPv6），一欄打完最快；用一列一個位址，表單要能長出新的列，
  components 沒有這種欄位。

### 全域動作

只有 `[q]uit`（tdp K9；直接離開，沒有要問的）。

### App Log

`!` 打開（tdp dialog/note，terminu-go 內建；toast 與 error popup 自己會記進去）。hostu 另外記：

- 使用者做的事與結果（OK）：新增、改名、複製、刪除 profile，新增、儲存、刪除 domain。啟用有 Success toast，不另外記。
- hostu 自己做的事（INFO）：第一次寫 hosts 檔之前留下 `hosts.original`。第一次打開建 `default` 有 toast，搬開讀不了的檔
  （「讀不了時」）由 terminu-go 記。

## 啟用

在 [1] 的 profile 上按 `Enter`：

1. **confirm**：`Write office to /etc/hosts?`，上面寫出會寫幾個 domain。profile 裡沒有 `localhost` 時加一句警告（Peach）：
   `office has no localhost: programs that connect to localhost stop working.`
2. **第一次寫 hosts 檔之前**，把原本的 hosts 檔原樣複製到資料目錄的 `hosts.original`（只做這一次，之後不蓋掉）。解析成 profile
   會丟掉註解與空白列，原本的檔留一份，什麼都找得回來。
3. **寫**：hosts 檔整個換成這個 profile 匯出的內容（見「資料」）。
   - 直接寫得進去就直接寫（用 `HOSTU__HOSTS_FILE` 指到自己的檔、用管理員跑時）。
   - **macOS、Linux 沒有權限時**：先試 `sudo -n`（不問密碼；剛輸入過密碼時直接成功）；要密碼時開 terminal popup
     `󰆍 SUDO: /etc/hosts` 跑 `sudo cp <暫存檔> /etc/hosts`，使用者在裡面打密碼，跑完 popup 自己關。沒有 sudo 時用 `doas`。
     這個 terminal 只跑一個程式做一件事，沒有歷史與選取模式，每個鍵都給 sudo（tdp dialog/terminal）。
   - **Windows 沒有權限時**：用 `sudo`（Windows 11 24H2 起內建）或 `gsudo` 跑複製，系統跳出 UAC 的視窗；terminu-go 的終端機只有 Unix，
     而 UAC 本來就不在終端機裡問。兩個都沒有時開 error popup，寫出要裝 gsudo 或用系統管理員開終端機。
4. **成功**：這個 profile 變成生效的那一個（icon 移過來），Success toast `Wrote office to /etc/hosts`（動詞在前，App Log 照同一句記，大寫開頭）。失敗：error popup 寫出原因，
   生效的那一個不變。

## 資料

| 什麼 | 放在哪（tdp E3） | 覆寫 |
|---|---|---|
| 設定 `config.yaml`（`input_tab_width`，預設 4；preview 的 viewer 用到） | `$XDG_CONFIG_HOME/hostu/`（fallback `~/.config/hostu/`） | `HOSTU__CONFIG` |
| profile `profiles.yaml`（使用者寫給 hostu 的清單，算資料） | `~/.hostu/` | `HOSTU__DATA` |
| 原本的 hosts 檔 `hosts.original`（hostu 替使用者留的原值，刪了就找不回來，算資料） | 同上 | `HOSTU__DATA` |
| hosts 檔 | macOS、Linux `/etc/hosts`；Windows `%SystemRoot%\System32\drivers\etc\hosts` | `HOSTU__HOSTS_FILE` |

Windows 上 `~` 是 `%USERPROFILE%`（tdp E3）。沒有狀態與快取要存。hostu 還沒發過版，`profiles.yaml` 從設定目錄搬到資料目錄
不用搬家（tdp E3「換了位置時」）。

**讀不了時**（tdp E3）：

- `config.yaml` 或 `profiles.yaml` 讀不了（格式錯、`profiles.yaml` 的內容檢查不過、權限……）：啟動時開 `MOVE ASIDE` confirm，寫出路徑與
  原因（yaml 的錯拿掉開頭的 `yaml:`，接成一行，型別不對寫成 `line 1: 3 is not a list`）。兩個都讀不了時先問 `config.yaml`，搬好再問
  `profiles.yaml`。都問完 hostu 才開始（讀 hosts 檔、第一次打開時建 `default`），所以按 `Esc` 時什麼都還沒寫。
  - `Enter`：改名成 `.bak`（已有就 `.bak.1`、`.bak.2`…），像第一次打開一樣開始（tdp E3-18）：`config.yaml` 不另外建，用預設值（第一次
    打開也不建）；`profiles.yaml` 把 hosts 檔讀成 `default` 存起來。
  - `Esc`：離開 hostu，路徑與原因印到 stderr。
- hosts 檔讀不到：不是 hostu 自己的檔，hostu 照常開著；啟動時開 error popup 寫原因（記進 App Log），statusbar 寫 `(can't read)`。

**`profiles.yaml`**：

```yaml
active: office
profiles:
  - name: office
    domains:
      - domain: localhost
        addresses: [127.0.0.1, "::1"]
      - domain: git.corp.local
        addresses: [10.1.2.3]
```

- 每改一次就存（先寫暫存檔再換名，存到一半斷掉也不會壞掉）。
- **第一次打開**（還沒有 `profiles.yaml`）：把 hosts 檔讀成 `default`，設成生效的那一個，存起來；Info toast 說從哪裡讀的，
  有跳過的行時改用 Warning，加上跳過了幾行（跟匯入一樣）。

**匯出成 hosts 檔**：開頭兩行註解，接著每個 domain 的每個位址一行，位址那一欄補空白對齊：

```
# Written by hostu from the profile "office".
# hostu writes this whole file again when you activate a profile: change it in hostu.
127.0.0.1        localhost
::1              localhost
10.1.2.3         git.corp.local
```

**解析 hosts 檔**：去掉 `#` 之後的字與空白列；每一行第一個字是位址、後面是 domain；同一個 domain 出現在好幾行，位址照出現的順序
放在一起。第一個字不是位址、或 domain 不合格式的行跳過，第一次打開與匯入完的 toast 寫跳過了幾行。「生效的 profile 跟 hosts 檔對不上」就是拿
解析出來的結果跟 profile 比。

## 命令列

`hostu` 打開 TUI；`hostu version`（`--version`、`-v`）、`hostu help`（`-h`、`--help`）照 tdp E1。沒有別的指令。

## 不做的

- 不做 hosts 檔裡的「只管一塊」（舊的 hostfile 的 managed block）：user 要的是整個 profile 匯出、覆蓋 hosts 檔。
- 不做停用單一 domain、搜尋、JSON 匯出入、`$EDITOR` 直接編 hosts 檔：需求沒有，也不讓使用者直接編 hosts 檔。
- 不監看 hosts 檔：啟動時與每次寫完讀一次，對不上時 statusbar 說。

## tdp 沒規範、要回報的地方

（沒有。寫系統檔要提權時怎麼問密碼，tdp v0.9.0 照 hostu 的做法寫進 dialog/terminal 的「Raising privileges」。）

## 里程碑

每一塊做完都 `make check`、用 harness 的終端機開起來看、給 user 看畫面，再 commit。

1. **骨架**：照 tdp project 開 repo（範本檔、Makefile、CHANGELOG、兩份 README、dev-remarks、LICENSE、`docs/icon.svg`）；
   `hostu version`、`hostu help`；兩個空的 panel、statusbar、splash。
2. **資料**：`profiles.yaml`、`config.yaml`、hosts 檔的解析與匯出、第一次打開建 `default`；兩個 panel 顯示 profile 與 domain、生效的
   icon、statusbar。
3. **編輯**：profile 的新增、改名、複製、刪除、匯入；domain 的新增、編輯、刪除（表單）。
4. **啟用**：confirm、`hosts.original`、直接寫／`sudo -n`／terminal popup 的 sudo／Windows 的 UAC、preview、statusbar 的對不上。
5. **收尾**：README 寫完整、dev-remarks、tdp L4 跨尺寸與 tdp E5 兩種寬度的畫面測試、splash 與 `docs/icon.svg` 一致的測試；然後 judge。
