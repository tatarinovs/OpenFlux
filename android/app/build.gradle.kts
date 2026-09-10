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
        versionCode = 1
        versionName = "1.0.0"

        ndk {
            abiFilters.addAll(listOf("arm64-v8a"))
        }

        val secretKeyFile = file("../../secret_key.txt").takeIf { it.exists() }
            ?: file("../secret_key.txt").takeIf { it.exists() }
            ?: file("../../SECRET_KEY.txt").takeIf { it.exists() }
        val defaultSecretKey = secretKeyFile?.readText()?.trim() ?: ""
        buildConfigField("String", "DEFAULT_SECRET_KEY", "\"$defaultSecretKey\"")
    }

    signingConfigs {
        create("release") {
            val ks = file("../openflux-release.jks")
            if (ks.exists()) {
                storeFile = ks
                storePassword = "openflux2026"
                keyAlias = "openflux"
                keyPassword = "openflux2026"
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
            signingConfig = signingConfigs.getByName("release")
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
