# hostu

由 analysis 從 README 整理（2026-10-26），不是使用者的原話。

在終端機裡用 profile 切換 hosts 檔。

把幾組 hosts 設定存成 profile —— 公司、家裡、客戶現場 —— 要哪一組，按一個鍵就寫進 hosts 檔。每個 profile 是一串 domain，
每個 domain 指到一個或好幾個位址（IPv4 與 IPv6）；在表單裡一筆一筆加、改，不用自己編 hosts 檔。隨時可以把 hosts 檔現在的內容
讀成一個新的 profile。

給誰用：要在幾組 hosts 設定之間切換的人。

平台：macOS、Linux、Windows。

不能發生的事：

- 原本的 hosts 檔不見：第一次打開時，hosts 檔現在的內容讀成叫 `default` 的 profile；第一次寫 hosts 檔之前，先把它原樣留一份。
- 沒說就寫 hosts 檔：改 profile 只改 hostu 自己的檔；啟用 profile 時才寫 hosts 檔，寫之前先問。
- 要用管理員身分跑 hostu：用自己的帳號跑，只有寫 hosts 檔的時候才要 sudo（Windows 是 UAC）。
- 讀不了的設定或 profile 檔照樣被用、被蓋掉：先問，改名留著，或離開。

限制：

- hostu 照 profile 寫整個 hosts 檔：自己在 hosts 檔寫的註解與內容不會留在裡面。
- hosts 檔裡不是「位址加名稱」的行不會讀進 profile。
- hostu 不監看 hosts 檔：啟動時與寫完之後才讀。


## 使用者後來改的

- 2026-10-27，看到生效的勾畫在名稱那一格裡：「左邊 side bar 對其很有問題，應該要有預留一個 checked 的欄位，這樣才會對其」
- 2026-10-27，寫 hosts 檔要 sudo 密碼時開了撐滿畫面的終端機：「跳 sudo password 的視窗也太大了吧... 有需要嗎?」「這沒辦法套用 PIN
  的 popup 表示方式嗎?」；選了「改用 password popup 收密碼」
