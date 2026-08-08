import org.gradle.api.tasks.Exec

plugins {
    id("com.android.application")
}

val releaseVersionName = providers.gradleProperty("releaseVersionName").orElse("1.0.0")
val releaseVersionCode = providers.gradleProperty("releaseVersionCode").map(String::toInt).orElse(1)
val releaseKeystorePath = providers.environmentVariable("ANDROID_KEYSTORE_FILE").orNull

val buildTun2SocksAar by tasks.registering(Exec::class) {
    workingDir(rootProject.projectDir.resolve("enginebridge"))
    commandLine("bash", "build-aar.sh")
    inputs.files(
        rootProject.projectDir.resolve("enginebridge/go.mod"),
        rootProject.projectDir.resolve("enginebridge/go.sum"),
        rootProject.projectDir.resolve("enginebridge/mobile.go"),
        rootProject.projectDir.resolve("enginebridge/build-aar.sh"),
        rootProject.projectDir.parentFile.resolve("go.mod"),
        rootProject.projectDir.parentFile.resolve("go.sum"),
    )
    inputs.files(fileTree(rootProject.projectDir.parentFile.resolve("engine")) {
        include("**/*.go")
        exclude("**/*_test.go")
    })
    outputs.file(projectDir.resolve("libs/tun2socks.aar"))
}

tasks.named("preBuild").configure {
    dependsOn(buildTun2SocksAar)
}

android {
    namespace = "com.santaklouse.gosocks2vpn"
    compileSdk = 36

    signingConfigs {
        if (releaseKeystorePath != null) {
            create("release") {
                storeFile = file(releaseKeystorePath)
                storePassword = providers.environmentVariable("ANDROID_KEYSTORE_PASSWORD").orNull
                keyAlias = providers.environmentVariable("ANDROID_KEY_ALIAS").orNull
                keyPassword = providers.environmentVariable("ANDROID_KEY_PASSWORD").orNull
            }
        }
    }

    defaultConfig {
        applicationId = "com.santaklouse.gosocks2vpn"
        minSdk = 24
        targetSdk = 36
        versionCode = releaseVersionCode.get()
        versionName = releaseVersionName.get()
    }

    buildTypes {
        release {
            if (releaseKeystorePath != null) {
                signingConfig = signingConfigs.getByName("release")
            }
            isMinifyEnabled = true
            isShrinkResources = true
            proguardFiles(
                getDefaultProguardFile("proguard-android-optimize.txt"),
                "proguard-rules.pro",
            )
        }
    }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }
}

dependencies {
    implementation(files("libs/tun2socks.aar"))
}
