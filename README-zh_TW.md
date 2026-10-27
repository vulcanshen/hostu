# hostu

[![Release](https://img.shields.io/github/v/release/vulcanshen/hostu)](https://github.com/vulcanshen/hostu/releases)
[![License](https://img.shields.io/github/license/vulcanshen/hostu)](LICENSE)

**Language**: [English](README.md) · 繁體中文

**在終端機裡用 profile 切換 hosts 檔。**

把幾組 hosts 設定存成 profile —— 公司、家裡、客戶現場 —— 要哪一組，按一個鍵就寫進 hosts 檔。每個 profile 是一串 domain，每個
domain 指到一個或好幾個位址；在表單裡一筆一筆加、改，不用自己編 hosts 檔。第一次打開時，hostu 把 hosts 檔現在的內容讀成一個叫
`default` 的 profile，原本有的都不會不見。

![hostu：在 profile 之間移動、加一個有兩個位址的 domain、啟用 office](docs/demo.gif)

## 特色

- **一次一個生效，標得出來**：綠色的勾標出 hosts 檔現在是哪一個 profile；hosts 檔跟它對不上時（之後改了 profile，或別的程式
  寫過 hosts 檔），statusbar 會說。
- **一個 domain 一筆**：在表單裡加、改、刪 domain；一個 domain 可以指到好幾個位址（IPv4 與 IPv6）。名稱與位址打錯當場就擋下來。
- **說了才寫**：改 profile 只改 hostu 自己的 `profiles.yaml`；啟用 profile 時才寫 hosts 檔，寫之前先問，寫出會寫幾個 domain，
  沒有 `localhost` 時警告。
- **舊的 hosts 檔留著**：hostu 第一次寫 hosts 檔之前，先把它原樣留一份。
- **只有寫的時候才要 sudo**：用自己的帳號跑 hostu；hosts 檔要管理員才寫得了時，hostu 開一個小框問 sudo 的密碼，只交給 sudo
  （doas 在 hostu 裡的終端機自己問；Windows 是系統的 UAC 視窗）。
- **從 hosts 檔建 profile**：隨時可以把 hosts 檔現在的內容讀成一個新的 profile。

## 安裝

### 需求

- macOS、Linux 或 Windows。
- 寫系統的 hosts 檔：macOS、Linux 要有 `sudo` 或 `doas`；Windows 要有 `gsudo` 或 Windows 11 的 `sudo`。
- [Nerd Font](https://www.nerdfonts.com) 的終端機設定：icon 與外框都是 Nerd Font 的字。
- truecolor（24-bit 色）的終端機：淡化的顏色與層色在 256 色下分不出來。
- 終端機至少 60 欄寬、24 列高；比這小時 hostu 只畫名稱、需要的大小與怎麼離開（連按兩次 `Ctrl-C`），終端機拉大就回到原本的樣子。

hostu 啟動時會量 icon 與 powerline 符號各佔幾格。量錯時可以自己設：

| 變數 | 設什麼 |
|---|---|
| `HOSTU__ICON_WIDTH` | icon 佔幾格，只給 hostu |
| `HOSTU__POWERLINE_WIDTH` | powerline 符號佔幾格，只給 hostu |
| `TERMINU__ICON_WIDTH` | icon 佔幾格，terminu 家族每個 app 共用：設一次，家族的 app 都讀得到 |
| `TERMINU__POWERLINE_WIDTH` | powerline 符號佔幾格，terminu 家族每個 app 共用 |
| `LC_TERMINU_ICON_WIDTH` | icon 佔幾格，有 ssh 的家族 app 隔著 ssh 帶過來 |
| `LC_TERMINU_POWERLINE_WIDTH` | powerline 符號佔幾格，有 ssh 的家族 app 隔著 ssh 帶過來 |

icon 與 powerline 符號各自照表上的順序取第一個有設的，都沒設才量。在家族 app 的終端機裡跑時，`TERMINU__` 的兩個由外層的 app
設好；從家族 app 隔著 ssh 連過來時是 `LC_TERMINU_` 的兩個。

### Homebrew

```sh
brew install vulcanshen/tap/hostu
```

### Scoop（Windows）

```powershell
scoop bucket add vulcanshen https://github.com/vulcanshen/scoop-bucket
scoop install hostu
```

### 安裝腳本

```sh
curl -fsSL https://raw.githubusercontent.com/vulcanshen/hostu/main/install.sh | sh
```

不用 sudo：裝到 `~/.local/bin`。Windows 在 PowerShell 裡：

```powershell
irm https://raw.githubusercontent.com/vulcanshen/hostu/main/install.ps1 | iex
```

### Go

```sh
go install github.com/vulcanshen/hostu/cmd/hostu@latest
```

### 第一次啟動

hostu 把 hosts 檔讀成叫 `default` 的 profile，標成生效中（hosts 檔就是它），存進 `profiles.yaml`。啟用 profile 之前不會寫 hosts 檔。

### 更新與移除

更新：用同樣的方式再裝一次。移除：

```sh
brew uninstall hostu
```

用安裝腳本裝的：

```sh
curl -fsSL https://raw.githubusercontent.com/vulcanshen/hostu/main/uninstall.sh | sh
```

```powershell
irm https://raw.githubusercontent.com/vulcanshen/hostu/main/uninstall.ps1 | iex
```

移除腳本刪 profile 與舊 hosts 檔的副本之前會先問。移除 hostu 不動 hosts 檔：它留在 hostu 最後寫的樣子。

## 快速開始

1. 執行 `hostu`。`default` 這個 profile 就是你原本的 hosts 檔。
2. 按 `n` 建一個 profile，例如 `office`；`Tab` 移到它的 domain。
3. 在那裡按 `n` 加 domain：在 Domain 上按 `Enter`、打 `git.corp.local`、`Enter`；`j`、在 Addresses 上按 `Enter`、打 `10.1.2.3`、
   `Enter`；`Tab`、`Enter` 就加進去了。
4. 按 `1` 回到 profile，在 `office` 上按 `Enter`：hostu 先問，再按 `Enter` 就寫進 hosts 檔。
5. 在 `default` 上按 `Enter` 就換回原本的 hosts 檔。`q` 離開。

## 使用方式與按鍵

```
hostu            啟動
hostu version    印出版本（-v、--version）
hostu help       印出命令列的用法（-h、--help）
```

家族每個 app 都一樣的鍵：

| 鍵 | 做什麼 |
|---|---|
| `Space` | 這裡能做什麼：對這一項、對這個 panel、對整個 app |
| `?` | 這裡能按的每一個鍵 |
| `!` | App Log：這次打開以來 hostu 做了什麼、哪裡出了問題 |
| `Tab`、`1`–`2` | 換到另一個 panel；照編號跳到那個 panel |
| `j\|k` | 移動；走過最後一個回到第一個 |
| `gg\|G` | 到最上面、最下面 |
| `Enter` | 最直觀的那件事：啟用 profile、編輯 domain |
| `Esc` | 關掉最上面的 popup，一次一層；不會離開 hostu |
| `q` | 離開 |

hostu 自己的：

| 鍵 | 在哪裡 | 做什麼 |
|---|---|---|
| `Enter` | `[1] PROFILES` | 啟用這個 profile：先問，再寫進 hosts 檔 |
| `p` | `[1] PROFILES` | 預覽這個 profile 會寫成的 hosts 檔 |
| `r`、`c` | `[1] PROFILES` | 改名；複製成一個新的 profile |
| `D` | `[1] PROFILES` | 刪掉這個 profile，先問（生效中的不能刪） |
| `n` | `[1] PROFILES` | 新增一個空的 profile |
| `i` | `[1] PROFILES` | 把 hosts 檔現在的內容讀成一個新的 profile |
| `Enter` | `[2] DOMAINS` | 編輯這個 domain 與它的位址 |
| `D` | `[2] DOMAINS` | 刪掉這個 domain，先問 |
| `n` | `[2] DOMAINS` | 新增 domain：NEW DOMAIN 表單 |

表單的 Addresses 可以一次打好幾個位址，用空白隔開：`10.1.2.3 fd00::3`。

## 設定與資料存放位置

| 什麼 | 在哪裡 | 用什麼改 |
|---|---|---|
| 設定 `config.yaml` | `$XDG_CONFIG_HOME/hostu`，沒設就是 `~/.config/hostu` | `HOSTU__CONFIG` |
| profile `profiles.yaml` | `~/.hostu` | `HOSTU__DATA` |
| hostu 第一次寫之前的 hosts 檔 `hosts.original` | `~/.hostu` | `HOSTU__DATA` |
| hostu 寫的 hosts 檔 | `/etc/hosts`；Windows 是 `%SystemRoot%\System32\drivers\etc\hosts` | `HOSTU__HOSTS_FILE` |

Windows 上的 `~` 是 `%USERPROFILE%`。`config.yaml` 只有一個設定，terminu 家族共用的：

| key | 做什麼 | 預設 |
|---|---|---|
| `input_tab_width` | hostu 顯示的文字裡，Tab 畫成幾格空白 | `4` |

`profiles.yaml` 是 hostu 自己的格式，每改一次就存一次。想試 hostu 又不想動到 hosts 檔，把 `HOSTU__HOSTS_FILE` 指到一份複本就好。

`config.yaml` 或 `profiles.yaml` 讀不了時（例如手改壞了），hostu 一開始就先問：`Enter` 把它改名成 `.bak`，像第一次打開
一樣開始，`Esc` 離開，印出是哪個檔、哪裡不對。

## 限制

- hostu 照 profile 寫整個 hosts 檔：你自己在 hosts 檔寫的註解與內容不會留在裡面（第一次寫之前的樣子留在 `hosts.original`）。
  要改請在 hostu 裡改 profile，不要直接改 hosts 檔。
- hosts 檔裡不是「位址加名稱」的行（打錯的、hostu 不認得的位址）不會讀進 profile；hostu 會說跳過了幾行。
- hostu 不監看 hosts 檔：啟動時與寫完之後才讀。

## 相關連結

- [CHANGELOG](CHANGELOG.md)

## terminu 家族

hostu 遵循 [terminu design](https://github.com/vulcanshen/terminu)，用 [terminu-go](https://github.com/vulcanshen/terminu-go)
做的。家族的其他 app：

| App | 是什麼 |
|---|---|
| [kbu](https://github.com/vulcanshen/kbu) | Kubernetes 的 TUI 儀表板 |
| [filu](https://github.com/vulcanshen/filu) | 不用學就會用的終端機檔案管理員 |
| [sshu](https://github.com/vulcanshen/sshu) | ssh 與 sftp 的終端機介面 |
| [webu](https://github.com/vulcanshen/webu) | 把網頁當文件讀的終端機瀏覽器 |
| [locku](https://github.com/vulcanshen/locku) | 終端機的螢幕保護，帶 PIN 鎖 |
| [kamonu](https://github.com/vulcanshen/kamonu) | 一個字一個字畫家族 mark（家紋）的終端機編輯器 |
| [ccdrawu](https://github.com/vulcanshen/ccdrawu) | Claude Code 的側邊欄：用量、context、花費與 git 跟對話並排 |
| [demou](https://github.com/vulcanshen/demou) | 家族的操作教學 app，邊按邊學 |

## License

[GPL-3.0](LICENSE)
