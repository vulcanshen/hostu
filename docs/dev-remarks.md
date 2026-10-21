# hostu 開發者備忘

用 profile 切換 hosts 檔的 TUI，遵循 [terminu design principle](https://github.com/vulcanshen/terminu)（tdp v0.9.2），用
terminu-go v0.4.3 做。

## 運作方式

- **profile 是 hostu 自己的格式**（`internal/profile`）：`profiles.yaml` 存每個 profile 的名字與 domain（一個 domain 一筆、一個或好幾個
  位址），加上 `active`：最後寫進 hosts 檔的是哪一個。每改一次就存，先寫暫存檔再換名。
- **hosts 檔是匯出出去的**（`internal/hosts`）：`Format` 寫兩行註解加上每個 domain 的每個位址一行；`Parse` 反過來讀，去掉註解，
  同一個 domain 在好幾行的位址收在一起，不是「位址加名稱」的行跳過並計數。「hosts 檔跟生效的 profile 對不上」就是拿 `Parse` 的結果
  跟 profile 比（`hosts.Same`），所以兩邊用同一套規則，`default` 一開始一定對得上。
- **寫 hosts 檔**（`hosts.Write`）：直接寫得進去就直接寫；權限不夠時把內容放進暫存檔，macOS、Linux 先試 `sudo -n cp`（或 `doas -n`），
  要密碼時把 `sudo cp <暫存檔> <hosts 檔>` 交回給 UI，在 terminal popup 裡跑，讓 sudo 自己問密碼；Windows 用 `gsudo` 或 `sudo` 跑
  PowerShell 的 `Copy-Item`，系統跳 UAC。寫在 tea.Cmd 裡，不擋畫面。`cp` 蓋過既有的檔，hosts 檔的擁有者與權限不變。
- **第一次寫之前**把 hosts 檔原樣複製到資料目錄的 `hosts.original`，之後不再蓋掉（`keepOriginal`）。
- **位置**（`internal/place`）：tdp E3 的設定目錄（`HOSTU__CONFIG`，`config.yaml`）、資料目錄（`HOSTU__DATA`，`profiles.yaml` 與
  `hosts.original`），hosts 檔是 `HOSTU__HOSTS_FILE` 或系統的。
- **自己的檔讀不了**：`config.Load`、`profile.Load` 讀不了時給預設值、空的 store 與錯誤；`ui` 把它們排進 `unread`，啟動時一個一個用
  terminu-go 的 `MoveAside` 問（`askUnread`），搬好一個才問下一個。都問完才 `begin`：讀 hosts 檔、第一次打開時建 `default` 存檔，
  所以按 `Esc` 離開時什麼都沒寫。原因先過 `reason`：拿掉 `*os.PathError` 的路徑、yaml 的 `yaml:`，`*yaml.TypeError` 的好幾行接成
  一行（confirm 是 components「控制字元」裡的「名字與短字」，`\n` 會畫成紅色的名字），型別不對的那句改成使用者的話
  （`line 1: 3 is not a list`，不寫 Go 的型別名）。
- **App Log**：terminu-go 內建（`!`、footer 的提醒、toast 與 error popup 自己記）；hostu 用 `App.Log` 記使用者的動作與背景做的事
  （`hosts.original`）。

## 設計決定

- **user 的需求**（2026-10-18、19）：側邊是 profile 清單、旁邊是內容；一次一個生效、要有 icon；profile 用 hostu 自己的格式存，啟用時才匯出
  覆蓋 hosts 檔；能把 hosts 檔解析成 profile 建「預設」；在 TUI 裡一個 domain 一個 domain 加、一個 domain 好幾個位址，不直接編 hosts 檔；
  Windows、macOS、Linux。其餘照 tdp 與 TUI 的 UX 決定，寫在 `docs/spec.md`。
- **整個 hosts 檔就是 profile**：不學舊的 hostfile 只管 hosts 檔裡的一塊；user 要的是「匯出覆蓋」。代價是 hosts 檔裡手寫的註解進不了
  profile，所以第一次寫之前留 `hosts.original`。
- **`Enter` 是啟用**：這個 app 最常做的就是換 profile；明細本來就跟著 cursor 換（tdp K3、layout/panel 的明細），不用 `Enter` 進去看。
  寫的是系統檔，所以每次都先 confirm（tdp F6），confirm 寫出會寫幾個 domain，沒有 `localhost` 時加一句警告。
- **改 profile 不寫 hosts 檔**：照 user 的「啟用時才匯出」。改了生效中的 profile 之後，statusbar 的狀態變成 `(differs)`，再按一次
  `Enter` 就寫進去。
- **生效中的 profile 不能刪**（tdp M6 停用）：刪了 hosts 檔裡的東西就沒有 profile 對得上。
- **位址是一欄 text input，用空白隔開**：位址通常一兩個（IPv4 與 IPv6）；一列一個位址的話表單要能長出新的列，components 沒有這種欄位。
- **sudo 在 terminal popup 裡問密碼**：不自己做密碼框收 sudo 的密碼（hostu 不碰密碼）；`sudo -n` 先試，剛打過密碼時不會閃一個 popup。
  Windows 沒有 PTY（terminu-go 的終端機只有 Unix），UAC 本來就是系統的視窗。
- **最低尺寸 80×24**（tdp L1 可以調低）：兩個 panel 與每個 popup 在 24 列都放得下。
- **statusbar 的欄位不動**（tdp L2）：路徑最多佔三分之一寬，太長從前面截，後面的狀態與 `active` 一定看得到，寬度只跟終端機的寬有關；
  狀態（`(in sync)`、`(differs)`、`(can't read)`）補空白到最寬的那一個，名稱長短不定的生效 profile 放最後。
- **長欄標 `Shrink`**（tdp v0.8.2 layout/panel）：profile 的名稱、domain、位址是長值，分剩下的寬度、太長截短；domain 數是短值，照最寬的
  值不縮。窄的時候先收位址那一欄、名稱與 domain 永遠留著。
- **profile 名稱不分大小寫不能重複**、domain 在同一個 profile 裡不分大小寫不能重複：hosts 檔的名稱不分大小寫。
- **`profiles.yaml` 是資料**（tdp E3，hostu 回報的 HU17）：使用者寫給 app 的清單算資料，放 `~/.hostu`；設定只留 app 自己的選項。
  hostu 還沒發過版，沒有寫搬家。
- **讀不了的檔搬開之後照第一次打開**（tdp E3-18）：hostu 沒有 profile 的時候本來就把 hosts 檔讀成 `default`，搬開之後也一樣，生效的
  profile 一開始就對得上 hosts 檔；hosts 檔也讀不到時不建，第一次存檔才有檔。`config.yaml` 第一次打開不建（hostu 沒有改設定的地方），
  搬開之後也不建，用預設值。
- **hosts 檔讀不到時開 error popup**：hosts 檔不是 hostu 自己的檔（tdp E3：照常開著）；hostu 沒有顯示 hosts 檔的 panel，只有 statusbar
  的 `(can't read)`，所以照 tdp F5 開 error popup 寫原因（自己會記進 App Log）。terminu 判過（2026-10-20）：沒有 panel 時照 F5。
- **`input_tab_width` 留著**：preview 的 viewer 用到（tdp E3-11：只有 App Log 的 app 才不用）。preview 的內容現在不會有 Tab，照樣留：
  key 跟著功能走，不跟著內容走（terminu 判過，2026-10-20）。
- **App Log 一筆大寫開頭**：toast 會照同一句記進去，toast 是大寫開頭的句子，hostu 自己記的也一樣（terminu 判過，2026-10-20）。

## 已否決，不要重提

- 用 textarea 或 `$EDITOR` 直接編 hosts 檔：user 要的是 TUI 一筆一筆加（第一版的 hostu 就這樣做，被否決，2026-10-19）。
- 只管 hosts 檔裡的一塊（managed block）：見上面。
- 改了生效中的 profile 就自動寫 hosts 檔：user 要的是啟用時才匯出；每改一次都要 sudo 也很煩。

## 已知的牆與未做

- Windows 的寫入（`gsudo`／`sudo` 加 PowerShell）只有單元測試換掉了 exec，沒有在 Windows 上跑過；`make check` 照 Windows vet 一次。
- 不監看 hosts 檔：啟動時與寫完之後才讀，別的程式改了要重開 hostu 或再寫一次才看得到。
- macOS 的 DNS 快取通常會自己跟上 hosts 檔；跟不上時要自己清（`sudo killall -HUP mDNSResponder`），hostu 不做。

## 偏離 tdp

（沒有。）

## 設計文件導讀

- [`docs/spec.md`](spec.md)：需求、版面、每個動作、資料格式、里程碑。

## 建置與開發

- `make build`、`make check`（加上 `cross`：照 Windows 與 Linux 再 vet 一次）。
- `make sandbox`：在暫存目錄裡跑 hostu（`tools/sandbox.sh`）：hosts 檔是一份示範檔，家目錄、設定、資料都是新的，每次都像第一次啟動。
  開發、judge 都用它，不碰真的 hosts 檔與家目錄。不複製 `/etc/hosts`：畫面會進 judge 的證據與錄影，`/etc/hosts` 可能有公司的機器名。
- 測試（`internal/ui`）不用終端機：送鍵、讀畫面；每個測試在 `/tmp` 下開一個短路徑的暫存目錄放 hosts 檔與設定（路徑短，confirm 與
  toast 裡的路徑才不折行）。tdp L4 跑五種尺寸、tdp E5 三種 icon 寬度、tdp L1 比最低尺寸小，`TestSplashIsDocsIcon` 守住 `internal/ui/icon.svg` 跟
  `docs/icon.svg` 一樣（tdp E6）。sudo 要密碼的那條路用 `sh` 代替 sudo 測。
- `tdp_test.go` 是 terminu-go 的 tdptest：`TestTDP` 在沙箱裡照第一次啟動跑家族的走法（每種尺寸、icon 寬度，每個 panel 開關 `Space`、
  `!`、`?`），`TestTDPWithTrickyValues` 用 tdptest 的怪值當 profile 名稱、hosts 檔的檔名帶 Tab 與顏色碼。hosts 檔一律指到沙箱裡的檔
  （`HOSTU__HOSTS_FILE`），不讀系統的。`MoveAside` 的 `Esc` 寫到 stderr 的那句由 terminu-go 給，harness 拿不到，只測離開與檔案沒動。
- `make gif`：`.local/demos/demo.tape`，用同目錄的示範 hosts 檔與 `profiles.yaml`，複製到 `/tmp` 的新目錄再跑；不拿真的 `/etc/hosts`
  錄（它可能有公司的 domain）。
- `tools/` 是開發用的腳本，不算 app 本身（tdp project 的目錄）。

## 發布

還沒發過版。照 tdp project 的「發布」：`make check`、CHANGELOG 的 `## Unreleased` 改成 `## vX.Y.Z — tdp v0.9.2`、打 tag、push；
release.yml 在 ubuntu、macOS、Windows 跑測試，goreleaser 發 Homebrew tap 與 scoop bucket。`install.ps1`、`uninstall.ps1` 是 hostu 自己寫的
（範本沒有）。
