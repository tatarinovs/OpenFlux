import java.util.Properties

plugins {
    id("com.android.application")
    id("org.jetbrains.kotlin.android")
}

android {
    namespace = "com.openflux.client"
    compileSdk = 35

    defaultConfig {
        applicationId = "com.openflux.client"
        minSdk = 24
        targetSdk = 35
        versionCode = 2
        versionName = "1.0.1"

        ndk {
            abiFilters.addAll(listOf("arm64-v8a", "armeabi-v7a", "x86_64", "x86"))
        }

        val secretKeyFile = file("../../secret_key.txt").takeIf { it.exists() }
            ?: file("../secret_key.txt").takeIf { it.exists() }
            ?: file("../../SECRET_KEY.txt").takeIf { it.exists() }
        val defaultSecretKey = secretKeyFile?.readText()?.trim() ?: ""
        buildConfigField("String", "DEFAULT_SECRET_KEY", "\"$defaultSecretKey\"")
    }

    splits {
        abi {
            isEnable = true
            reset()
            include("arm64-v8a", "armeabi-v7a", "x86_64", "x86")
            isUniversalApk = true
        }
    }

    val localProps = Properties().apply {
        val f = rootProject.file("local.properties")
        if (f.exists()) {
            f.inputStream().use { load(it) }
        }
    }

    fun getProp(name: String, envName: String, fallback: String = ""): String {
        return System.getenv(envName)
            ?: localProps.getProperty(name)
            ?: fallback
    }

    val keystorePath = getProp("openflux.keystore.path", "OPENFLUX_KEYSTORE_PATH", "../openflux-release.jks")
    val ksFile = file(keystorePath)
    val storePass = getProp("openflux.keystore.password", "OPENFLUX_KEYSTORE_PASSWORD", "")
    val keyAliasName = getProp("openflux.key.alias", "OPENFLUX_KEY_ALIAS", "openflux")
    val keyPass = getProp("openflux.key.password", "OPENFLUX_KEY_PASSWORD", storePass)

    signingConfigs {
        create("release") {
            if (ksFile.exists() && storePass.isNotEmpty()) {
                storeFile = ksFile
                storePassword = storePass
                keyAlias = keyAliasName
                keyPassword = keyPass
            }
        }
    }

    buildTypes {
        release {
            isMinifyEnabled = true
            isShrinkResources = true
            proguardFiles(
                getDefaultProguardFile("proguard-android-optimize.txt"),
                "proguard-rules.pro"
            )
            if (ksFile.exists() && storePass.isNotEmpty()) {
                signingConfig = signingConfigs.getByName("release")
            }
        }
        debug {
            isMinifyEnabled = false
        }
    }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }

    kotlinOptions {
        jvmTarget = "17"
    }

    buildFeatures {
        viewBinding = true
        buildConfig = true
    }

    sourceSets {
        getByName("main") {
            jniLibs.srcDirs("src/main/jniLibs")
        }
    }
}

dependencies {
    implementation("androidx.core:core-ktx:1.15.0")
    implementation("androidx.appcompat:appcompat:1.7.0")
    implementation("com.google.android.material:material:1.12.0")
    implementation("androidx.constraintlayout:constraintlayout:2.2.0")
    implementation("androidx.recyclerview:recyclerview:1.3.2")
    implementation("androidx.lifecycle:lifecycle-runtime-ktx:2.8.7")
    implementation("androidx.lifecycle:lifecycle-viewmodel-ktx:2.8.7")
    implementation("org.jetbrains.kotlinx:kotlinx-coroutines-android:1.9.0")
}
