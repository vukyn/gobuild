# Audit: root-level static files vs the SPA catch-all

> Lives in `docs/` as `.md`, **not** `docs/todo/*.todo`: `.gitignore` has a bare
> `todo` pattern, so anything under a `todo/` folder is invisible to git and this
> note would never have been committed. `☐` marks open items (platform convention).

```
gobuild — preset `platform-service`: root-level static files vs the SPA catch-all
  Audit ngày 10/08/2026 (từ gardener PR #106). Read-only, CHƯA sửa gì.

  Kết luận: LỖI TIỀM ẨN, không phải đang cháy. Preset sinh ra service ĐÚNG ở thời
  điểm sinh, nhưng shape của nó vỡ ngay khi ai đó thêm bất kỳ file tĩnh nào ở root.
  Đó chính là đường gardener đã đi.

  Todos:
    ☐ Port cách sửa của gardener vào `templates/platform-service/internal/server/server.go.tmpl`.
      HIỆN TRẠNG (dòng 91-108): chỉ `app.Get("/favicon.svg", ...)` rồi `app.Get("/*", renderHomePage)`.
      Preset chỉ ship `ui/public/favicon.svg` và file đó CÓ route → sinh ra là đúng, không hỏng.
      RỦI RO: thêm manifest/icon/sw.js/robots.txt/og-image... là file đó trả index.html
      với status **200** (không phải 404 — nên log nhìn vẫn xanh).
      Hệ quả đã xảy ra thật ở gardener: manifest không parse (Add-to-Home-Screen mất tên +
      icon + standalone; iOS lấy ẢNH CHỤP TRANG làm icon), và `registerSW()` chết với
      `SecurityError: unsupported MIME type ('text/html')` → service worker không bao giờ
      cài → offline chết trên production.
      CÁCH SỬA (copy từ gardener/internal/server/server.go sau PR #106): `app.Get("/:file", ...)`
      resolve từ embedded FS, `c.Next()` khi không có file. KHÔNG dùng danh sách tên cứng như
      rainy làm: chunk workbox có content hash trong tên nên danh sách sẽ mục sau mỗi lần
      build UI và fail im lặng. Loại trừ `index.html` (nó phải qua template renderer để nhận
      APIBaseURL). Route SPA không có file trùng nên rơi xuống catch-all bình thường.
      ⚠️ `.webmanifest` không có trong mime table của Go, VÀ set Content-Type TRƯỚC
      `filesystem.SendFile` là vô ích — SendFile ghi đè, manifest ra `application/octet-stream`
      (browser bỏ qua y như HTML). Phải đọc bytes rồi `c.Send`.
      ⚠️ `/:file` chỉ khớp MỘT tầng. Thư mục ở root (ví dụ `public/sounds/`) cần mount riêng
      kiểu `app.Use("/sounds", filesystem.New(...))` như `/assets`. Xem note của tomatime —
      chỗ đó đang là bug THẬT.
    ☐ Đổi template là phải regenerate golden: `go test -update` rồi commit
      `testdata/golden/platform-service/internal/server/server.go` (bản golden hiện tại
      chứa đúng lỗi này ở dòng 104-108). Golden được `git add -f`.
    ☐ Cân nhắc: có nên đưa luôn VitePWA + manifest + icon vào preset? Hiện preset KHÔNG có
      PWA (không VitePWA, không webmanifest, không apple-touch-icon; `ui/public/` chỉ có
      favicon.svg). Nếu thêm thì phải sửa route TRƯỚC, không thì mọi service sinh mới đều
      ra đời kèm PWA hỏng.

  Trạng thái các repo cùng shape (audit 10/08/2026):
    - rainy: ĐÃ SỬA (danh sách 4 tên cứng, `rainy/internal/server/server.go:91-105`)
    - gardener: ĐÃ SỬA (FS-resolve, PR #106)
    - isme, medioa2: shape y hệt (1 root route + catch-all) nhưng `ui/public/` chỉ có
      favicon.svg và nó có route → hiện KHÔNG hỏng, tiềm ẩn
    - tomatime: ĐANG HỎNG THẬT — xem `tomatime/docs/todo/pwa-root-file-routes.todo`
    - memz: không áp dụng (không embed SPA)
```
