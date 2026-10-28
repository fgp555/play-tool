package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"google.golang.org/api/androidpublisher/v3"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

// type Config struct {
// 	Pkg   string
// 	Track string
// 	AAB   string
// 	Key   string
// }

func newService(key string) (*androidpublisher.Service, error) {
	if key == "" {
		return nil, fmt.Errorf("falta el campo obligatorio \"key\" en %s", configName)
	}
	info, err := os.Stat(key)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no existe la clave de Google Play en %s; revisa el campo \"key\" de %s: %w", key, configName, err)
		}
		return nil, fmt.Errorf("no se pudo acceder a la clave de Google Play en %s; revisa el campo \"key\" de %s: %w", key, configName, err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("la ruta configurada en el campo \"key\" es una carpeta, no un archivo: %s", key)
	}

	svc, err := androidpublisher.NewService(context.Background(), option.WithCredentialsFile(key))
	if err != nil {
		return nil, fmt.Errorf("no se pudo cargar la clave de Google Play %s: %w", key, err)
	}
	return svc, nil
}

// showVersion muestra los versionCode publicados en la pista. No cambia nada.
func showVersion(cfg Config) error {
	fmt.Println("🔐 Validando credenciales de Google Play...")
	svc, err := newService(cfg.Key)
	if err != nil {
		return err
	}

	fmt.Printf("🔎 Consultando Google Play (%s)...\n", cfg.Track)
	edit, err := svc.Edits.Insert(cfg.Pkg, &androidpublisher.AppEdit{}).Do()
	if err != nil {
		return err
	}
	// Sin Commit, la edit se descarta al cerrarla.
	defer svc.Edits.Delete(cfg.Pkg, edit.Id).Do()

	t, err := svc.Edits.Tracks.Get(cfg.Pkg, edit.Id, cfg.Track).Do()
	if err != nil {
		return err
	}

	var codes []int64
	var max int64
	for _, r := range t.Releases {
		for _, c := range r.VersionCodes {
			codes = append(codes, c)
			if c > max {
				max = c
			}
		}
	}

	fmt.Printf("🎯 Pista:           %s\n", cfg.Track)
	fmt.Printf("📋 Versiones:       %v\n", codes)
	fmt.Printf("🏷️  versionCode max: %d\n", max)
	return nil
}

// uploadAAB sube el AAB ya compilado y lo publica en la pista.
func uploadAAB(cfg Config) error {
	f, err := os.Open(cfg.AAB)
	if err != nil {
		return fmt.Errorf("no se pudo abrir el AAB en %s; revisa el campo \"aab\" de %s: %w", cfg.AAB, configName, err)
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return fmt.Errorf("no se pudo obtener el tamaño del AAB %s: %w", cfg.AAB, err)
	}
	fileSize := info.Size()

	fmt.Println("🔐 Validando credenciales de Google Play...")
	svc, err := newService(cfg.Key)
	if err != nil {
		return err
	}

	fmt.Println("🧾 Preparando la carga...")
	edit, err := svc.Edits.Insert(cfg.Pkg, &androidpublisher.AppEdit{}).Do()
	if err != nil {
		return err
	}

	committed := false
	defer func() {
		if !committed {
			svc.Edits.Delete(cfg.Pkg, edit.Id).Do()
		}
	}()

	fmt.Printf("🔎 Revisando la pista %q...\n", cfg.Track)
	t, err := svc.Edits.Tracks.Get(cfg.Pkg, edit.Id, cfg.Track).Do()
	if err != nil {
		return err
	}
	var maxPublished int64
	for _, r := range t.Releases {
		for _, c := range r.VersionCodes {
			if c > maxPublished {
				maxPublished = c
			}
		}
	}

	fmt.Printf("📦 Archivo: %s (%s)\n", cfg.AAB, formatBytes(fileSize))
	printUploadProgress(0, fileSize)
	bundle, err := svc.Edits.Bundles.Upload(cfg.Pkg, edit.Id).
		ResumableMedia(context.Background(), f, fileSize, "application/octet-stream").
		ProgressUpdater(googleapi.ProgressUpdater(printUploadProgress)).
		Do()
	if err != nil {
		fmt.Println()
		return fmt.Errorf("no se pudo subir el AAB: %w", err)
	}
	printUploadProgress(fileSize, fileSize)
	fmt.Println()

	if bundle.VersionCode <= maxPublished {
		return fmt.Errorf("el versionCode del AAB (%d) debe ser mayor que el publicado (%d); cambia versionCode y recompila",
			bundle.VersionCode, maxPublished)
	}

	fmt.Printf("🚀 Publicando versionCode %d en %q...\n", bundle.VersionCode, cfg.Track)
	_, err = svc.Edits.Tracks.Update(cfg.Pkg, edit.Id, cfg.Track, &androidpublisher.Track{
		Track: cfg.Track,
		Releases: []*androidpublisher.TrackRelease{
			{
				VersionCodes: []int64{bundle.VersionCode},
				Status:       "completed",
			},
		},
	}).Do()
	if err != nil {
		return fmt.Errorf("no se pudo asignar el release: %w", err)
	}

	fmt.Println("⏳ Confirmando publicación...")
	if _, err := svc.Edits.Commit(cfg.Pkg, edit.Id).Do(); err != nil {
		return fmt.Errorf("no se pudo confirmar la edit: %w", err)
	}
	committed = true

	fmt.Printf("✅ Subido versionCode %d a la pista %q\n", bundle.VersionCode, cfg.Track)
	return nil
}

func printUploadProgress(current, total int64) {
	if total <= 0 {
		fmt.Print("\r⬆️  Subiendo AAB...")
		return
	}
	if current < 0 {
		current = 0
	} else if current > total {
		current = total
	}
	percent := int(float64(current) / float64(total) * 100)
	const width = 24
	filled := percent * width / 100
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	fmt.Printf("\r⬆️  Subiendo AAB [%s] %3d%% (%s / %s)", bar, percent, formatBytes(current), formatBytes(total))
}

func formatBytes(size int64) string {
	const mb = 1024 * 1024
	if size >= mb {
		return fmt.Sprintf("%.1f MB", float64(size)/mb)
	}
	return fmt.Sprintf("%.1f KB", float64(size)/1024)
}

func publishedMax(cfg Config) (int64, error) {
	svc, err := newService(cfg.Key)
	if err != nil {
		return 0, err
	}

	edit, err := svc.Edits.Insert(cfg.Pkg, &androidpublisher.AppEdit{}).Do()
	if err != nil {
		return 0, err
	}
	defer svc.Edits.Delete(cfg.Pkg, edit.Id).Do()

	t, err := svc.Edits.Tracks.Get(cfg.Pkg, edit.Id, cfg.Track).Do()
	if err != nil {
		return 0, err
	}

	var max int64
	for _, r := range t.Releases {
		for _, c := range r.VersionCodes {
			if c > max {
				max = c
			}
		}
	}
	return max, nil
}
