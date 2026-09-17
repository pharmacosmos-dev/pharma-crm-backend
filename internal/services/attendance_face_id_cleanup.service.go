package services

import (
	"context"
	"errors"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
	"gorm.io/gorm"
)

const (
	// attendanceUploadDir — /upload/file endpointi fayllarni saqlaydigan papka.
	attendanceUploadDir = "./app/uploads"
	attendanceFaceIdTrashDir = ".attendance-face-id-trash"
	attendanceFaceIdCleanupLock = "attendance_face_id_cleanup"
)


const attendanceFaceIdNamesInUseSQL = `
	SELECT DISTINCT n.name
	FROM unnest(?::text[]) AS n(name)
	JOIN attendance_logs al
	  ON al.face_id_url IS NOT NULL
	 AND regexp_replace(al.face_id_url, '^.*/', '') = n.name
	WHERE NOT (al.id = ANY(COALESCE(?::uuid[], '{}')))
`

// CleanupOldAttendanceFaceIds — Toshkent vaqti bo'yicha oxirgi keepDays kundan eski
// check-in/check-out yozuvlarining face_id_url maydonini NULL qiladi va shu yozuvlar
// ko'rsatgan fayllarni ./app/uploads dan o'chiradi. keepDays=2 bo'lsa bugungi va
// kechagi kun rasmlari qoladi, keepDays=0 bo'lsa hammasi tozalanadi. Uploads papkasi
// skan qilinmaydi: faqat face_id_url'da yozilgan, xavfsiz nomli fayllarga tegiladi.
//
// PostgreSQL tranzaksiyasi fayl tizimini qamramaydi, shuning uchun o'chirish ikki fazali:
//  1. Tranzaksiya ichida yozuvlar FOR UPDATE bilan olinadi va fayllar trash papkaga
//     ko'chiriladi (rename — qaytariladigan amal). Fayli trash'ga ko'chgan yoki diskda
//     umuman yo'q yozuvlargina NULL qilinadi, keyin COMMIT.
//  2. Faqat COMMIT'dan keyin trash'dagi fayllar butunlay o'chiriladi.
//
// COMMIT o'tmasa yoki natijasi noma'lum bo'lsa (ulanish uzilgan), trash'dagi fayllar
// bazaning haqiqiy holatiga qarab joyiga qaytariladi yoki o'chiriladi. Butun jarayon
// sessiya darajasidagi advisory lock ostida: parallel chaqiruv ConflictError oladi.
func (s *Services) CleanupOldAttendanceFaceIds(ctx context.Context, keepDays int) (*domain.AttendanceFaceIdCleanupResult, error) {
	// os.Root barcha fayl amallarini uploads ichida ushlaydi: "..", absolyut yo'l yoki
	// tashqariga qaragan symlink orqali chiqib bo'lmaydi (filepath.Join buni bermaydi).
	root, err := os.OpenRoot(attendanceUploadDir)
	if err != nil {
		s.log.Errorf("could not open uploads dir: %v", err)
		return nil, domain.InternalServerError
	}
	defer root.Close()

	if err := root.MkdirAll(attendanceFaceIdTrashDir, 0o755); err != nil {
		s.log.Errorf("could not create attendance face id trash dir: %v", err)
		return nil, domain.InternalServerError
	}

	result := &domain.AttendanceFaceIdCleanupResult{}
	err = s.db.WithContext(ctx).Connection(func(conn *gorm.DB) error {
		var locked bool
		if err := conn.Raw(`SELECT pg_try_advisory_lock(hashtext(?))`, attendanceFaceIdCleanupLock).Row().Scan(&locked); err != nil {
			s.log.Errorf("could not acquire attendance face id cleanup lock: %v", err)
			return domain.InternalServerError
		}
		if !locked {
			return domain.ConflictError
		}
		defer func() {
			// So'rov konteksti tugagan bo'lsa ham bo'shatiladi, aks holda ulanish pool'ga
			// lock bilan qaytib, keyingi chaqiruvlar 409 olaverardi.
			if err := conn.WithContext(context.Background()).Exec(`SELECT pg_advisory_unlock(hashtext(?))`, attendanceFaceIdCleanupLock).Error; err != nil {
				s.log.Errorf("could not release attendance face id cleanup lock: %v", err)
			}
		}()

		// Oldingi chaqiruvdan (crash, COMMIT yoki o'chirish xatosi) trash'da qolgan fayllar.
		if leftovers := s.attendanceFaceIdTrashNames(root); len(leftovers) > 0 {
			s.reconcileAttendanceFaceIdTrash(ctx, conn, root, leftovers)
		}

		moved, err := s.clearOldAttendanceFaceIds(ctx, conn, root, keepDays, result)
		if err != nil {
			if len(moved) > 0 {
				// ctx muddati o'tgan bo'lishi mumkin — qaytarish uchun yangi kontekst.
				reconcileCtx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
				defer cancel()
				s.reconcileAttendanceFaceIdTrash(reconcileCtx, conn, root, moved)
			}
			return err
		}

		for _, name := range moved {
			removeErr := root.Remove(filepath.Join(attendanceFaceIdTrashDir, name))
			if removeErr != nil && !errors.Is(removeErr, fs.ErrNotExist) {
				result.DeleteErrorCount++
				s.log.Errorf("could not delete attendance photo %s, left in trash: %v", name, removeErr)
				continue
			}
			result.DeletedFileCount++
		}
		return nil
	})
	if err != nil {
		var domainErr *domain.Error
		if !errors.As(err, &domainErr) {
			s.log.Errorf("could not cleanup old attendance face ids: %v", err)
			return nil, domain.InternalServerError
		}
		return nil, err
	}

	s.log.Infof("attendance face id cleanup keep_days=%d: %+v", keepDays, *result)
	return result, nil
}


func (s *Services) clearOldAttendanceFaceIds(ctx context.Context, conn *gorm.DB, root *os.Root, keepDays int, result *domain.AttendanceFaceIdCleanupResult) (moved []string, err error) {
	tx := conn.WithContext(ctx).Begin()
	if tx.Error != nil {
		s.log.Errorf("could not begin attendance face id cleanup transaction: %v", tx.Error)
		return nil, domain.InternalServerError
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	var logs []struct {
		Id        string
		FaceIdUrl string
	}
	if err = tx.Raw(`
		SELECT id, face_id_url
		FROM attendance_logs
		WHERE face_id_url IS NOT NULL
		  AND event_at < (((now() AT TIME ZONE 'Asia/Tashkent')::date - ?::int + 1)::timestamp AT TIME ZONE 'Asia/Tashkent')
		ORDER BY id
		FOR UPDATE
	`, keepDays).Scan(&logs).Error; err != nil {
		s.log.Errorf("could not select old attendance face ids: %v", err)
		return nil, domain.InternalServerError
	}
	result.SelectedCount = len(logs)

	var names, clearableIds []string
	idsByName := make(map[string][]string)
	for _, l := range logs {
		name, ok := attendanceFaceIdFileName(l.FaceIdUrl)
		if !ok {
			result.SkippedInvalidPathCount++
			s.log.Warnf("attendance log %s has unsafe face_id_url %q, skipped", l.Id, l.FaceIdUrl)
			continue
		}
		if _, seen := idsByName[name]; !seen {
			names = append(names, name)
		}
		idsByName[name] = append(idsByName[name], l.Id)
		clearableIds = append(clearableIds, l.Id)
	}

	// keep_days ichidagi yozuv ham shu faylni ko'rsatsa, fayl diskda qoladi.
	inUse := make(map[string]bool)
	if len(names) > 0 {
		var used []string
		if err = tx.Raw(attendanceFaceIdNamesInUseSQL, pq.Array(names), pq.Array(clearableIds)).Scan(&used).Error; err != nil {
			s.log.Errorf("could not check attendance face ids in use: %v", err)
			return nil, domain.InternalServerError
		}
		for _, name := range used {
			inUse[name] = true
		}
	}

	// Yozuv faqat fayli trash'ga ko'chgan, diskda yo'q yoki boshqa yozuvga kerak bo'lsa
	// NULL qilinadi. Ko'chirib bo'lmagan fayl yozuvi o'zgarmaydi — aks holda fayl bazada
	// hech qayerda yozilmagan, mahsulot rasmlaridan ajratib bo'lmaydigan yetimga aylanadi.
	var clearIds []string
	for _, name := range names {
		if inUse[name] {
			result.StillInUseFileCount++
			clearIds = append(clearIds, idsByName[name]...)
			continue
		}

		info, statErr := root.Lstat(name)
		switch {
		case errors.Is(statErr, fs.ErrNotExist):
			result.FileNotFoundCount++
		case statErr != nil:
			result.DeleteErrorCount++
			s.log.Errorf("could not stat attendance photo %s: %v", name, statErr)
			continue
		case !info.Mode().IsRegular():
			result.SkippedInvalidPathCount += len(idsByName[name])
			s.log.Warnf("attendance photo %s is not a regular file, skipped", name)
			continue
		default:
			if mvErr := root.Rename(name, filepath.Join(attendanceFaceIdTrashDir, name)); mvErr != nil {
				result.DeleteErrorCount++
				s.log.Errorf("could not move attendance photo %s to trash: %v", name, mvErr)
				continue
			}
			moved = append(moved, name)
		}
		clearIds = append(clearIds, idsByName[name]...)
	}

	if len(clearIds) > 0 {
		res := tx.Exec(`UPDATE attendance_logs SET face_id_url = NULL WHERE id = ANY(?::uuid[])`, pq.Array(clearIds))
		if err = res.Error; err != nil {
			s.log.Errorf("could not clear old attendance face ids: %v", err)
			return moved, domain.InternalServerError
		}
		result.UpdatedCount = int(res.RowsAffected)
	}

	if err = tx.Commit().Error; err != nil {
		s.log.Errorf("could not commit attendance face id cleanup: %v", err)
		return moved, domain.InternalServerError
	}

	return moved, nil
}


func (s *Services) attendanceFaceIdTrashNames(root *os.Root) []string {
	entries, err := fs.ReadDir(root.FS(), attendanceFaceIdTrashDir)
	if err != nil {
		s.log.Errorf("could not read attendance face id trash dir: %v", err)
		return nil
	}

	var names []string
	for _, e := range entries {
		if name, ok := attendanceFaceIdFileName(e.Name()); ok && name == e.Name() && e.Type().IsRegular() {
			names = append(names, name)
		}
	}
	return names
}

// reconcileAttendanceFaceIdTrash — trash'dagi fayllarni bazaning COMMIT qilingan holatiga
// moslaydi: biror yozuv hali ko'rsatayotgan fayl uploads'ga qaytariladi (tozalash rollback
// bo'lgan), hech kim ko'rsatmayotgani o'chiriladi (COMMIT o'tgan, o'chirish tugamagan).
// Faqat advisory lock ostida chaqiriladi, aks holda parallel tozalash hozirgina trash'ga
// ko'chirgan (hali COMMIT qilinmagan) faylni qaytarib yuborishi mumkin. Baza javob
// bermasa fayllarga tegilmaydi — ular trash'da keyingi chaqiruvni kutadi.
func (s *Services) reconcileAttendanceFaceIdTrash(ctx context.Context, conn *gorm.DB, root *os.Root, names []string) {
	var used []string
	if err := conn.WithContext(ctx).Raw(attendanceFaceIdNamesInUseSQL, pq.Array(names), pq.Array([]string{})).Scan(&used).Error; err != nil {
		s.log.Errorf("could not reconcile attendance face id trash, %d files left in %s: %v", len(names), attendanceFaceIdTrashDir, err)
		return
	}
	inUse := make(map[string]bool, len(used))
	for _, name := range used {
		inUse[name] = true
	}

	var restored, deleted int
	for _, name := range names {
		trashPath := filepath.Join(attendanceFaceIdTrashDir, name)
		if !inUse[name] {
			if err := root.Remove(trashPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
				s.log.Errorf("could not delete attendance photo %s from trash: %v", name, err)
				continue
			}
			deleted++
			continue
		}

		if _, err := root.Lstat(name); err == nil {
			s.log.Errorf("could not restore attendance photo %s: file already exists in uploads", name)
			continue
		}
		if err := root.Rename(trashPath, name); err != nil && !errors.Is(err, fs.ErrNotExist) {
			s.log.Errorf("could not restore attendance photo %s from trash: %v", name, err)
			continue
		}
		restored++
	}

	s.log.Warnf("attendance face id trash reconciled: restored=%d deleted=%d", restored, deleted)
}

// attendanceFaceIdFileName — face_id_url qiymatidan ./app/uploads ichidagi fayl nomini
// ajratadi. Loyiha konvensiyasi: /upload/file faylni uploads'ga subpapkasiz
// "<uuid>.<jpg|jpeg|png>" nomi bilan saqlaydi va face_id_url'ga aynan shu nom yoziladi.
// Qo'shimcha ravishda "/upload/<nom>", "/v1/upload/<nom>", "/uploads/<nom>" va shunday
// yo'lli to'liq URL qabul qilinadi. Qolgan hammasi — subpapka ("attendance/x.png"), "..",
// absolyut yo'l, query, uuid bo'lmagan nom yoki boshqa kengaytma (masalan uploads'dagi
// DejaVuSans.ttf, categories.json) — rad etiladi.
func attendanceFaceIdFileName(raw string) (string, bool) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Opaque != "" || u.RawQuery != "" || u.Fragment != "" {
		return "", false
	}
	if u.Scheme != "" && u.Scheme != "http" && u.Scheme != "https" {
		return "", false
	}

	name := u.Path
	for _, prefix := range []string{"/v1/upload/", "/upload/", "/uploads/"} {
		if strings.HasPrefix(name, prefix) {
			name = strings.TrimPrefix(name, prefix)
			break
		}
	}

	// Faqat bitta yo'l elementi: "../x", "a/b", "/abs", `a\b` o'tmaydi.
	if !filepath.IsLocal(name) || strings.ContainsAny(name, `/\`) {
		return "", false
	}

	ext := filepath.Ext(name)
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
		return "", false
	}
	if uuid.Validate(strings.TrimSuffix(name, ext)) != nil {
		return "", false
	}

	return name, true
}
