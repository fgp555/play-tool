package main

import (
	"fmt"
	"os"
)

// runCI ejecuta comandos sin menú interactivo, también desde GitHub Actions.
func runCI(cfg Config, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("falta un comando; ejecuta play-tool --help")
	}

	switch args[0] {
	case "assembleDebug", "assembleRelease", "bundleRelease", "installDebug", "installRelease":
		if err := requireArgs(args, 1, args[0]); err != nil {
			return err
		}
		fmt.Printf("🛠️  Ejecutando Gradle %s...\n", args[0])
		return runGradleTasks(cfg, args[0])

	case "adbInstallDebug":
		if err := requireArgs(args, 1, args[0]); err != nil {
			return err
		}
		return adbInstall(cfg, "debug")

	case "adbInstallRelease":
		if err := requireArgs(args, 1, args[0]); err != nil {
			return err
		}
		return adbInstall(cfg, "release")

	case "info":
		if len(args) != 2 {
			return fmt.Errorf("uso: play-tool info assembleDebug|assembleRelease|bundleRelease")
		}
		return showArtifactInfo(cfg, args[1])

	case "show":
		if err := requireArgs(args, 1, "show"); err != nil {
			return err
		}
		return showVersion(cfg)

	case "build":
		if len(args) == 1 {
			return build(cfg)
		}
		return buildVariant(cfg, args[1:])

	case "install":
		return installVariant(cfg, args[1:])

	case "verify":
		return verifyArtifact(cfg, args[1:])

	case "manifest":
		return inspectManifest(cfg, args[1:])

	case "upload":
		if err := requireArgs(args, 1, "upload"); err != nil {
			return err
		}
		return uploadAAB(cfg)

	case "deploy":
		if err := requireArgs(args, 1, "deploy"); err != nil {
			return err
		}
		n, err := bumpVersionCode(cfg)
		if err != nil {
			return fmt.Errorf("bump: %w", err)
		}
		fmt.Printf("🔢 versionCode asignado: %d\n", n)

		if err := build(cfg); err != nil {
			return fmt.Errorf("build: %w", err)
		}
		return uploadAAB(cfg)

	default:
		return fmt.Errorf("comando desconocido %q; ejecuta play-tool --help", args[0])
	}
}

func requireArgs(args []string, count int, command string) error {
	if len(args) != count {
		return fmt.Errorf("uso incorrecto de %s; ejecuta play-tool --help", command)
	}
	return nil
}
func buildVariant(cfg Config, args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("uso: play-tool build apk debug|release o play-tool build bundle release")
	}
	var task string
	switch {
	case args[0] == "apk" && args[1] == "debug":
		task = "assembleDebug"
	case args[0] == "apk" && args[1] == "release":
		task = "assembleRelease"
	case args[0] == "bundle" && args[1] == "release":
		task = "bundleRelease"
	default:
		return fmt.Errorf("tipo o variante de build no válida; usa apk debug|release o bundle release")
	}
	fmt.Printf("🛠️  Ejecutando Gradle %s...\n", task)
	return runGradleTasks(cfg, task)
}

func installVariant(cfg Config, args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("uso: play-tool install gradle release o play-tool install apk debug|release")
	}
	if args[0] == "gradle" && args[1] == "release" {
		fmt.Println("📲 Instalando release mediante Gradle...")
		return runGradleTasks(cfg, "installRelease")
	}
	if args[0] != "apk" {
		return fmt.Errorf("tipo de instalación no válida; usa gradle release o apk debug|release")
	}
	return adbInstall(cfg, args[1])
}

func adbInstall(cfg Config, variant string) error {
	apk, err := apkPath(cfg, variant)
	if err != nil {
		return err
	}
	if err := requireFile(apk, "APK"); err != nil {
		return err
	}
	adb, err := resolveAndroidTool("adb")
	if err != nil {
		return err
	}
	fmt.Printf("📲 Instalando APK %s en el dispositivo conectado...\n", variant)
	return runTool(adb, cfg.GradleRoot, "install", "-r", "-d", apk)
}

func verifyArtifact(cfg Config, args []string) error {
	if len(args) == 1 && args[0] == "bundle" {
		if err := requireFile(cfg.AAB, "AAB"); err != nil {
			return err
		}
		keytool, err := resolveKeytool()
		if err != nil {
			return err
		}
		fmt.Println("🔏 Verificando certificado del AAB...")
		return runTool(keytool, cfg.GradleRoot, "-printcert", "-jarfile", cfg.AAB)
	}
	if len(args) != 2 {
		return fmt.Errorf("uso: play-tool verify apk debug|release o play-tool verify bundle")
	}
	switch args[0] {
	case "apk":
		apk, err := apkPath(cfg, args[1])
		if err != nil {
			return err
		}
		if err := requireFile(apk, "APK"); err != nil {
			return err
		}
		apksigner, err := resolveAndroidTool("apksigner")
		if err != nil {
			return err
		}
		fmt.Printf("🔏 Verificando certificado del APK %s...\n", args[1])
		return runTool(apksigner, cfg.GradleRoot, "verify", "--print-certs", apk)

	default:
		return fmt.Errorf("artefacto no válido; usa apk debug|release o bundle")
	}
}

func inspectManifest(cfg Config, args []string) error {
	if len(args) != 2 || (args[0] != "min-sdk" && args[0] != "target-sdk") {
		return fmt.Errorf("uso: play-tool manifest min-sdk|target-sdk debug|release")
	}
	apk, err := apkPath(cfg, args[1])
	if err != nil {
		return err
	}
	if err := requireFile(apk, "APK"); err != nil {
		return err
	}
	apkanalyzer, err := resolveAndroidTool("apkanalyzer")
	if err != nil {
		return err
	}
	fmt.Printf("📄 Leyendo %s del APK %s...\n", args[0], args[1])
	return runTool(apkanalyzer, cfg.GradleRoot, "manifest", args[0], apk)
}

func apkPath(cfg Config, variant string) (string, error) {
	switch variant {
	case "debug":
		return cfg.APKDebug, nil
	case "release":
		return cfg.APKRelease, nil
	default:
		return "", fmt.Errorf("variante no válida %q; usa debug o release", variant)
	}
}

func requireFile(path, description string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("no se encontró el %s en %s: %w", description, path, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("la ruta del %s no es un archivo: %s", description, path)
	}
	return nil
}
