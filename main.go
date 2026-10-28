package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/manifoldco/promptui"
)

// version puede reemplazarse al compilar con -ldflags.
var version = "1.0.8"

func main() {
	if len(os.Args) == 2 && os.Args[1] == "--version" {
		fmt.Printf("play-tool %s %s %s/%s\n", version, runtime.Version(), runtime.GOOS, runtime.GOARCH)
		return
	}
	if len(os.Args) == 2 && (os.Args[1] == "--help" || os.Args[1] == "-h") {
		printHelp()
		return
	}

	cfg, err := loadConfig()
	if err != nil {
		fmt.Println("❌ Error:", err)
		os.Exit(1)
	}

	configLabel := cfg.ConfigPath
	if configLabel == "" {
		configLabel = "automática (sin play-tool.json)"
	}
	fmt.Printf("⚙️  Config:   %s\n📱 App:      %s\n📂 Proyecto: %s\n", configLabel, cfg.Pkg, cfg.GradleRoot)

	if len(os.Args) > 1 {
		if err := runCI(cfg, os.Args[1:]); err != nil {
			fmt.Println("❌ Error:", err)
			os.Exit(1)
		}
		return
	}

	runInteractive(cfg)
}

func runInteractive(cfg Config) {
	for {
		idx, err := selectOption("🧭 Menú principal", []string{
			"🔨 Build e instalación",
			"🔍 Información y verificación",
			"☁️  Google Play",
			"🚪 Salir",
		})
		if err != nil {
			return
		}

		switch idx {
		case 0:
			buildInstallMenu(cfg)
		case 1:
			infoMenu(cfg)
		case 2:
			googlePlayMenu(cfg)
		case 3:
			return
		}
	}
}

func buildInstallMenu(cfg Config) {
	items := []string{
		"🛠️  assembleDebug",
		"📲 installDebug (Gradle)",
		"📲 adbInstallDebug (APK existente)",
		"🛠️  assembleRelease",
		"📲 installRelease (Gradle)",
		"📲 adbInstallRelease (APK existente)",
		"📦 bundleRelease",
		"🧹 Build AAB limpio (clean + bundleRelease)",
		"↩️  Volver",
	}
	for {
		idx, err := selectOption("🔨 Build e instalación", items)
		if err != nil || idx == len(items)-1 {
			return
		}
		commands := [][]string{
			{"assembleDebug"},
			{"installDebug"},
			{"adbInstallDebug"},
			{"assembleRelease"},
			{"installRelease"},
			{"adbInstallRelease"},
			{"bundleRelease"},
			{"build"},
		}
		report(runCI(cfg, commands[idx]))
	}
}

func infoMenu(cfg Config) {
	items := []string{
		"📄 info assembleDebug",
		"📄 info assembleRelease",
		"📄 info bundleRelease",
		"🔏 verify APK debug",
		"🔏 verify APK release",
		"🔏 verify bundle",
		"🤖 manifest min-sdk debug",
		"🎯 manifest target-sdk debug",
		"🤖 manifest min-sdk release",
		"🎯 manifest target-sdk release",
		"↩️  Volver",
	}
	commands := [][]string{
		{"info", "assembleDebug"},
		{"info", "assembleRelease"},
		{"info", "bundleRelease"},
		{"verify", "apk", "debug"},
		{"verify", "apk", "release"},
		{"verify", "bundle"},
		{"manifest", "min-sdk", "debug"},
		{"manifest", "target-sdk", "debug"},
		{"manifest", "min-sdk", "release"},
		{"manifest", "target-sdk", "release"},
	}
	for {
		idx, err := selectOption("🔍 Información y verificación", items)
		if err != nil || idx == len(items)-1 {
			return
		}
		report(runCI(cfg, commands[idx]))
	}
}

func googlePlayMenu(cfg Config) {
	items := []string{
		"🔎 Ver versionCode publicado",
		"📤 Subir AAB ya compilado",
		"🚀 Compilar y subir",
		"🔢 Deploy (bump + build + upload)",
		"↩️  Volver",
	}
	for {
		idx, err := selectOption("☁️  Google Play", items)
		if err != nil || idx == len(items)-1 {
			return
		}

		switch idx {
		case 0:
			report(showVersion(cfg))
		case 1:
			if confirm(fmt.Sprintf("📤 ¿Subir %s a %s?", cfg.AAB, cfg.Track)) {
				report(uploadAAB(cfg))
			}
		case 2:
			if err := build(cfg); err != nil {
				report(err)
				continue
			}
			if confirm(fmt.Sprintf("🚀 ¿Subir el AAB recién compilado a %s?", cfg.Track)) {
				report(uploadAAB(cfg))
			}
		case 3:
			if confirm(fmt.Sprintf("🔢 ¿Incrementar versionCode, compilar y publicar en %s?", cfg.Track)) {
				report(runCI(cfg, []string{"deploy"}))
			}
		}
	}
}

func printHelp() {
	fmt.Printf(`play-tool %s

Uso:
  play-tool                 Abre el menú interactivo
  play-tool show            Muestra las versiones publicadas
  play-tool assembleDebug
  play-tool installDebug
  play-tool assembleRelease
  play-tool installRelease
  play-tool bundleRelease
  play-tool adbInstallDebug
  play-tool adbInstallRelease
  play-tool info assembleDebug|assembleRelease|bundleRelease
  play-tool build           Compila el AAB release
  play-tool build apk debug|release
  play-tool build bundle release
  play-tool install gradle release
  play-tool install apk debug|release
  play-tool verify apk debug|release
  play-tool verify bundle
  play-tool manifest min-sdk|target-sdk debug|release
  play-tool upload          Sube el AAB configurado
  play-tool deploy          Actualiza versionCode, compila y sube
  play-tool --version       Muestra la versión de play-tool, Go y la plataforma
  play-tool --help          Muestra esta ayuda

Configuración:
  play-tool.json es opcional. Reconoce Android nativo y React Native.
  package se detecta desde applicationId en build.gradle.kts o build.gradle.
  En React Native, el wrapper se ejecuta desde android/ y el JSON puede estar
  en la raíz que contiene package.json.
  aab usa la ruta release estándar y PLAY_TOOL_KEY indica la clave de Google Play.
  Los APK usan las rutas estándar bajo app/build/outputs/apk; puedes sobrescribirlas
  con apkDebug y apkRelease. Las herramientas Android se buscan en el SDK o PATH;
  keytool se busca en JAVA_HOME o PATH.
  info bundleRelease también necesita BUNDLETOOL_JAR apuntando a bundletool-all.jar.
  El JSON puede sobrescribir esos valores; sus rutas relativas parten de su carpeta.
`, version)
}

func selectOption(label string, items []string) (int, error) {
	p := promptui.Select{
		Label: label,
		Items: items,
		Size:  len(items),
	}
	idx, _, err := p.Run()
	return idx, err
}

func confirm(question string) bool {
	idx, err := selectOption(question, []string{"Sí", "No"})
	return err == nil && idx == 0
}

// build ejecuta Gradle en la carpeta del proyecto definida en play-tool.json.
func build(cfg Config) error {
	fmt.Println("🛠️  Compilando AAB release...")
	return runGradleTasks(cfg, "clean", "bundleRelease")
}

func runGradleTasks(cfg Config, tasks ...string) error {
	gradle := filepath.Join(cfg.GradleRoot, "gradlew")
	if runtime.GOOS == "windows" {
		gradle += ".bat"
	}

	cmd := exec.Command(gradle, tasks...)
	cmd.Dir = cfg.GradleRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func report(err error) {
	if err != nil {
		fmt.Println("❌ Error:", err)
		return
	}
	fmt.Println("✅ Listo.")
}
