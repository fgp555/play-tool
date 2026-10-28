```sh
cd E:/dev/kotlin/com.frankgp.tools/tools/play-tool
go install .
C:\Users\Frank\go\bin
where.exe play-tool
play-tool --version
play-tool --help

cd E:/dev/kotlin/com.frankgp.tools
cd E:/dev/rn/com.frankgp.app/android

$env:PLAY_TOOL_KEY = "E:\dev\android\play-service-account.json"
echo $env:PLAY_TOOL_KEY

# https://github.com/google/bundletool/releases
$env:BUNDLETOOL_JAR = "E:\dev\android\bundletool-all-1.18.3.jar"
Test-Path $env:BUNDLETOOL_JAR
echo $env:BUNDLETOOL_JAR

play-tool show
play-tool build
play-tool upload
play-tool deploy

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

gradlew.bat assembleDebug
gradlew.bat assembleRelease
gradlew.bat bundleRelease
gradlew.bat installRelease
adb install -r -d android/app/build/outputs/apk/debug/app-debug.apk
adb install -r -d android/app/build/outputs/apk/release/app-release.apk
apksigner.bat verify --print-certs android/app/build/outputs/apk/debug/app-debug.apk
apksigner.bat verify --print-certs android/app/build/outputs/apk/release/app-release.apk
keytool -printcert -jarfile android/app/build/outputs/bundle/release/app-release.aab
apkanalyzer.bat manifest min-sdk app/build/outputs/apk/debug/app-debug.apk
apkanalyzer.bat manifest target-sdk app/build/outputs/apk/debug/app-debug.apk





```
