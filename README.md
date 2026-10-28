# play-tool

`play-tool` permite consultar y publicar una app Android en Google Play, tanto nativa como React Native. Detecta el wrapper Gradle para compilar/subir y reconoce la raíz React Native mediante `package.json` y `android/`.

## Instalación en Windows

Desde PowerShell, instala el ejecutable:

```powershell
Set-Location E:\dev\kotlin\com.frankgp.tools\tools\play-tool
go install .
```

Go coloca el ejecutable en `$(go env GOPATH)\bin` (normalmente `%USERPROFILE%\go\bin`). Agrega esa carpeta al `PATH` de Windows para invocar `play-tool` desde cualquier terminal. Después de cambiar el `PATH`, abre una terminal nueva.

## Configuración automática y opcional

`play-tool.json` es opcional. Sin él, `play-tool` detecta `applicationId` en `app/build.gradle.kts` o `app/build.gradle`, usa la pista `internal` y asume la salida AAB estándar `app/build/outputs/bundle/release/app-release.aab`. En React Native, esa ruta se calcula desde la carpeta `android/`.

El comando `deploy` incrementa el `versionCode` numérico en cualquiera de esos dos formatos Gradle antes de compilar y subir.

La clave de Google Play se indica con la variable de entorno `PLAY_TOOL_KEY` o con el campo `key` de `play-tool.json`. La variable de entorno tiene prioridad. En PowerShell, para definirla en la terminal actual:

```powershell
$env:PLAY_TOOL_KEY = "E:\dev\android\play-service-account.json"
```

Si prefieres configurar la clave por proyecto, crea un `play-tool.json` mínimo en la raíz Gradle o en la raíz React Native (donde está `package.json`):

```json
{
  "key": "E:/dev/android/play-service-account.json"
}
```

El JSON puede sobrescribir `package`, `track`, `aab`, `apkDebug`, `apkRelease`, `key` y `gradleRoot`. Sus rutas relativas se resuelven desde la carpeta donde está el JSON; la ruta relativa de `PLAY_TOOL_KEY` se resuelve desde la raíz Gradle. Si `applicationId` no está escrito como literal en `app/build.gradle.kts` o `app/build.gradle`, configura `package` manualmente. Para proyectos con flavors, configura las rutas y el paquete de la variante que publiques. No subas credenciales al repositorio.

Para inspeccionar metadatos de un AAB, instala `bundletool-all.jar` por tu cuenta e indica su ruta en la sesión de PowerShell:

```powershell
$env:BUNDLETOOL_JAR = "E:\tools\bundletool-all.jar"
```

## Uso

Consulta los comandos disponibles sin necesidad de configuración:

```powershell
play-tool --help
```

Consulta la versión sin necesidad de que exista un `play-tool.json` en la carpeta actual:

```powershell
play-tool --version
# play-tool 1.0.8 go1.27.0 windows/amd64
```

Desde cualquier carpeta dentro del proyecto —incluida la raíz— ejecuta el comando sin opciones de configuración:

```powershell
play-tool show
play-tool assembleDebug
play-tool installDebug
play-tool assembleRelease
play-tool installRelease
play-tool bundleRelease
play-tool adbInstallDebug
play-tool adbInstallRelease
play-tool info assembleDebug
play-tool info assembleRelease
play-tool info bundleRelease
play-tool build
play-tool build apk debug
play-tool build apk release
play-tool build bundle release
play-tool install gradle release
play-tool install apk debug
play-tool install apk release
play-tool verify apk debug
play-tool verify apk release
play-tool verify bundle
play-tool manifest min-sdk debug
play-tool manifest target-sdk debug
play-tool upload
play-tool deploy
```

`build` sin argumentos conserva el comportamiento anterior (`clean bundleRelease`). `install gradle release` instala con Gradle; `install apk` instala el APK existente con `adb`. `verify` muestra el certificado del APK/AAB y `manifest` consulta minSdk o targetSdk. Las herramientas se buscan en las ubicaciones estándar de Android SDK/JDK y después en `PATH`.

Los alias `assembleDebug`, `assembleRelease` y `bundleRelease` ejecutan sus tareas Gradle. `installDebug` y `installRelease` instalan mediante Gradle. Los comandos `adbInstall...` instalan un APK ya generado usando `adb install -r -d`. `info` inspecciona el artefacto ya existente, muestra paquete, versiones, minSdk/targetSdk, firma, SHA-1, SHA-256, ruta y tamaño. Para AAB, requiere `BUNDLETOOL_JAR`; muestra el certificado de carga del bundle. `compileSdk` no está incluido en el artefacto final y no se reporta.

Los comandos muestran estados con emojis. `upload` presenta una barra porcentual basada en el progreso de carga por bloques del AAB; después indica cuándo publica y confirma la pista.

Si existe, el programa encuentra el `play-tool.json` más cercano al subir desde el directorio actual y se detiene en la raíz Gradle más cercana. También puedes omitirlo si la detección automática cubre tu proyecto y `PLAY_TOOL_KEY` está configurada.

Al ejecutar `play-tool` sin comando se abre el menú interactivo, agrupado en Build e instalación, Información y verificación, y Google Play. Las opciones de Deploy y publicación piden confirmación antes de ejecutarse.
